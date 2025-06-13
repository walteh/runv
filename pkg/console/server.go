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
	"net"
	"sync"

	"github.com/containerd/console"
	"github.com/containerd/containerd/v2/pkg/stdio"
	"google.golang.org/grpc"

	runvv1 "github.com/walteh/runv/proto/v1"
)

// SimpleConsoleServer implements the server-side simple console service for Linux
type SimpleConsoleServer struct {
	runvv1.UnimplementedSimpleConsoleServiceServer

	platform stdio.Platform // Real Linux platform

	// Track active console streams
	mu      sync.RWMutex
	streams map[string]*consoleStream
}

type consoleStream struct {
	console console.Console
	stream  grpc.BidiStreamingServer[runvv1.ConsoleChunk, runvv1.ConsoleChunk]
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewSimpleConsoleServer creates a new server
func NewSimpleConsoleServer(platform stdio.Platform) *SimpleConsoleServer {
	return &SimpleConsoleServer{
		platform: platform,
		streams:  make(map[string]*consoleStream),
	}
}

// StartSimpleConsoleServer starts the gRPC server
func StartSimpleConsoleServer(address string, platform stdio.Platform) error {
	server := NewSimpleConsoleServer(platform)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	runvv1.RegisterSimpleConsoleServiceServer(grpcServer, server)

	// Listen on unix socket
	listener, err := net.Listen("unix", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}
	defer listener.Close()

	// Start server
	return grpcServer.Serve(listener)
}

// CopyConsole implements the simple RPC method
func (s *SimpleConsoleServer) CopyConsole(ctx context.Context, req *runvv1.SimpleCopyConsoleRequest) (*runvv1.SimpleCopyConsoleResponse, error) {
	// This is much simpler - just call the real Linux platform
	// The actual PTY I/O will happen through StreamConsole
	console, err := s.platform.CopyConsole(ctx, nil, req.GetId(), req.GetStdin(), req.GetStdout(), req.GetStderr(), &sync.WaitGroup{})
	if err != nil {
		errMsg := err.Error()
		return runvv1.NewSimpleCopyConsoleResponseE(&runvv1.SimpleCopyConsoleResponse_builder{
			Success: false,
			Error:   errMsg,
		})
	}

	// Store console for streaming
	s.mu.Lock()
	s.streams[req.GetId()] = &consoleStream{
		console: console,
	}
	s.mu.Unlock()

	return runvv1.NewSimpleCopyConsoleResponseE(&runvv1.SimpleCopyConsoleResponse_builder{
		Success: true,
	})
}

func shutdownResponse(success bool, error string) (*runvv1.SimpleShutdownConsoleResponse, error) {
	return runvv1.NewSimpleShutdownConsoleResponseE(&runvv1.SimpleShutdownConsoleResponse_builder{
		Success: success,
		Error:   error,
	})
}

// ShutdownConsole implements the simple RPC method
func (s *SimpleConsoleServer) ShutdownConsole(ctx context.Context, req *runvv1.SimpleShutdownConsoleRequest) (*runvv1.SimpleShutdownConsoleResponse, error) {

	s.mu.Lock()
	stream, ok := s.streams[req.GetId()]
	if ok {
		delete(s.streams, req.GetId())
	}
	s.mu.Unlock()

	if !ok {
		errMsg := "console not found"
		return shutdownResponse(false, errMsg)
	}

	// Shutdown the real console
	err := s.platform.ShutdownConsole(ctx, stream.console)
	if err != nil {
		errMsg := err.Error()
		return shutdownResponse(false, errMsg)
	}

	return shutdownResponse(true, "")
}

func closePlatformResponse(success bool, error string) (*runvv1.SimpleClosePlatformResponse, error) {
	return runvv1.NewSimpleClosePlatformResponseE(&runvv1.SimpleClosePlatformResponse_builder{
		Success: success,
		Error:   error,
	})
}

// ClosePlatform implements the simple RPC method
func (s *SimpleConsoleServer) ClosePlatform(ctx context.Context, req *runvv1.SimpleClosePlatformRequest) (*runvv1.SimpleClosePlatformResponse, error) {
	// Close all active streams
	s.mu.Lock()
	for id, stream := range s.streams {
		if stream.cancel != nil {
			stream.cancel()
		}
		delete(s.streams, id)
	}
	s.mu.Unlock()

	// Close the real platform
	err := s.platform.Close()
	if err != nil {
		errMsg := err.Error()
		return closePlatformResponse(false, errMsg)
	}

	return closePlatformResponse(true, "")
}

// StreamConsole implements the bidirectional streaming RPC
func (s *SimpleConsoleServer) StreamConsole(stream grpc.BidiStreamingServer[runvv1.ConsoleChunk, runvv1.ConsoleChunk]) error {
	ctx := stream.Context()

	// This is where the magic happens - we multiplex console I/O over the stream
	// For now, this is a simplified implementation that handles the stream

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Handle incoming chunks
		switch chunk.WhichChunkType() {
		case runvv1.ConsoleChunk_Data_case:
			// Write data to appropriate console
			// This would need console ID to route properly
			// For now, simplified implementation

		case runvv1.ConsoleChunk_Control_case:
			// Handle control messages
			s.handleControlMessage(stream, chunk.GetControl())
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

func (s *SimpleConsoleServer) handleControlMessage(stream grpc.BidiStreamingServer[runvv1.ConsoleChunk, runvv1.ConsoleChunk], msg *runvv1.ControlMessage) {
	switch msg.WhichMessageType() {
	case runvv1.ControlMessage_Resize_case:
		// Handle window resize - forward to appropriate console

	case runvv1.ControlMessage_Env_case:
		// Handle environment variable

	case runvv1.ControlMessage_Error_case:
		// Handle error message
	}
}
