package epoll

import "github.com/containerd/console"

var _ console.Console = (*EpollConsole)(nil)

type EpollConsole struct {
	Console console.Console
}

//////////////////////////////////////////////////////////////////
// these are the only ones called by runc-v2

func (e *EpollConsole) Shutdown(close func(int) error) error {
	panic("unimplemented")
}

// Write implements console.Console.
func (e *EpollConsole) Write(p []byte) (n int, err error) {
	panic("unimplemented")
}

// Read implements console.Console.
func (e *EpollConsole) Read(p []byte) (n int, err error) {
	panic("unimplemented")
}

//////////////////////////////////////////////////////////////////

// Close implements console.Console.
func (e *EpollConsole) Close() error {
	panic("unimplemented")
}

// DisableEcho implements console.Console.
func (e *EpollConsole) DisableEcho() error {
	panic("unimplemented")
}

// Fd implements console.Console.
func (e *EpollConsole) Fd() uintptr {
	panic("unimplemented")
}

// Name implements console.Console.
func (e *EpollConsole) Name() string {
	panic("unimplemented")
}

// Reset implements console.Console.
func (e *EpollConsole) Reset() error {
	panic("unimplemented")
}

// Resize implements console.Console.
func (e *EpollConsole) Resize(console.WinSize) error {
	panic("unimplemented")
}

// ResizeFrom implements console.Console.
func (e *EpollConsole) ResizeFrom(console.Console) error {
	panic("unimplemented")
}

// SetRaw implements console.Console.
func (e *EpollConsole) SetRaw() error {
	panic("unimplemented")
}

// Size implements console.Console.
func (e *EpollConsole) Size() (console.WinSize, error) {
	panic("unimplemented")
}
