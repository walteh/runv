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

package console

import (
	"fmt"
	"net"
	"os"

	"github.com/containerd/containerd/v2/pkg/stdio"
	"google.golang.org/grpc"

	runvv1 "github.com/walteh/runv/proto/v1"
)

// NewLinuxPlatform creates a real Linux platform implementation
func NewLinuxPlatform() (stdio.Platform, error) {
	// This would import and create the real Linux platform
	// For now, return a mock or stub implementation
	return nil, fmt.Errorf("linux platform not yet available - use NewPlatform() for proxy")
}

// StartSimpleConsoleServerWithLinuxPlatform starts the server with real Linux platform
func StartSimpleConsoleServerWithLinuxPlatform(address string) error {
	platform, err := NewLinuxPlatform()
	if err != nil {
		return fmt.Errorf("failed to create Linux platform: %w", err)
	}

	server := NewSimpleConsoleServer(platform)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	runvv1.RegisterSimpleConsoleServiceServer(grpcServer, server)

	// Clean up existing socket
	if _, err := os.Stat(address); err == nil {
		os.Remove(address)
	}

	// Listen on unix socket
	listener, err := net.Listen("unix", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}
	defer listener.Close()

	// Start server
	return grpcServer.Serve(listener)
}
