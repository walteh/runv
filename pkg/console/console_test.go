package console

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/containerd/console"
	"github.com/containerd/containerd/v2/pkg/stdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	runvv1 "github.com/walteh/runv/proto/v1"
)

// Test SimpleProxyPlatform creation
func TestNewSimplePlatform(t *testing.T) {
	tests := []struct {
		name         string
		address      string
		expectNil    bool
		expectClosed bool
	}{
		{
			name:      "invalid address",
			address:   "invalid-address",
			expectNil: false, // gRPC client creation succeeds, connection fails later
		},
		{
			name:      "unix socket address",
			address:   "unix:///tmp/test.sock",
			expectNil: false, // gRPC client creation succeeds, connection fails later
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platform, err := NewSimplePlatform(tt.address)
			// gRPC client creation should succeed even if target doesn't exist
			assert.NoError(t, err)
			assert.NotNil(t, platform)

			if platform != nil {
				// Test that we can close the platform
				err = platform.Close()
				assert.NoError(t, err)

				// Test double close is safe
				err = platform.Close()
				assert.NoError(t, err)
			}
		})
	}
}

// Test SimpleProxyPlatform with closed state
func TestSimpleProxyPlatformClosed(t *testing.T) {
	platform, err := NewSimplePlatform("unix:///tmp/test.sock")
	require.NoError(t, err)
	require.NotNil(t, platform)

	// Close the platform
	err = platform.Close()
	require.NoError(t, err)

	ctx := context.Background()
	wg := &sync.WaitGroup{}

	// Operations on closed platform should fail
	_, err = platform.CopyConsole(ctx, nil, "test", "stdin", "stdout", "stderr", wg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform is closed")

	err = platform.ShutdownConsole(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform is closed")
}

// Test platform wrapper functions
func TestPlatformWrapper(t *testing.T) {
	// Test NewPlatform function - should succeed on non-Linux, fail on Linux
	platform, err := NewPlatform()
	if err != nil {
		// Expected on Linux or when address env var is not set correctly
		assert.Error(t, err)
		assert.Nil(t, platform)
	} else {
		// On non-Linux, client creation succeeds but connection will fail later
		assert.NotNil(t, platform)
		platform.Close()
	}

	// Test NewSimplePlatformWithAddress
	platform2, err := NewSimplePlatformWithAddress("unix:///tmp/test.sock")
	assert.NoError(t, err) // Should succeed in creating client
	assert.NotNil(t, platform2)
	if platform2 != nil {
		platform2.Close()
	}
}

// Test MockPlatform implementation
func TestMockPlatform(t *testing.T) {
	platform := NewMockPlatform()
	require.NotNil(t, platform)

	// Test interface compliance
	var _ stdio.Platform = platform

	ctx := context.Background()
	wg := &sync.WaitGroup{}

	// Test CopyConsole
	cons, err := platform.CopyConsole(ctx, nil, "test-id", "stdin", "stdout", "stderr", wg)
	require.NoError(t, err)
	require.NotNil(t, cons)

	// Test console interface compliance
	var _ console.Console = cons

	// Test console methods
	assert.NotZero(t, cons.Fd())
	assert.NotEmpty(t, cons.Name())

	size, err := cons.Size()
	require.NoError(t, err)
	assert.Equal(t, uint16(24), size.Height)
	assert.Equal(t, uint16(80), size.Width)

	// Test resize
	newSize := console.WinSize{Height: 30, Width: 100}
	err = cons.Resize(newSize)
	assert.NoError(t, err)

	// Test resize from another console
	err = cons.ResizeFrom(cons)
	assert.NoError(t, err)

	// Test mock methods that are no-ops
	assert.NoError(t, cons.SetRaw())
	assert.NoError(t, cons.DisableEcho())
	assert.NoError(t, cons.Reset())

	// Test ShutdownConsole
	err = platform.ShutdownConsole(ctx, cons)
	assert.NoError(t, err)

	// Test platform close
	err = platform.Close()
	assert.NoError(t, err)

	// Test operations on closed platform
	_, err = platform.CopyConsole(ctx, nil, "test-id-2", "stdin", "stdout", "stderr", wg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "platform is closed")

	// Test second close is harmless
	err = platform.Close()
	assert.NoError(t, err)
}

// Test MockConsole Read/Write operations
func TestMockConsoleIO(t *testing.T) {
	platform := NewMockPlatform()
	defer platform.Close()

	ctx := context.Background()
	wg := &sync.WaitGroup{}

	cons, err := platform.CopyConsole(ctx, nil, "test-io", "stdin", "stdout", "stderr", wg)
	require.NoError(t, err)
	require.NotNil(t, cons)

	// Test writing and reading
	testData := []byte("hello world")
	n, err := cons.Write(testData)
	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)

	// Close the console and test operations on closed console
	err = cons.Close()
	assert.NoError(t, err)

	// Operations on closed console should return EOF
	buf := make([]byte, 100)
	n, err = cons.Read(buf)
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)

	n, err = cons.Write(testData)
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)

	// Second close should be harmless
	err = cons.Close()
	assert.NoError(t, err)
}

