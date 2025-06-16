package runtime

import (
	"context"
	"io"
	"net"
	"os"
	"os/exec"

	gorunc "github.com/containerd/go-runc"
)

type GuestIO struct {
	pipe      gorunc.IO
	listeners []net.Listener
}

func NewGuestIO(ctx context.Context, fwdr ListenForwarders, opts ...gorunc.IOOpt) (*GuestIO, error) {

	p, err := gorunc.NewPipeIO(os.Getuid(), os.Getgid(), opts...)
	if err != nil {
		return nil, err
	}

	self := &GuestIO{
		pipe:      p,
		listeners: []net.Listener{},
	}

	stdinListener, stdoutListener, stderrListener, err := ForwardListeners(ctx, fwdr, p.Stdin(), p.Stdout(), p.Stderr())
	if err != nil {
		return nil, err
	}

	if stdinListener != nil {
		self.listeners = append(self.listeners, stdinListener)
	}
	if stdoutListener != nil {
		self.listeners = append(self.listeners, stdoutListener)
	}
	if stderrListener != nil {
		self.listeners = append(self.listeners, stderrListener)
	}

	return self, nil
}

// Stderr implements runtime.IO.
func (s *GuestIO) Stderr() io.ReadCloser {
	return s.pipe.Stderr()
}

// Stdin implements runtime.IO.
func (s *GuestIO) Stdin() io.WriteCloser {
	return s.pipe.Stdin()
}

// Stdout implements runtime.IO.
func (s *GuestIO) Stdout() io.ReadCloser {
	return s.pipe.Stdout()
}

func (s *GuestIO) Close() error {
	go s.pipe.Close()
	for _, listener := range s.listeners {
		go listener.Close()
	}
	return nil
}

func (s *GuestIO) Set(cmd *exec.Cmd) {
	s.pipe.Set(cmd)
}
