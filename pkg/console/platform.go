/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package console

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/containerd/console"
	"github.com/containerd/containerd/v2/pkg/stdio"
	"github.com/creack/pty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	runvv1 "github.com/walteh/runv/proto/v1"
)

// SimpleProxyPlatform implements stdio.Platform using the simple approach
// Based on research: creack/pty + gRPC bidirectional streams + simple proxy pattern
type SimpleProxyPlatform struct {
	conn   *grpc.ClientConn
	client runvv1.SimpleConsoleServiceClient

	mu     sync.Mutex
	closed bool
}

// ProxyConsole represents a console that uses local PTY + remote streaming
type ProxyConsole struct {
	ptmx   *os.File // Local PTY master
	tty    *os.File // Local PTY slave
	stream grpc.BidiStreamingClient[runvv1.ConsoleChunk, runvv1.ConsoleChunk]

	mu     sync.RWMutex
	closed bool
	wg     sync.WaitGroup
}

// NewSimplePlatform creates a new simple proxy platform
func NewSimplePlatform(address string) (stdio.Platform, error) {
	// Connect to remote gRPC server
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	client := runvv1.NewSimpleConsoleServiceClient(conn)

	return &SimpleProxyPlatform{
		conn:   conn,
		client: client,
	}, nil
}

// CopyConsole implements stdio.Platform using local PTY + remote RPC
func (p *SimpleProxyPlatform) CopyConsole(ctx context.Context, console console.Console, id, stdin, stdout, stderr string, wg *sync.WaitGroup) (console.Console, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, fmt.Errorf("platform is closed")
	}

	// 2. Tell remote to set up console copying
	copyReq, err := runvv1.NewSimpleCopyConsoleRequestE(&runvv1.SimpleCopyConsoleRequest_builder{
		Id:     id,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create copy request: %w", err)
	}

	// 1. Allocate local PTY using creack/pty
	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open PTY: %w", err)
	}

	copyResp, err := p.client.CopyConsole(ctx, copyReq)
	if err != nil {
		ptmx.Close()
		tty.Close()
		return nil, fmt.Errorf("remote CopyConsole failed: %w", err)
	}

	if !copyResp.GetSuccess() {
		ptmx.Close()
		tty.Close()
		return nil, fmt.Errorf("remote error: %s", copyResp.GetError())
	}

	// 3. Start bidirectional streaming
	stream, err := p.client.StreamConsole(ctx)
	if err != nil {
		ptmx.Close()
		tty.Close()
		return nil, fmt.Errorf("failed to start stream: %w", err)
	}

	proxyConsole := &ProxyConsole{
		ptmx:   ptmx,
		tty:    tty,
		stream: stream,
	}

	// 4. Start I/O pumping goroutines
	proxyConsole.startIOPump(wg)

	// 5. Handle window resize signals
	proxyConsole.handleWindowResize()

	return proxyConsole, nil
}

// ShutdownConsole implements stdio.Platform
func (p *SimpleProxyPlatform) ShutdownConsole(ctx context.Context, cons console.Console) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("platform is closed")
	}

	proxyConsole, ok := cons.(*ProxyConsole)
	if !ok {
		return fmt.Errorf("expected ProxyConsole, got %T", cons)
	}

	// Close the proxy console
	return proxyConsole.Close()
}

// Close implements stdio.Platform
func (p *SimpleProxyPlatform) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Tell remote to close
	_, _ = p.client.ClosePlatform(ctx, runvv1.NewSimpleClosePlatformRequest(&runvv1.SimpleClosePlatformRequest_builder{}))

	// Close connection
	return p.conn.Close()
}

// ProxyConsole methods

