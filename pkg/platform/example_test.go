//go:build !linux

package platform

import (
	"testing"
	"time"

	"github.com/containerd/containerd/v2/pkg/stdio"
)

// Example showing how to use the proxy platform on non-Linux systems
func ExampleNewLinuxProxyPlatform() {
	// This would run on macOS/non-Linux client

	// Connect to remote Linux server
	platform, err := NewLinuxProxyPlatform("unix:///var/run/runv-console.sock")
	if err != nil {
		panic(err)
	}
	defer platform.Close()

	// Use the platform just like the Linux version
	// console, err := platform.CopyConsole(ctx, localConsole, "process-id", stdin, stdout, stderr, wg)
	// if err != nil {
	//     panic(err)
	// }
	//
	// // Work with the console...
	//
	// // Shutdown when done
	// err = platform.ShutdownConsole(ctx, console)
	// if err != nil {
	//     panic(err)
	// }
}

// Test the proxy platform creation
func TestNewLinuxProxyPlatform(t *testing.T) {
	// Test with invalid address
	_, err := NewLinuxProxyPlatform("")
	if err == nil {
		t.Error("Expected error for empty address")
	}

	// Test with valid address (will fail to connect but that's expected)
	_, err = NewLinuxProxyPlatformWithConfig(LinuxProxyPlatformConfig{
		Address: "/tmp/test.sock",
		Timeout: 1 * time.Second,
	})
	// We expect this to fail since there's no server running
	if err == nil {
		t.Error("Expected connection error")
	}
}

// Test the platform proxy interface
func TestPlatformProxy(t *testing.T) {
	// Test that NewPlatform returns a valid platform interface
	_, err := NewPlatformWithProxy("/tmp/test.sock")
	if err == nil {
		t.Error("Expected connection error")
	}

	// Verify it implements stdio.Platform interface
	var _ stdio.Platform = (*LinuxProxyPlatform)(nil)
}
