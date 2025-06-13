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
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/containerd/console"
	"github.com/creack/pty"
)

// MockPlatform implements stdio.Platform for testing
type MockPlatform struct {
	mu       sync.Mutex
	consoles map[string]*MockConsole
	closed   bool
}

// MockConsole implements console.Console for testing
type MockConsole struct {
	id     string
	ptmx   *os.File
	tty    *os.File
	closed bool
	mu     sync.Mutex
}

// NewMockPlatform creates a new mock platform for testing
func NewMockPlatform() *MockPlatform {
	return &MockPlatform{
		consoles: make(map[string]*MockConsole),
	}
}

// CopyConsole implements stdio.Platform
func (p *MockPlatform) CopyConsole(ctx context.Context, console console.Console, id, stdin, stdout, stderr string, wg *sync.WaitGroup) (console.Console, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, fmt.Errorf("platform is closed")
	}

	// Create a real PTY for testing
	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open PTY: %w", err)
	}

	mockConsole := &MockConsole{
		id:   id,
		ptmx: ptmx,
		tty:  tty,
	}

	p.consoles[id] = mockConsole
	return mockConsole, nil
}

// ShutdownConsole implements stdio.Platform
func (p *MockPlatform) ShutdownConsole(ctx context.Context, cons console.Console) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	mockCons, ok := cons.(*MockConsole)
	if !ok {
		return fmt.Errorf("expected MockConsole, got %T", cons)
	}

	delete(p.consoles, mockCons.id)
	return mockCons.Close()
}

// Close implements stdio.Platform
func (p *MockPlatform) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	for _, cons := range p.consoles {
		cons.Close()
	}
	p.consoles = make(map[string]*MockConsole)

	return nil
}

// MockConsole methods

func (c *MockConsole) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, io.EOF
	}
	return c.tty.Read(p)
}

func (c *MockConsole) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, io.EOF
	}
	return c.tty.Write(p)
}

func (c *MockConsole) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	if c.tty != nil {
		c.tty.Close()
	}
	if c.ptmx != nil {
		c.ptmx.Close()
	}

	return nil
}

func (c *MockConsole) Fd() uintptr {
	if c.tty != nil {
		return c.tty.Fd()
	}
	return 0
}

func (c *MockConsole) Name() string {
	if c.tty != nil {
		return c.tty.Name()
	}
	return "mock-console"
}

func (c *MockConsole) SetRaw() error {
	return nil // Mock implementation - no-op
}

func (c *MockConsole) DisableEcho() error {
	return nil // Mock implementation - no-op
}

func (c *MockConsole) Reset() error {
	return nil // Mock implementation - no-op
}

func (c *MockConsole) Size() (console.WinSize, error) {
	return console.WinSize{Height: 24, Width: 80}, nil
}

func (c *MockConsole) Resize(ws console.WinSize) error {
	return pty.Setsize(c.ptmx, &pty.Winsize{
		Rows: ws.Height,
		Cols: ws.Width,
	})
}

func (c *MockConsole) ResizeFrom(cons console.Console) error {
	size, err := cons.Size()
	if err != nil {
		return err
	}
	return c.Resize(size)
}
