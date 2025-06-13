//go:build linux

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
	"os"
	"sync"
	"time"

	"github.com/containerd/console"
	"github.com/containerd/containerd/v2/pkg/stdio"
	"github.com/containerd/ttrpc"
	"github.com/google/uuid"
	"go.uber.org/zap"

	runvv1 "github.com/walteh/runv/proto/v1"
)

// ConsoleProxyServer implements the console proxy services for Linux systems
type ConsoleProxyServer struct {
	platform     stdio.Platform
	sessions     map[string]*proxySession
	sessionMutex sync.RWMutex
	logger       *zap.Logger
}

type proxySession struct {
	sessionID string
	platform  stdio.Platform
	consoles  map[string]*proxyConsole
	epoll     *console.Epoller
	mutex     sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

type proxyConsole struct {
	consoleID      string
	console        console.Console
	processID      string
	proxyConsoleID string
}

// NewConsoleProxyServer creates a new console proxy server
func NewConsoleProxyServer(platform stdio.Platform, logger *zap.Logger) *ConsoleProxyServer {
	return &ConsoleProxyServer{
		platform: platform,
		sessions: make(map[string]*proxySession),
		logger:   logger,
	}
}

// StartConsoleProxyServer starts the ttrpc server on the specified address
func StartConsoleProxyServer(address string, platform stdio.Platform) error {
	server := NewConsoleProxyServer(platform, nil)

	// Create ttrpc server
	ttrpcServer, err := ttrpc.NewServer()
	if err != nil {
		return fmt.Errorf("failed to create ttrpc server: %w", err)
	}

	// Register services
	runvv1.RegisterTTRPCPlatformProxyServiceService(ttrpcServer, server)
	runvv1.RegisterTTRPCEpollerServiceService(ttrpcServer, server)
	runvv1.RegisterTTRPCConsoleIOServiceService(ttrpcServer, server)

	// Listen on unix socket
	listener, err := net.Listen("unix", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}
	defer listener.Close()

	// Start server
	return ttrpcServer.Serve(context.Background(), listener)
}

// Platform service methods

// CopyConsole implements the PlatformProxyService.CopyConsole method
func (s *ConsoleProxyServer) CopyConsole(ctx context.Context, req *runvv1.CopyConsoleRequest) (*runvv1.CopyConsoleResponse, error) {
	sessionID := req.GetSessionId()
	consoleID := req.GetConsoleId()
	processID := req.GetProcessId()
	stdinPath := req.GetStdinPath()
	stdoutPath := req.GetStdoutPath()
	stderrPath := req.GetStderrPath()

	session := s.createSession(sessionID)
	if session == nil {
		return runvv1.NewCopyConsoleResponseE(func(b *runvv1.CopyConsoleResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"failed to create session"}[0]
		})
	}

	// Create a dummy console - in real implementation this would come from the container process
	// For now, we'll use stdin/stdout/stderr files to create a console-like interface
	var mockConsole console.Console
	if stdinPath != "" {
		// Try to open the stdin file as a console
		file, err := os.OpenFile(stdinPath, os.O_RDWR, 0)
		if err == nil {
			if cons, err := console.ConsoleFromFile(file); err == nil {
				mockConsole = cons
			} else {
				file.Close()
			}
		}
	}

	if mockConsole == nil {
		// If we can't create a real console, create a minimal mock
		return runvv1.NewCopyConsoleResponseE(func(b *runvv1.CopyConsoleResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"failed to create console from paths"}[0]
		})
	}

	// Create console using the platform (this will need the console, not separate paths)
	var wg sync.WaitGroup
	platformConsole, err := session.platform.CopyConsole(ctx, mockConsole, processID, stdinPath, stdoutPath, stderrPath, &wg)
	if err != nil {
		return runvv1.NewCopyConsoleResponseE(func(b *runvv1.CopyConsoleResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{fmt.Sprintf("failed to copy console: %v", err)}[0]
		})
	}

	// Generate proxy console ID
	proxyConsoleID := uuid.New().String()

	// Store console in session
	session.mutex.Lock()
	session.consoles[consoleID] = &proxyConsole{
		consoleID:      consoleID,
		console:        platformConsole,
		processID:      processID,
		proxyConsoleID: proxyConsoleID,
	}
	session.mutex.Unlock()

	return runvv1.NewCopyConsoleResponseE(func(b *runvv1.CopyConsoleResponse_builder) {
		b.Success = &[]bool{true}[0]
		b.ProxyConsoleId = &proxyConsoleID
		b.ProxyAddress = &[]string{""}[0] // Could be set if needed
	})
}