// Test MockConsole edge cases
func TestMockConsoleEdgeCases(t *testing.T) {
	platform := NewMockPlatform()
	defer platform.Close()

	ctx := context.Background()
	wg := &sync.WaitGroup{}

	cons, err := platform.CopyConsole(ctx, nil, "test-edge", "stdin", "stdout", "stderr", wg)
	require.NoError(t, err)
	require.NotNil(t, cons)

	mockCons := cons.(*MockConsole)

	// Test Fd() when tty is nil
	mockCons.tty = nil
	assert.Equal(t, uintptr(0), mockCons.Fd())

	// Test Name() when tty is nil
	assert.Equal(t, "mock-console", mockCons.Name())

	// Close and test again
	err = cons.Close()
	assert.NoError(t, err)
}

// Test ResizeFrom error case
func TestMockConsoleResizeFromError(t *testing.T) {
	platform := NewMockPlatform()
	defer platform.Close()

	ctx := context.Background()
	wg := &sync.WaitGroup{}

	cons, err := platform.CopyConsole(ctx, nil, "test-resize", "stdin", "stdout", "stderr", wg)
	require.NoError(t, err)
	require.NotNil(t, cons)

	// Create a console that returns an error for Size()
	errorConsole := &ErrorConsole{}
	err = cons.ResizeFrom(errorConsole)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "size error")
}

// Test SimpleConsoleServer with mock platform
func TestSimpleConsoleServer(t *testing.T) {
	mockPlatform := NewMockPlatform()
	defer mockPlatform.Close()

	server := NewSimpleConsoleServer(mockPlatform)
	require.NotNil(t, server)

	ctx := context.Background()

	// Test CopyConsole
	req, err := runvv1.NewSimpleCopyConsoleRequestE(&runvv1.SimpleCopyConsoleRequest_builder{
		Id:     "test-console",
		Stdin:  "stdin",
		Stdout: "stdout",
		Stderr: "stderr",
	})
	require.NoError(t, err)

	resp, err := server.CopyConsole(ctx, req)
	require.NoError(t, err)
	assert.True(t, resp.GetSuccess())
	assert.Empty(t, resp.GetError())

	// Test ShutdownConsole
	shutdownReq, err := runvv1.NewSimpleShutdownConsoleRequestE(&runvv1.SimpleShutdownConsoleRequest_builder{
		Id: "test-console",
	})
	require.NoError(t, err)

	shutdownResp, err := server.ShutdownConsole(ctx, shutdownReq)
	require.NoError(t, err)
	assert.True(t, shutdownResp.GetSuccess())

	// Test shutdown of non-existent console
	shutdownReq2, err := runvv1.NewSimpleShutdownConsoleRequestE(&runvv1.SimpleShutdownConsoleRequest_builder{
		Id: "non-existent",
	})
	require.NoError(t, err)

	shutdownResp2, err := server.ShutdownConsole(ctx, shutdownReq2)
	require.NoError(t, err)
	assert.False(t, shutdownResp2.GetSuccess())
	assert.Contains(t, shutdownResp2.GetError(), "console not found")

	// Test ClosePlatform
	closeReq, err := runvv1.NewSimpleClosePlatformRequestE(&runvv1.SimpleClosePlatformRequest_builder{})
	require.NoError(t, err)

	closeResp, err := server.ClosePlatform(ctx, closeReq)
	require.NoError(t, err)
	assert.True(t, closeResp.GetSuccess())
}

