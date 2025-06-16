package client

import (
	"fmt"

	"github.com/walteh/runv/core/runc/runtime"
	"github.com/walteh/runv/core/runc/state"
	runvv1 "github.com/walteh/runv/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a client for the runc service.

type Client struct {
	runtime         runvv1.RuncServiceClient
	runtimeExtras   runvv1.RuncExtrasServiceClient
	socketAllocator runvv1.SocketAllocatorServiceClient

	vsockProxier runtime.VsockProxier

	conn *grpc.ClientConn

	state *state.State
}

// NewRuncClient creates a new client for the runc service.
func NewClient(target string, opts ...grpc.DialOption) (*Client, error) {
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to runc service: %w", err)
	}

	return NewClientFromConn(conn)
}

// NewClientFromConn creates a new client from an existing connection.
func NewClientFromConn(conn *grpc.ClientConn) (*Client, error) {

	client := &Client{
		runtime:         runvv1.NewRuncServiceClient(conn),
		runtimeExtras:   runvv1.NewRuncExtrasServiceClient(conn),
		socketAllocator: runvv1.NewSocketAllocatorServiceClient(conn),
		conn:            conn,
		state:           state.NewState(),
	}

	return client, nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