// ShutdownConsole implements the PlatformProxyService.ShutdownConsole method
func (s *ConsoleProxyServer) ShutdownConsole(ctx context.Context, req *runvv1.ShutdownConsoleRequest) (*runvv1.ShutdownConsoleResponse, error) {
	sessionID := req.GetSessionId()
	proxyConsoleID := req.GetProxyConsoleId()

	session, exists := s.getSession(sessionID)
	if !exists {
		return runvv1.NewShutdownConsoleResponseE(func(b *runvv1.ShutdownConsoleResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"session not found"}[0]
		})
	}

	// Find console by proxy console ID
	session.mutex.Lock()
	var targetConsole *proxyConsole
	var targetConsoleID string
	for consoleID, console := range session.consoles {
		if console.proxyConsoleID == proxyConsoleID {
			targetConsole = console
			targetConsoleID = consoleID
			break
		}
	}

	if targetConsole == nil {
		session.mutex.Unlock()
		return runvv1.NewShutdownConsoleResponseE(func(b *runvv1.ShutdownConsoleResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"console not found"}[0]
		})
	}

	// Shutdown console using the platform interface
	err := session.platform.ShutdownConsole(ctx, targetConsole.console)
	if err != nil {
		session.mutex.Unlock()
		return runvv1.NewShutdownConsoleResponseE(func(b *runvv1.ShutdownConsoleResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{fmt.Sprintf("failed to shutdown console: %v", err)}[0]
		})
	}

	// Remove from session
	delete(session.consoles, targetConsoleID)
	session.mutex.Unlock()

	return runvv1.NewShutdownConsoleResponseE(func(b *runvv1.ShutdownConsoleResponse_builder) {
		b.Success = &[]bool{true}[0]
	})
}

// ClosePlatform implements the PlatformProxyService.ClosePlatform method
func (s *ConsoleProxyServer) ClosePlatform(ctx context.Context, req *runvv1.ClosePlatformRequest) (*runvv1.ClosePlatformResponse, error) {
	sessionID := req.GetSessionId()

	session, exists := s.getSession(sessionID)
	if !exists {
		return runvv1.NewClosePlatformResponseE(func(b *runvv1.ClosePlatformResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"session not found"}[0]
		})
	}

	// Close platform for session
	err := session.platform.Close()
	if err != nil {
		return runvv1.NewClosePlatformResponseE(func(b *runvv1.ClosePlatformResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{fmt.Sprintf("failed to close platform: %v", err)}[0]
		})
	}

	// Remove session
	s.removeSession(sessionID)

	return runvv1.NewClosePlatformResponseE(func(b *runvv1.ClosePlatformResponse_builder) {
		b.Success = &[]bool{true}[0]
	})
}

// Epoll service methods

// Add implements the EpollerService.Add method
func (s *ConsoleProxyServer) Add(ctx context.Context, req *runvv1.AddRequest) (*runvv1.AddResponse, error) {
	sessionID := req.GetSessionId()
	fd := req.GetFd()

	session, exists := s.getSession(sessionID)
	if !exists {
		return runvv1.NewAddResponseE(func(b *runvv1.AddResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"session not found"}[0]
		})
	}

	// Find the console by ID to add to epoll
	session.mutex.RLock()
	var targetConsole console.Console
	for _, proxyConsole := range session.consoles {
		if int32(proxyConsole.console.Fd()) == fd {
			targetConsole = proxyConsole.console
			break
		}
	}
	session.mutex.RUnlock()

	if targetConsole == nil {
		return runvv1.NewAddResponseE(func(b *runvv1.AddResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"console not found for fd"}[0]
		})
	}

	// Add console to epoll
	_, err := session.epoll.Add(targetConsole)
	if err != nil {
		return runvv1.NewAddResponseE(func(b *runvv1.AddResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{fmt.Sprintf("failed to add to epoll: %v", err)}[0]
		})
	}

	return runvv1.NewAddResponseE(func(b *runvv1.AddResponse_builder) {
		b.Success = &[]bool{true}[0]
		b.ProxyConsoleAddress = &[]string{""}[0] // Could be set if needed
	})
}

