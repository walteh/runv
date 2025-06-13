//go:build !linux
// +build !linux

package console

import (
	"context"
	"testing"

	"github.com/containerd/console"
	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	runvv1 "github.com/walteh/runv/proto/v1"
)

// Test ProxyConsole creation and basic operations
func TestProxyConsoleOperations(t *testing.T) {
	// Create a ProxyConsole directly for testing
	ptmx, tty, err := pty.Open()
	require.NoError(t, err)
	defer ptmx.Close()
	defer tty.Close()

	proxyConsole := &ProxyConsole{
		ptmx: ptmx,
		tty:  tty,
	}

	// Test console interface methods (don't test actual I/O to avoid hangs)
	assert.NotZero(t, proxyConsole.Fd())
	assert.NotEmpty(t, proxyConsole.Name())

	// Skip SetRaw, DisableEcho, Reset, Size tests since they delegate to console.Current()
	// which panics in test environment ("provided file is not a console")
	// These are simple delegations anyway, so testing them doesn't add much value

	// Test Resize
	err = proxyConsole.Resize(console.WinSize{Height: 25, Width: 80})
	assert.NoError(t, err)

	// Skip ResizeFrom test since it may call Size() which delegates to console.Current()
	// and can panic in test environment

	// Test Close
	err = proxyConsole.Close()
	assert.NoError(t, err)

	// Test double close
	err = proxyConsole.Close()
	assert.NoError(t, err)
}

// Test SimpleProxyPlatform ShutdownConsole with wrong console type
func TestSimpleProxyPlatformShutdownWrongType(t *testing.T) {
	platform, err := NewSimplePlatform("unix:///tmp/test.sock")
	require.NoError(t, err)
	defer platform.Close()

	// Try to shutdown with wrong console type
	wrongConsole := &DifferentConsole{}
	err = platform.ShutdownConsole(context.Background(), wrongConsole)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected ProxyConsole")
}

// Test basic ProxyConsole functionality without I/O pumping
func TestProxyConsoleInternalMethods(t *testing.T) {
	ptmx, tty, err := pty.Open()
	require.NoError(t, err)
	defer ptmx.Close()
	defer tty.Close()

	proxyConsole := &ProxyConsole{
		ptmx: ptmx,
		tty:  tty,
	}

	// Test that basic operations work
	assert.NotZero(t, proxyConsole.Fd())
	assert.NotEmpty(t, proxyConsole.Name())

	// Test Resize works
	err = proxyConsole.Resize(console.WinSize{Height: 25, Width: 80})
	assert.NoError(t, err)

	// Close the console
	err = proxyConsole.Close()
	assert.NoError(t, err)
}

// Test ProxyConsole basic functionality
func TestProxyConsoleBasics(t *testing.T) {
	ptmx, tty, err := pty.Open()
	require.NoError(t, err)
	defer ptmx.Close()
	defer tty.Close()

	proxyConsole := &ProxyConsole{
		ptmx: ptmx,
		tty:  tty,
	}

	// Test basic properties
	assert.NotZero(t, proxyConsole.Fd())
	assert.NotEmpty(t, proxyConsole.Name())

	// Test Resize
	err = proxyConsole.Resize(console.WinSize{Height: 30, Width: 100})
	assert.NoError(t, err)

	// Close the console
	err = proxyConsole.Close()
	assert.NoError(t, err)
}

// Test handleControlMessage
func TestProxyConsoleHandleControlMessage(t *testing.T) {
	ptmx, tty, err := pty.Open()
	require.NoError(t, err)
	defer ptmx.Close()
	defer tty.Close()

	proxyConsole := &ProxyConsole{
		ptmx: ptmx,
		tty:  tty,
	}

	// Test resize control message
	resizeMsg := runvv1.NewControlMessage(&runvv1.ControlMessage_builder{
		Resize: runvv1.NewWindowResize(&runvv1.WindowResize_builder{
			Rows: 30,
			Cols: 100,
		}),
	})

	proxyConsole.handleControlMessage(resizeMsg)

	// Test exit control message
	exitMsg := runvv1.NewControlMessage(&runvv1.ControlMessage_builder{
		Exit: runvv1.NewExitStatus(&runvv1.ExitStatus_builder{
			Code: 0,
		}),
	})

	proxyConsole.handleControlMessage(exitMsg)
	// This should close the console

	// Test error control message
	errorMsg := runvv1.NewControlMessage(&runvv1.ControlMessage_builder{
		Error: runvv1.NewErrorMessage(&runvv1.ErrorMessage_builder{
			Message: "test error",
		}),
	})

	proxyConsole.handleControlMessage(errorMsg)
	// This should print to stderr, but we can't easily capture it
}

// Test empty control message
func TestProxyConsoleEmptyControlMessage(t *testing.T) {
	ptmx, tty, err := pty.Open()
	require.NoError(t, err)
	defer ptmx.Close()
	defer tty.Close()

	proxyConsole := &ProxyConsole{
		ptmx: ptmx,
		tty:  tty,
	}

	// Test with empty control message (should not panic)
	emptyMsg := runvv1.NewControlMessage(&runvv1.ControlMessage_builder{})
	proxyConsole.handleControlMessage(emptyMsg)

	err = proxyConsole.Close()
	assert.NoError(t, err)
}