// startIOPump starts goroutines to pump I/O between local PTY and gRPC stream
func (pc *ProxyConsole) startIOPump(wg *sync.WaitGroup) {
	pc.wg.Add(2)

	// Local PTY → gRPC stream
	go func() {
		defer pc.wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := pc.ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					// Send error to remote
					errMsg := err.Error()
					errorMsg := runvv1.NewErrorMessage(&runvv1.ErrorMessage_builder{
						Message: errMsg,
					})
					controlMsg := runvv1.NewControlMessage(&runvv1.ControlMessage_builder{
						Error: errorMsg,
					})
					chunk := runvv1.NewConsoleChunk(&runvv1.ConsoleChunk_builder{
						Control: controlMsg,
					})
					pc.stream.Send(chunk)
				}
				break
			}

			// Send data to remote
			data := buf[:n]
			chunk := runvv1.NewConsoleChunk(&runvv1.ConsoleChunk_builder{
				Data: data,
			})
			err = pc.stream.Send(chunk)
			if err != nil {
				break
			}
		}
	}()

	// gRPC stream → Local PTY
	go func() {
		defer pc.wg.Done()
		defer wg.Done() // Signal completion to caller

		for {
			chunk, err := pc.stream.Recv()
			if err != nil {
				break
			}

			// Handle incoming chunks based on which field is set
			if chunk.HasData() {
				// Write data to local PTY
				pc.ptmx.Write(chunk.GetData())
			} else if chunk.HasControl() {
				// Handle control messages
				pc.handleControlMessage(chunk.GetControl())
			}
		}
	}()
}

// handleWindowResize sets up SIGWINCH handling to resize remote PTY
func (pc *ProxyConsole) handleWindowResize() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGWINCH)

	go func() {
		for range sigs {
			rows, cols, err := pty.Getsize(pc.ptmx)
			if err != nil {
				continue
			}

			// Send resize to remote
			rowsU32 := uint32(rows)
			colsU32 := uint32(cols)

			resize, err := runvv1.NewWindowResizeE(&runvv1.WindowResize_builder{
				Rows: rowsU32,
				Cols: colsU32,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create window resize: %v\n", err)
				continue
			}

			control, err := runvv1.NewControlMessageE(&runvv1.ControlMessage_builder{
				Resize: resize,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create control message: %v\n", err)
				continue
			}

			chunk, err := runvv1.NewConsoleChunkE(&runvv1.ConsoleChunk_builder{
				Control: control,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create console chunk: %v\n", err)
				continue
			}

			pc.stream.Send(chunk)
		}
	}()
}

// handleControlMessage processes control messages from remote
func (pc *ProxyConsole) handleControlMessage(msg *runvv1.ControlMessage) {
	if msg.HasResize() {
		// Apply resize to local PTY
		resize := msg.GetResize()
		pty.Setsize(pc.ptmx, &pty.Winsize{
			Rows: uint16(resize.GetRows()),
			Cols: uint16(resize.GetCols()),
		})
	} else if msg.HasExit() {
		// Handle exit
		pc.Close()
	} else if msg.HasError() {
		// Log error
		fmt.Fprintf(os.Stderr, "Remote console error: %s\n", msg.GetError().GetMessage())
	}
}

// Console interface methods

func (pc *ProxyConsole) Read(p []byte) (int, error) {
	return pc.tty.Read(p)
}

func (pc *ProxyConsole) Write(p []byte) (int, error) {
	return pc.tty.Write(p)
}

func (pc *ProxyConsole) Close() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.closed {
		return nil
	}

	pc.closed = true

	// Close stream
	if pc.stream != nil {
		pc.stream.CloseSend()
	}

	// Close PTYs
	if pc.tty != nil {
		pc.tty.Close()
	}
	if pc.ptmx != nil {
		pc.ptmx.Close()
	}

	// Wait for goroutines
	pc.wg.Wait()

	return nil
}

func (pc *ProxyConsole) Fd() uintptr {
	return pc.tty.Fd()
}

// Implement remaining console.Console interface methods
func (pc *ProxyConsole) Name() string {
	return pc.tty.Name()
}

func (pc *ProxyConsole) SetRaw() error {
	return console.Current().SetRaw()
}

func (pc *ProxyConsole) DisableEcho() error {
	return console.Current().DisableEcho()
}

func (pc *ProxyConsole) Reset() error {
	return console.Current().Reset()
}

func (pc *ProxyConsole) Size() (console.WinSize, error) {
	return console.Current().Size()
}

func (pc *ProxyConsole) Resize(ws console.WinSize) error {
	return pty.Setsize(pc.ptmx, &pty.Winsize{
		Rows: ws.Height,
		Cols: ws.Width,
	})
}

func (pc *ProxyConsole) ResizeFrom(cons console.Console) error {
	size, err := cons.Size()
	if err != nil {
		return err
	}
	return pc.Resize(size)
}