// CloseConsole implements the EpollerService.CloseConsole method
func (s *ConsoleProxyServer) CloseConsole(ctx context.Context, req *runvv1.CloseConsoleRequest) (*runvv1.CloseConsoleResponse, error) {
	sessionID := req.GetSessionId()
	targetConsoleID := req.GetConsoleId()

	session, exists := s.getSession(sessionID)
	if !exists {
		return runvv1.NewCloseConsoleResponseE(func(b *runvv1.CloseConsoleResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"session not found"}[0]
		})
	}

	// Find console by ID
	session.mutex.RLock()
	var targetFd int
	for _, proxyConsole := range session.consoles {
		if int32(proxyConsole.console.Fd()) == targetConsoleID {
			targetFd = int(proxyConsole.console.Fd())
			break
		}
	}
	session.mutex.RUnlock()

	if targetFd == 0 {
		return runvv1.NewCloseConsoleResponseE(func(b *runvv1.CloseConsoleResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"console not found"}[0]
		})
	}

	// Close console in epoll
	err := session.epoll.CloseConsole(targetFd)
	if err != nil {
		return runvv1.NewCloseConsoleResponseE(func(b *runvv1.CloseConsoleResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{fmt.Sprintf("failed to close console in epoll: %v", err)}[0]
		})
	}

	return runvv1.NewCloseConsoleResponseE(func(b *runvv1.CloseConsoleResponse_builder) {
		b.Success = &[]bool{true}[0]
	})
}

// Wait implements the EpollerService.Wait method (streaming)
func (s *ConsoleProxyServer) Wait(ctx context.Context, req *runvv1.WaitRequest, stream runvv1.TTRPCEpollerService_WaitServer) error {
	sessionID := req.GetSessionId()

	session, exists := s.getSession(sessionID)
	if !exists {
		// Send error response
		errorEvent, _ := runvv1.NewEpollEventE(func(eb *runvv1.EpollEvent_builder) {
			eb.ConsoleId = &[]int32{-1}[0]
			eb.EventType = &[]runvv1.EpollEventType{runvv1.EpollEventType_EPOLL_EVENT_TYPE_ERROR}[0]
			eb.Error = &[]string{"session not found"}[0]
		})

		if errorEvent != nil {
			errorResp, _ := runvv1.NewWaitResponseE(func(b *runvv1.WaitResponse_builder) {
				b.SessionId = &sessionID
				b.Events = []*runvv1.EpollEvent{errorEvent}
			})
			if errorResp != nil {
				stream.Send(errorResp)
			}
		}
		return fmt.Errorf("session not found")
	}

	// Start the epoll wait loop in a goroutine
	go session.epoll.Wait()

	// For simplicity, we'll just send a periodic ping
	// In a real implementation, this would integrate with the epoll events
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-session.ctx.Done():
			return nil
		case <-ticker.C:
			// Send a simple event to show the system is working
			// In a real implementation, this would be replaced with actual epoll events
			resp, err := runvv1.NewWaitResponseE(func(b *runvv1.WaitResponse_builder) {
				b.SessionId = &sessionID
				b.Events = []*runvv1.EpollEvent{} // Empty for now
			})
			if err == nil && resp != nil {
				if err := stream.Send(resp); err != nil {
					return err
				}
			}
		}
	}
}

// Console I/O service methods