// Test SimpleConsoleServer with platform errors
func TestSimpleConsoleServerErrors(t *testing.T) {
	// Create a platform that will return errors
	mockPlatform := &ErrorMockPlatform{}
	server := NewSimpleConsoleServer(mockPlatform)

	ctx := context.Background()

	// Test CopyConsole with platform error
	req, err := runvv1.NewSimpleCopyConsoleRequestE(&runvv1.SimpleCopyConsoleRequest_builder{
		Id:     "test-console",
		Stdin:  "stdin",
		Stdout: "stdout",
		Stderr: "stderr",
	})
	require.NoError(t, err)

	resp, err := server.CopyConsole(ctx, req)
	require.NoError(t, err)
	assert.False(t, resp.GetSuccess())
	assert.Contains(t, resp.GetError(), "mock platform error")

	// Test ClosePlatform with error
	closeReq, err := runvv1.NewSimpleClosePlatformRequestE(&runvv1.SimpleClosePlatformRequest_builder{})
	require.NoError(t, err)

	closeResp, err := server.ClosePlatform(ctx, closeReq)
	require.NoError(t, err)
	assert.False(t, closeResp.GetSuccess())
	assert.Contains(t, closeResp.GetError(), "mock close error")
}

// ErrorMockPlatform is a platform that always returns errors for testing
type ErrorMockPlatform struct{}

func (p *ErrorMockPlatform) CopyConsole(ctx context.Context, console console.Console, id, stdin, stdout, stderr string, wg *sync.WaitGroup) (console.Console, error) {
	return nil, errors.New("mock platform error")
}

func (p *ErrorMockPlatform) ShutdownConsole(ctx context.Context, cons console.Console) error {
	return errors.New("mock shutdown error")
}

func (p *ErrorMockPlatform) Close() error {
	return errors.New("mock close error")
}

// ErrorConsole is a console that returns errors for testing
type ErrorConsole struct{}

func (c *ErrorConsole) Read(p []byte) (int, error)  { return 0, errors.New("read error") }
func (c *ErrorConsole) Write(p []byte) (int, error) { return 0, errors.New("write error") }
func (c *ErrorConsole) Close() error                { return errors.New("close error") }
func (c *ErrorConsole) Fd() uintptr                 { return 0 }
func (c *ErrorConsole) Name() string                { return "error-console" }
func (c *ErrorConsole) SetRaw() error               { return errors.New("setraw error") }
func (c *ErrorConsole) DisableEcho() error          { return errors.New("disableecho error") }
func (c *ErrorConsole) Reset() error                { return errors.New("reset error") }
func (c *ErrorConsole) Size() (console.WinSize, error) {
	return console.WinSize{}, errors.New("size error")
}
func (c *ErrorConsole) Resize(console.WinSize) error     { return errors.New("resize error") }
func (c *ErrorConsole) ResizeFrom(console.Console) error { return errors.New("resizefrom error") }

// Test NewSimpleConsoleServer creation
func TestNewSimpleConsoleServer(t *testing.T) {
	mockPlatform := NewMockPlatform()
	defer mockPlatform.Close()

	server := NewSimpleConsoleServer(mockPlatform)
	require.NotNil(t, server)

	// Verify the server was created properly
	assert.NotNil(t, server)
}

