package grpcruntime

import (
	"gitlab.com/tozd/go/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/walteh/runm/core/runc/runtime"
	"github.com/walteh/runm/core/runc/state"

	runmv1 "github.com/walteh/runm/proto/v1"
)

// Client is a client for the runc service.

type GRPCClientRuntime struct {
	runtime         runmv1.RuncServiceClient
	runtimeExtras   runmv1.RuncExtrasServiceClient
	socketAllocator runmv1.SocketAllocatorServiceClient

	vsockProxier        runtime.VsockProxier
	sharedDirPathPrefix string

	conn *grpc.ClientConn

	state *state.State
}

// NewRuncClient creates a new client for the runc service.
func NewGRPCClientRuntime(target string, opts ...grpc.DialOption) (*GRPCClientRuntime, error) {
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, errors.Errorf("failed to connect to runc service: %w", err)
	}

	return NewGRPCClientRuntimeFromConn(conn)
}

// NewClientFromConn creates a new client from an existing connection.
func NewGRPCClientRuntimeFromConn(conn *grpc.ClientConn) (*GRPCClientRuntime, error) {

	client := &GRPCClientRuntime{
		runtime:         runmv1.NewRuncServiceClient(conn),
		runtimeExtras:   runmv1.NewRuncExtrasServiceClient(conn),
		socketAllocator: runmv1.NewSocketAllocatorServiceClient(conn),
		conn:            conn,
		state:           state.NewState(),
	}

	return client, nil
}

// Close closes the client connection.
func (c *GRPCClientRuntime) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
