package stdio

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"

	"github.com/mdlayher/vsock"

	gorunc "github.com/containerd/go-runc"
)

var hostVsockCounter atomic.Uint64
var hostUnixCounter atomic.Uint64

func init() {
	hostVsockCounter.Store(3000)
	hostUnixCounter.Store(0)
}

var _ gorunc.IO = (*HostVsockProxyIo)(nil)

type HostVsockProxyIo struct {
	StdinPort  uint64
	StdoutPort uint64
	StderrPort uint64

	StdinConn  net.Conn
	StdoutConn net.Conn
	StderrConn net.Conn

	stdinReader  io.ReadCloser
	stdinWriter  io.WriteCloser
	stdoutReader io.ReadCloser
	stdoutWriter io.WriteCloser
	stderrReader io.ReadCloser
	stderrWriter io.WriteCloser
}

func NewHostVsockProxyIo(ctx context.Context, opts ...gorunc.IOOpt) (*HostVsockProxyIo, error) {
	p := &HostVsockProxyIo{}

	optd := &gorunc.IOOption{}
	for _, opt := range opts {
		opt(optd)
	}
	if optd.OpenStdin {
		p.StdinPort = hostVsockCounter.Add(1)
		p.stdinReader, p.stdinWriter = io.Pipe()
	}
	if optd.OpenStdout {
		p.StdoutPort = hostVsockCounter.Add(1)
		p.stdoutReader, p.stdoutWriter = io.Pipe()
	}
	if optd.OpenStderr {
		p.StderrPort = hostVsockCounter.Add(1)
		p.stderrReader, p.stderrWriter = io.Pipe()
	}

	dialFunc := func(ctx context.Context, ctxId uint32, port uint64) (net.Conn, error) {
		return vsock.Dial(ctxId, uint32(port), nil)
	}
	listenFunc := func(ctx context.Context, ctxId uint32, port uint64) (net.Listener, error) {
		return vsock.ListenContextID(ctxId, uint32(port), nil)
	}

	vsockForwarder, err := NewVsockForwarder(ctx, 0, p.StdinPort, p.StdoutPort, p.StderrPort, dialFunc, listenFunc)
	if err != nil {
		return nil, err
	}

	stdinConn, stdoutConn, stderrConn, err := ForwardDialers(ctx, vsockForwarder, p.stdinReader, p.stdoutWriter, p.stderrWriter)
	if err != nil {
		return nil, err
	}

	p.StdinConn = stdinConn
	p.StdoutConn = stdoutConn
	p.StderrConn = stderrConn

	return p, nil
}

func safeBatchClose(closers ...io.Closer) {
	for _, closer := range closers {
		if closer != nil {
			go closer.Close()
		}
	}
}

func (p *HostVsockProxyIo) Close() error {
	safeBatchClose(
		p.stdinWriter,
		p.stdoutReader,
		p.stderrReader,
		p.StdinConn,
		p.StdoutConn,
		p.StderrConn,
	)
	return nil
}

func (p *HostVsockProxyIo) Stdin() io.WriteCloser {
	return p.stdinWriter
}

func (p *HostVsockProxyIo) Stdout() io.ReadCloser {
	return p.stdoutReader
}

func (p *HostVsockProxyIo) Stderr() io.ReadCloser {
	return p.stderrReader
}

func (p *HostVsockProxyIo) Set(stdio *exec.Cmd) {
}

var _ gorunc.IO = (*HostUnixProxyIo)(nil)

type HostUnixProxyIo struct {
	StdinPath  string
	StdoutPath string
	StderrPath string

	StdinConn  net.Conn
	StdoutConn net.Conn
	StderrConn net.Conn

	stdinReader  io.ReadCloser
	stdinWriter  io.WriteCloser
	stdoutReader io.ReadCloser
	stdoutWriter io.WriteCloser
	stderrReader io.ReadCloser
	stderrWriter io.WriteCloser
}

func NewHostUnixProxyIo(ctx context.Context, opts ...gorunc.IOOpt) (*HostUnixProxyIo, error) {
	p := &HostUnixProxyIo{}

	optd := &gorunc.IOOption{}
	for _, opt := range opts {
		opt(optd)
	}

	tempDir := os.TempDir()

	if optd.OpenStdin {
		p.StdinPath = filepath.Join(tempDir, fmt.Sprintf("runm-stdin-%d.sock", hostUnixCounter.Add(1)))
		p.stdinReader, p.stdinWriter = io.Pipe()
	}
	if optd.OpenStdout {
		p.StdoutPath = filepath.Join(tempDir, fmt.Sprintf("runm-stdout-%d.sock", hostUnixCounter.Add(1)))
		p.stdoutReader, p.stdoutWriter = io.Pipe()
	}
	if optd.OpenStderr {
		p.StderrPath = filepath.Join(tempDir, fmt.Sprintf("runm-stderr-%d.sock", hostUnixCounter.Add(1)))
		p.stderrReader, p.stderrWriter = io.Pipe()
	}

	unixForwarder, err := NewUnixForwarder(ctx, p.StdinPath, p.StdoutPath, p.StderrPath)
	if err != nil {
		return nil, err
	}

	stdinConn, stdoutConn, stderrConn, err := ForwardDialers(ctx, unixForwarder, p.stdinReader, p.stdoutWriter, p.stderrWriter)
	if err != nil {
		return nil, err
	}

	p.StdinConn = stdinConn
	p.StdoutConn = stdoutConn
	p.StderrConn = stderrConn

	return p, nil
}

func (p *HostUnixProxyIo) Close() error {
	safeBatchClose(
		p.stdinWriter,
		p.stdoutReader,
		p.stderrReader,
		p.StdinConn,
		p.StdoutConn,
		p.StderrConn,
	)

	// Clean up socket files
	if p.StdinPath != "" {
		os.Remove(p.StdinPath)
	}
	if p.StdoutPath != "" {
		os.Remove(p.StdoutPath)
	}
	if p.StderrPath != "" {
		os.Remove(p.StderrPath)
	}

	return nil
}

func (p *HostUnixProxyIo) Stdin() io.WriteCloser {
	return p.stdinWriter
}

func (p *HostUnixProxyIo) Stdout() io.ReadCloser {
	return p.stdoutReader
}

func (p *HostUnixProxyIo) Stderr() io.ReadCloser {
	return p.stderrReader
}

func (p *HostUnixProxyIo) Set(stdio *exec.Cmd) {
}

type HostNullIo struct {
	gorunc.IO
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
