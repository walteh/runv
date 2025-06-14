package runc

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/containerd/go-runc"
	"github.com/walteh/runv/core/runc/server"
	"google.golang.org/grpc"
)

// ServerConfig holds configuration for the runc server
type ServerConfig struct {
	RuncRoot      string
	RuncBinary    string
	Debug         bool
	PdeathSignal  syscall.Signal
	SystemdCgroup bool
	Rootless      *bool
}

// NewDefaultServerConfig returns a default server configuration
func NewDefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		RuncRoot:      "/run/runc",
		RuncBinary:    "runc",
		Debug:         false,
		PdeathSignal:  syscall.SIGKILL,
		SystemdCgroup: false,
		Rootless:      nil, // Auto-detect
	}
}

// RunServer starts a gRPC server with the RuncService
func RunServer(listener net.Listener, config *ServerConfig) (*grpc.Server, error) {
	if config == nil {
		config = NewDefaultServerConfig()
	}

	// Create runc client
	runcClient := &runc.Runc{
		Command:       config.RuncBinary,
		Root:          config.RuncRoot,
		Debug:         config.Debug,
		PdeathSignal:  config.PdeathSignal,
		SystemdCgroup: config.SystemdCgroup,
		Rootless:      config.Rootless,
	}

	// Create gRPC server
	s := grpc.NewServer()

	// Create and register our service
	runcServer := server.NewRuncServerWithOptions(runcClient)
	runcServer.Register(s)

	// Start server in a goroutine
	go func() {
		if err := s.Serve(listener); err != nil {
			// Log error but don't crash - let the caller handle this
		}
	}()

	return s, nil
}

// SetupSignalHandler sets up a signal handler for graceful shutdown
func SetupSignalHandler(srv *grpc.Server) chan struct{} {
	// Set up channel for handling signals
	sigs := make(chan os.Signal, 1)
	done := make(chan struct{})

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		srv.GracefulStop()
		close(done)
	}()

	return done
}
