package server

import (
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/walteh/runv/core/runc/runtime"
	"github.com/walteh/runv/core/runc/state"
)

type Server struct {
	runtime         runtime.Runtime
	runtimeExtras   runtime.RuntimeExtras
	socketAllocator runtime.SocketAllocator

	state *state.State
}

func NewServer(r runtime.Runtime, runtimeExtras runtime.RuntimeExtras, socketAllocator runtime.SocketAllocator) *Server {
	return &Server{
		runtime:         r,
		runtimeExtras:   runtimeExtras,
		socketAllocator: socketAllocator,
		state:           state.NewState(),
	}
}

// 	// Create gRPC server
// 	s := grpc.NewServer()

// 	srv := NewServer(config, nil)

// 	runvv1.RegisterRuncServiceServer(s, srv)

// 	// Start server in a goroutine
// 	go func() {
// 		if err := s.Serve(listener); err != nil {
// 			// Log error but don't crash - let the caller handle this
// 		}
// 	}()

// 	return s, nil
// }

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
