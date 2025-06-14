package runc_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/runv/core/runc/client"
	"github.com/walteh/runv/core/runc/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestBasicClientServer(t *testing.T) {
	// Set up a buffer for gRPC connections
	lis := bufconn.Listen(1024 * 1024)
	defer lis.Close()

	// Create a gRPC server
	s := grpc.NewServer()
	defer s.Stop()

	// Create and register our RuncServer service
	runcServer := server.NewRuncServer() // Using default runc configuration
	runcServer.Register(s)

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
	runcClient, err := client.NewClientFromConn(conn)
	require.NoError(t, err)
	defer runcClient.Close()

	// Test the Ping method
	err = runcClient.Ping(ctx)
	assert.NoError(t, err, "Ping should not return an error")
}
