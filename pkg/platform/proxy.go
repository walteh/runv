//go:build !linux
// +build !linux

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

package platform

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/containerd/console"
	"github.com/containerd/containerd/v2/pkg/stdio"
	"github.com/containerd/ttrpc"
	"github.com/google/uuid"
	runvv1 "github.com/walteh/runv/proto/v1"
)

// LinuxProxyPlatformConfig holds configuration for the proxy platform
type LinuxProxyPlatformConfig struct {
	Address string
	Timeout time.Duration
}

// LinuxProxyPlatform implements stdio.Platform by proxying operations to a remote Linux server
type LinuxProxyPlatform struct {
	// ttrpc clients for different services
	platformClient  runvv1.TTRPCPlatformProxyServiceService
	epollerClient   runvv1.TTRPCEpollerServiceClient
	consoleIOClient runvv1.TTRPCConsoleIOServiceService

	conn      net.Conn
	sessionID string
	timeout   time.Duration

	mu     sync.Mutex
	closed bool
}

// ProxyConsole represents a console that proxies I/O to a remote server
type ProxyConsole struct {
	consoleID string
	sessionID string
	client    runvv1.TTRPCConsoleIOServiceService
	timeout   time.Duration

	mu     sync.RWMutex
	closed bool
}

// NewLinuxProxyPlatform creates a new proxy platform that connects to a remote Linux server
func NewLinuxProxyPlatform(address string) (stdio.Platform, error) {
	return NewLinuxProxyPlatformWithConfig(LinuxProxyPlatformConfig{
		Address: address,
		Timeout: 30 * time.Second,
	})
}

// NewLinuxProxyPlatformWithConfig creates a proxy platform with custom configuration
func NewLinuxProxyPlatformWithConfig(config LinuxProxyPlatformConfig) (stdio.Platform, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("address is required")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Connect to the remote ttrpc server
	conn, err := net.Dial("unix", config.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ttrpc server: %w", err)
	}

	// Create ttrpc client
	ttrpcClient := ttrpc.NewClient(conn)

	// Create service clients
	platformClient := runvv1.NewTTRPCPlatformProxyServiceClient(ttrpcClient)
	epollerClient := runvv1.NewTTRPCEpollerServiceClient(ttrpcClient)
	consoleIOClient := runvv1.NewTTRPCConsoleIOServiceClient(ttrpcClient)

	// Generate session ID
	sessionID := uuid.New().String()

	platform := &LinuxProxyPlatform{
		platformClient:  platformClient,
		epollerClient:   epollerClient,
		consoleIOClient: consoleIOClient,
		conn:            conn,
		sessionID:       sessionID,
		timeout:         timeout,
	}

	return platform, nil
}

// CopyConsole implements stdio.Platform.CopyConsole by proxying to the remote Linux server
func (p *LinuxProxyPlatform) CopyConsole(ctx context.Context, console console.Console, id, stdin, stdout, stderr string, wg *sync.WaitGroup) (console.Console, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, fmt.Errorf("platform is closed")
	}

	// Create a unique console ID for this proxy console
	consoleID := uuid.New().String()

	// Call the remote platform to set up the console copying
	copyReq, err := runvv1.NewCopyConsoleRequestE(&runvv1.CopyConsoleRequest_builder{
		SessionId:  p.sessionID,
		ConsoleId:  consoleID,
		ProcessId:  id,
		StdinPath:  stdin,
		StdoutPath: stdout,
		StderrPath: stderr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create copy console request: %w", err)
	}

	copyCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	copyResp, err := p.platformClient.CopyConsole(copyCtx, copyReq)
	if err != nil {
		return nil, fmt.Errorf("failed to copy console on remote server: %w", err)
	}

	if !copyResp.GetSuccess() {
		return nil, fmt.Errorf("remote server error: %s", copyResp.GetError())
	}

	// Add the console to the remote epoller
	addReq := runvv1.NewAddRequest(&runvv1.AddRequest_builder{
		SessionId: p.sessionID,
		// ConsoleId: int32(consoleID),
		Fd: int32(console.Fd()),
	})

	addCtx, addCancel := context.WithTimeout(ctx, p.timeout)
	defer addCancel()

	addResp, err := p.epollerClient.Add(addCtx, addReq)
	if err != nil {
		return nil, fmt.Errorf("failed to add console to remote epoller: %w", err)
	}

	if !addResp.GetSuccess() {
		return nil, fmt.Errorf("remote epoller error: %s", addResp.GetError())
	}

	// Create a proxy console that will handle I/O operations
	proxyConsole := &ProxyConsole{
		consoleID: copyResp.GetProxyConsoleId(),
		sessionID: p.sessionID,
		client:    p.consoleIOClient,
		timeout:   p.timeout,
	}

	return proxyConsole, nil
}

