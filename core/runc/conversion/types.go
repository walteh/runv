package conversion

import (
	"io"
	"os/exec"

	gorunc "github.com/containerd/go-runc"
)

var _ gorunc.ConsoleSocket = &PassThroughConsoleSocket{}

type PassThroughConsoleSocket struct {
	path string
}

func (s *PassThroughConsoleSocket) Path() string {
	return s.path
}

func NewPassThroughConsoleSocket(path string) *PassThroughConsoleSocket {
	return &PassThroughConsoleSocket{path: path}
}

var _ gorunc.IO = &PassThroughIO{}

type PassThroughIO struct {
}

// Close implements runc.IO.
func (s *PassThroughIO) Close() error {
	panic("unimplemented")
}

// Set implements runc.IO.
func (s *PassThroughIO) Set(*exec.Cmd) {
	panic("unimplemented")
}

// Stderr implements runc.IO.
func (s *PassThroughIO) Stderr() io.ReadCloser {
	panic("unimplemented")
}

// Stdin implements runc.IO.
func (s *PassThroughIO) Stdin() io.WriteCloser {
	panic("unimplemented")
}

// Stdout implements runc.IO.
func (s *PassThroughIO) Stdout() io.ReadCloser {
	panic("unimplemented")
}