// Read implements the ConsoleIOService.Read method
func (s *ConsoleProxyServer) Read(ctx context.Context, req *runvv1.ConsoleReadRequest) (*runvv1.ConsoleReadResponse, error) {
	sessionID := req.GetSessionId()
	consoleID := req.GetConsoleId()
	bufferSize := req.GetBufferSize()

	session, exists := s.getSession(sessionID)
	if !exists {
		return runvv1.NewConsoleReadResponseE(func(b *runvv1.ConsoleReadResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"session not found"}[0]
		})
	}

	session.mutex.RLock()
	proxyConsole, exists := session.consoles[consoleID]
	session.mutex.RUnlock()

	if !exists {
		return runvv1.NewConsoleReadResponseE(func(b *runvv1.ConsoleReadResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"console not found"}[0]
		})
	}

	// Read from console
	buffer := make([]byte, bufferSize)
	n, err := proxyConsole.console.Read(buffer)

	if err != nil && err != io.EOF {
		return runvv1.NewConsoleReadResponseE(func(b *runvv1.ConsoleReadResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{fmt.Sprintf("read error: %v", err)}[0]
		})
	}

	eof := err == io.EOF
	data := buffer[:n]

	return runvv1.NewConsoleReadResponseE(func(b *runvv1.ConsoleReadResponse_builder) {
		b.Success = &[]bool{true}[0]
		b.Data = data
		b.Count = &[]int32{int32(n)}[0]
		b.Eof = &eof
	})
}

// Write implements the ConsoleIOService.Write method
func (s *ConsoleProxyServer) Write(ctx context.Context, req *runvv1.ConsoleWriteRequest) (*runvv1.ConsoleWriteResponse, error) {
	sessionID := req.GetSessionId()
	consoleID := req.GetConsoleId()
	data := req.GetData()

	session, exists := s.getSession(sessionID)
	if !exists {
		return runvv1.NewConsoleWriteResponseE(func(b *runvv1.ConsoleWriteResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"session not found"}[0]
		})
	}

	session.mutex.RLock()
	proxyConsole, exists := session.consoles[consoleID]
	session.mutex.RUnlock()

	if !exists {
		return runvv1.NewConsoleWriteResponseE(func(b *runvv1.ConsoleWriteResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{"console not found"}[0]
		})
	}

	// Write to console
	n, err := proxyConsole.console.Write(data)
	if err != nil {
		return runvv1.NewConsoleWriteResponseE(func(b *runvv1.ConsoleWriteResponse_builder) {
			b.Success = &[]bool{false}[0]
			b.Error = &[]string{fmt.Sprintf("write error: %v", err)}[0]
		})
	}

	return runvv1.NewConsoleWriteResponseE(func(b *runvv1.ConsoleWriteResponse_builder) {
		b.Success = &[]bool{true}[0]
		b.Count = &[]int32{int32(n)}[0]
	})
}

// Helper methods

func (s *ConsoleProxyServer) getSession(sessionID string) (*proxySession, bool) {
	s.sessionMutex.RLock()
	defer s.sessionMutex.RUnlock()
	session, ok := s.sessions[sessionID]
	return session, ok
}

func (s *ConsoleProxyServer) createSession(sessionID string) *proxySession {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	if session, exists := s.sessions[sessionID]; exists {
		return session
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &proxySession{
		sessionID: sessionID,
		platform:  s.platform,
		consoles:  make(map[string]*proxyConsole),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Initialize epoll for this session
	epoll, err := console.NewEpoller()
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to create epoller for session",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
		cancel()
		return nil
	}
	session.epoll = epoll

	s.sessions[sessionID] = session
	return session
}

func (s *ConsoleProxyServer) removeSession(sessionID string) {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	if session, exists := s.sessions[sessionID]; exists {
		session.cancel()

		// Close all consoles in the session
		session.mutex.Lock()
		for _, console := range session.consoles {
			if console.console != nil {
				console.console.Close()
			}
		}
		session.mutex.Unlock()

		if session.epoll != nil {
			session.epoll.Close()
		}

		delete(s.sessions, sessionID)
	}
}

// RegisterServices registers the server with a ttrpc server
func (s *ConsoleProxyServer) RegisterServices(server *ttrpc.Server) {
	runvv1.RegisterTTRPCPlatformProxyServiceService(server, s)
	runvv1.RegisterTTRPCConsoleIOServiceService(server, s)
	runvv1.RegisterTTRPCEpollerServiceService(server, s)
}
