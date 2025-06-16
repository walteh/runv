package runc_test

import (
	"context"
	"errors"
	"net"
	"testing"

	gorunc "github.com/containerd/go-runc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpcruntime "github.com/walteh/runv/core/runc/runtime/grpc"
	"github.com/walteh/runv/core/runc/server"
	runtimemock "github.com/walteh/runv/gen/mocks/core/runc/runtime"
	runvv1 "github.com/walteh/runv/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestBasicClientServer(t *testing.T) {

	lis := bufconn.Listen(1024 * 1024)
	defer lis.Close()

	// Create a gRPC server
	s := grpc.NewServer()
	defer s.Stop()

	testErr := errors.New("test error")

	mockRuntime := &runtimemock.MockRuntime{
		CreateFunc: func(ctx context.Context, id, bundle string, opts *gorunc.CreateOpts) error {
			return testErr
		},
	}

	// Create and register our RuncServer service
	runcServer := server.NewServer(mockRuntime, nil) // Using default runc configuration
	runvv1.RegisterRuncServiceServer(s, runcServer)

	// Start the server
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()

	// Create a client connection
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	require.NoError(t, err)
	defer conn.Close()

	// Create a client
	runcClient, err := grpcruntime.NewGRPCClientRuntimeFromConn(conn)
	require.NoError(t, err)
	defer runcClient.Close()

	// Test the Ping method
	err = runcClient.Create(ctx, "test", "test", &gorunc.CreateOpts{})
	assert.ErrorContains(t, err, testErr.Error(), "Ping should return an error")
}