// Test helper functions
func TestHelperFunctions(t *testing.T) {
	// Test shutdownResponse helper
	resp, err := shutdownResponse(true, "")
	require.NoError(t, err)
	assert.True(t, resp.GetSuccess())
	assert.Empty(t, resp.GetError())

	resp, err = shutdownResponse(false, "test error")
	require.NoError(t, err)
	assert.False(t, resp.GetSuccess())
	assert.Equal(t, "test error", resp.GetError())

	// Test closePlatformResponse helper
	closeResp, err := closePlatformResponse(true, "")
	require.NoError(t, err)
	assert.True(t, closeResp.GetSuccess())
	assert.Empty(t, closeResp.GetError())

	closeResp, err = closePlatformResponse(false, "test error")
	require.NoError(t, err)
	assert.False(t, closeResp.GetSuccess())
	assert.Equal(t, "test error", closeResp.GetError())
}

// Test concurrent operations
func TestConcurrentOperations(t *testing.T) {
	platform := NewMockPlatform()
	defer platform.Close()

	ctx := context.Background()
	wg := &sync.WaitGroup{}

	// Create multiple consoles concurrently
	numConsoles := 10
	consoles := make([]console.Console, numConsoles)
	errors := make([]error, numConsoles)

	var createWg sync.WaitGroup
	for i := 0; i < numConsoles; i++ {
		createWg.Add(1)
		go func(idx int) {
			defer createWg.Done()
			cons, err := platform.CopyConsole(ctx, nil, fmt.Sprintf("console-%d", idx), "stdin", "stdout", "stderr", wg)
			consoles[idx] = cons
			errors[idx] = err
		}(i)
	}

	createWg.Wait()

	// Verify all consoles were created successfully
	for i := 0; i < numConsoles; i++ {
		assert.NoError(t, errors[i])
		assert.NotNil(t, consoles[i])
	}

	// Shutdown all consoles concurrently
	var shutdownWg sync.WaitGroup
	shutdownErrors := make([]error, numConsoles)

	for i := 0; i < numConsoles; i++ {
		shutdownWg.Add(1)
		go func(idx int) {
			defer shutdownWg.Done()
			if consoles[idx] != nil {
				shutdownErrors[idx] = platform.ShutdownConsole(ctx, consoles[idx])
			}
		}(i)
	}

	shutdownWg.Wait()

	// Verify all shutdowns were successful
	for i := 0; i < numConsoles; i++ {
		assert.NoError(t, shutdownErrors[i])
	}
}

// Benchmark platform operations
func BenchmarkMockPlatformOperations(b *testing.B) {
	platform := NewMockPlatform()
	defer platform.Close()

	ctx := context.Background()
	wg := &sync.WaitGroup{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cons, err := platform.CopyConsole(ctx, nil, fmt.Sprintf("bench-console-%d", i), "stdin", "stdout", "stderr", wg)
			if err != nil {
				b.Fatal(err)
			}
			err = platform.ShutdownConsole(ctx, cons)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

// Test error cases and edge conditions
func TestErrorCases(t *testing.T) {
	// Test shutdown console with wrong type
	platform := NewMockPlatform()
	defer platform.Close()

	// Create a different console type
	err := platform.ShutdownConsole(context.Background(), &DifferentConsole{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected MockConsole")
}

// DifferentConsole is a different console implementation for testing type errors
type DifferentConsole struct{}

func (c *DifferentConsole) Read(p []byte) (int, error)       { return 0, nil }
func (c *DifferentConsole) Write(p []byte) (int, error)      { return 0, nil }
func (c *DifferentConsole) Close() error                     { return nil }
func (c *DifferentConsole) Fd() uintptr                      { return 0 }
func (c *DifferentConsole) Name() string                     { return "different" }
func (c *DifferentConsole) SetRaw() error                    { return nil }
func (c *DifferentConsole) DisableEcho() error               { return nil }
func (c *DifferentConsole) Reset() error                     { return nil }
func (c *DifferentConsole) Size() (console.WinSize, error)   { return console.WinSize{}, nil }
func (c *DifferentConsole) Resize(console.WinSize) error     { return nil }
func (c *DifferentConsole) ResizeFrom(console.Console) error { return nil }
