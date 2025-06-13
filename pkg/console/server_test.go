//go:build linux

package console

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test Linux-specific server functions
func TestServerLinuxSpecific(t *testing.T) {
	// Test NewLinuxPlatform
	_, err := NewLinuxPlatform()
	// This should return an error since it's not implemented yet
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "linux platform not yet available")

	// Test StartSimpleConsoleServerWithLinuxPlatform
	err = StartSimpleConsoleServerWithLinuxPlatform("unix:///tmp/test-server.sock")
	// This should fail since NewLinuxPlatform returns an error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create Linux platform")
}

// Test StartSimpleConsoleServer function
func TestStartSimpleConsoleServerLinux(t *testing.T) {
	mockPlatform := NewMockPlatform()
	defer mockPlatform.Close()

	// We can't easily test the full server start without blocking,
	// but we can test that StartSimpleConsoleServer function exists
	// and would fail with invalid address
	err := StartSimpleConsoleServer("invalid-address", mockPlatform)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen")
}
