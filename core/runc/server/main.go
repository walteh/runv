package server

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/walteh/runv/core/runc/runtime"
	runvv1 "github.com/walteh/runv/proto/v1"
	"google.golang.org/grpc"
)

// RunServer starts a gRPC server with the RuncService
func RunServer(listener net.Listener, config runtime.Runtime) (*grpc.Server, error) {

	// Create gRPC server
	s := grpc.NewServer()

	srv := NewServer(config, nil)

	runvv1.RegisterRuncServiceServer(s, srv)

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
