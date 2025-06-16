package runtime

import (
	"context"
	"io"
	"os/exec"

	gorunc "github.com/containerd/go-runc"
)

var _ IO = &HostUnixProxyIo{}

type HostUnixProxyIo struct {
	StdinSocket  AllocatedSocket
	StdoutSocket AllocatedSocket
	StderrSocket AllocatedSocket
}

func NewHostUnixProxyIo(ctx context.Context, stdinRef, stdoutRef, stderrRef AllocatedSocket) *HostUnixProxyIo {
	return &HostUnixProxyIo{
		StdinSocket:  stdinRef,
		StdoutSocket: stdoutRef,
		StderrSocket: stderrRef,
	}
}

func (p *HostUnixProxyIo) Stdin() io.WriteCloser {
	if p.StdinSocket == nil {
		return nil
	}
	return p.StdinSocket.Conn()
}

func (p *HostUnixProxyIo) Stdout() io.ReadCloser {
	if p.StdoutSocket == nil {
		return nil
	}
	return p.StdoutSocket.Conn()
}

func (p *HostUnixProxyIo) Stderr() io.ReadCloser {
	if p.StderrSocket == nil {
		return nil
	}
	return p.StderrSocket.Conn()
}

func (p *HostUnixProxyIo) Set(stdio *exec.Cmd) {
}

func (p *HostUnixProxyIo) Close() error {
	if p.StdinSocket != nil {
		p.StdinSocket.Close()
	}
	if p.StdoutSocket != nil {
		p.StdoutSocket.Close()
	}
	if p.StderrSocket != nil {
		p.StderrSocket.Close()
	}
	return nil
}

type HostNullIo struct {
	IO
}

func NewHostNullIo() (*HostNullIo, error) {
	io, err := gorunc.NewNullIO()
	if err != nil {
		return nil, err
	}
	return &HostNullIo{
		IO: io,
	}, nil
}
