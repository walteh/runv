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
	management      runmv1.GuestManagementServiceClient
	cgroupAdapter   runmv1.GuestCgroupServiceClient
	eventPublisher  runmv1.EventServiceClient

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
		management:      runmv1.NewGuestManagementServiceClient(conn),
		cgroupAdapter:   runmv1.NewGuestCgroupServiceClient(conn),
		eventPublisher:  runmv1.NewEventServiceClient(conn),
		conn:            conn,
		state:           state.NewState(),
	}

	return client, nil
}

func (me *GRPCClientRuntime) Management() runmv1.GuestManagementServiceClient {
	return me.management
}

func (me *GRPCClientRuntime) Runtime() runmv1.RuncServiceClient {
	return me.runtime
}

func (me *GRPCClientRuntime) RuntimeExtras() runmv1.RuncExtrasServiceClient {
	return me.runtimeExtras
}

func (me *GRPCClientRuntime) SocketAllocator() runmv1.SocketAllocatorServiceClient {
	return me.socketAllocator
}

func (me *GRPCClientRuntime) CgroupAdapter() runmv1.GuestCgroupServiceClient {
	return me.cgroupAdapter
}

func (me *GRPCClientRuntime) EventPublisher() runmv1.EventServiceClient {
	return me.eventPublisher
}

// Close closes the client connection.
func (c *GRPCClientRuntime) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