// ShutdownConsole implements stdio.Platform.ShutdownConsole
func (p *LinuxProxyPlatform) ShutdownConsole(ctx context.Context, cons console.Console) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("platform is closed")
	}

	proxyConsole, ok := cons.(*ProxyConsole)
	if !ok {
		return fmt.Errorf("expected ProxyConsole, got %T", cons)
	}

	shutdownReq := runvv1.NewShutdownConsoleRequest(&runvv1.ShutdownConsoleRequest_builder{
		SessionId:      p.sessionID,
		ProxyConsoleId: proxyConsole.consoleID,
	})

	shutdownCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	shutdownResp, err := p.platformClient.ShutdownConsole(shutdownCtx, shutdownReq)
	if err != nil {
		return fmt.Errorf("failed to shutdown console on remote server: %w", err)
	}

	if !shutdownResp.GetSuccess() {
		return fmt.Errorf("remote server error: %s", shutdownResp.GetError())
	}

	return nil
}

// Close implements stdio.Platform.Close
func (p *LinuxProxyPlatform) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	closeReq := runvv1.NewClosePlatformRequest(&runvv1.ClosePlatformRequest_builder{
		SessionId: p.sessionID,
	})

	closeResp, err := p.platformClient.ClosePlatform(ctx, closeReq)
	if err != nil {
		// Best effort - continue to close connection
	} else if !closeResp.GetSuccess() {
		// Log error but continue
	}

	// Close the connection
	if p.conn != nil {
		return p.conn.Close()
	}

	return nil
}

// ProxyConsole methods - implements console.Console interface

// Read implements io.Reader for the proxy console
func (pc *ProxyConsole) Read(p []byte) (n int, err error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if pc.closed {
		return 0, fmt.Errorf("console is closed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), pc.timeout)
	defer cancel()

	readReq := runvv1.NewConsoleReadRequest(&runvv1.ConsoleReadRequest_builder{
		SessionId:  pc.sessionID,
		ConsoleId:  pc.consoleID,
		BufferSize: int32(len(p)),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create read request: %w", err)
	}

	resp, err := pc.client.Read(ctx, readReq)
	if err != nil {
		return 0, fmt.Errorf("failed to read from remote console: %w", err)
	}

	if !resp.GetSuccess() {
		return 0, fmt.Errorf("remote console error: %s", resp.GetError())
	}

	data := resp.GetData()
	copy(p, data)

	if resp.GetEof() {
		return int(resp.GetCount()), io.EOF
	}

	return int(resp.GetCount()), nil
}

// Write implements io.Writer for the proxy console
func (pc *ProxyConsole) Write(p []byte) (n int, err error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if pc.closed {
		return 0, fmt.Errorf("console is closed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), pc.timeout)
	defer cancel()

	writeReq := runvv1.NewConsoleWriteRequest(&runvv1.ConsoleWriteRequest_builder{
		SessionId: pc.sessionID,
		ConsoleId: pc.consoleID,
		Data:      p,
	})

	resp, err := pc.client.Write(ctx, writeReq)
	if err != nil {
		return 0, fmt.Errorf("failed to write to remote console: %w", err)
	}

	if !resp.GetSuccess() {
		return 0, fmt.Errorf("remote console error: %s", resp.GetError())
	}

	return int(resp.GetCount()), nil
}

// Close implements console.Console.Close
func (pc *ProxyConsole) Close() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.closed {
		return nil
	}

	pc.closed = true
	return nil
}

// Fd implements console.Console.Fd
func (pc *ProxyConsole) Fd() uintptr {
	// Return a dummy FD since this is a proxy
	return uintptr(0)
}

// Name implements console.Console.Name
func (pc *ProxyConsole) Name() string {
	return fmt.Sprintf("proxy-console-%s", pc.consoleID)
}

// Resize implements console.Console.Resize
func (pc *ProxyConsole) Resize(ws console.WinSize) error {
	// TODO: Implement resize proxying if needed
	return nil
}

// ResizeFrom implements console.Console.ResizeFrom
func (pc *ProxyConsole) ResizeFrom(cons console.Console) error {
	// TODO: Implement resize proxying if needed
	return nil
}

// SetRaw implements console.Console.SetRaw
func (pc *ProxyConsole) SetRaw() error {
	// TODO: Implement raw mode proxying if needed
	return nil
}

// DisableEcho implements console.Console.DisableEcho
func (pc *ProxyConsole) DisableEcho() error {
	// TODO: Implement echo disable proxying if needed
	return nil
}

// Reset implements console.Console.Reset
func (pc *ProxyConsole) Reset() error {
	// TODO: Implement reset proxying if needed
	return nil
}

// Size implements console.Console.Size
func (pc *ProxyConsole) Size() (console.WinSize, error) {
	// TODO: Implement size proxying if needed
	return console.WinSize{}, nil
}
