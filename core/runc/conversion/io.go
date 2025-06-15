package conversion

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	gorunc "github.com/containerd/go-runc"
	"github.com/mdlayher/vsock"
	"github.com/walteh/runv/core/runc/runtime"
	runvv1 "github.com/walteh/runv/proto/v1"
)

var clientVsockCounter atomic.Uint64
var closerRefs map[io.Closer]uint64
var closerRefsMutex sync.Mutex

func init() {
	clientVsockCounter.Store(3000)
}

func ConvertIOIn(ctx context.Context, req *runvv1.RuncIO) (runtime.IO, error) {
	return NewServerIOFromClient(ctx, req)
}

func ConvertIOOut(ctx context.Context, cio runtime.IO) (*runvv1.RuncIO, error) {
	proxy, err := NewClientIOFromRuntimeVsock(ctx, cio)
	if err != nil {
		return nil, err
	}
	go func() {
		<-ctx.Done()
		proxy.Close()
	}()
	return proxy.req, nil
}

var _ runtime.IO = (*ProxyIO)(nil)

type ProxyIO struct {
	req       *runvv1.RuncIO
	pipe      gorunc.IO
	listeners []net.Listener
	dialers   []net.Conn
}

func NewClientIOFromRuntimeVsock(ctx context.Context, rvio runtime.IO) (*ProxyIO, error) {
	rviod := &runvv1.RuncVsockIO{}
	var stdinRef, stdoutRef, stderrRef uint64

	closerRefsMutex.Lock()
	if closer, ok := closerRefs[rvio.Stdin()]; ok {
		stdinRef = closer
	} else {
		closerRefs[rvio.Stdin()] = clientVsockCounter.Add(1)
		stdinRef = clientVsockCounter.Load()
	}
	if closer, ok := closerRefs[rvio.Stdout()]; ok {
		stdoutRef = closer
	} else {
		closerRefs[rvio.Stdout()] = clientVsockCounter.Add(1)
		stdoutRef = clientVsockCounter.Load()
	}
	if closer, ok := closerRefs[rvio.Stderr()]; ok {
		stderrRef = closer
	} else {
		closerRefs[rvio.Stderr()] = clientVsockCounter.Add(1)
		stderrRef = clientVsockCounter.Load()
	}
	closerRefsMutex.Unlock()

	rviod.SetStdinVsockPort(stdinRef)
	rviod.SetStdoutVsockPort(stdoutRef)
	rviod.SetStderrVsockPort(stderrRef)

	forwarder, err := NewVsockLibForwarder(ctx, rviod)
	if err != nil {
		return nil, err
	}

	stdinConn, stdoutConn, stderrConn, err := ForwardDialers(ctx, forwarder, rvio.Stdin(), rvio.Stdout(), rvio.Stderr())
	if err != nil {
		return nil, err
	}

	self := &ProxyIO{
		pipe: rvio,
	}

	self.dialers = []net.Conn{
		stdinConn,
		stdoutConn,
		stderrConn,
	}

	self.req = &runvv1.RuncIO{}
	self.req.SetVsock(rviod)

	return self, nil
}

func NewServerIOFromClient(ctx context.Context, rvio *runvv1.RuncIO) (*ProxyIO, error) {
	opts := []gorunc.IOOpt{
		func(o *gorunc.IOOption) {
			switch rvio.WhichIo() {
			case runvv1.RuncIO_Vsock_case:
				o.OpenStdin = rvio.GetVsock().GetStdinVsockPort() != 0
				o.OpenStdout = rvio.GetVsock().GetStdoutVsockPort() != 0
				o.OpenStderr = rvio.GetVsock().GetStderrVsockPort() != 0
			case runvv1.RuncIO_Unix_case:
				o.OpenStdin = rvio.GetUnix().GetStdinPath() != ""
				o.OpenStdout = rvio.GetUnix().GetStdoutPath() != ""
				o.OpenStderr = rvio.GetUnix().GetStderrPath() != ""
			}
		},
	}

	// uid := int(rvio.GetUid())
	// gid := int(rvio.GetGid())

	// if uid == -1 {
	// 	uid = os.Getuid()
	// }
	// if gid == -1 {
	// 	gid = os.Getgid()
	// }

	p, err := gorunc.NewPipeIO(os.Getuid(), os.Getgid(), opts...)
	if err != nil {
		return nil, err
	}

	var forwarder listenForwarder
	switch rvio.WhichIo() {
	case runvv1.RuncIO_Vsock_case:
		forwarder, err = NewVsockLibForwarder(ctx, rvio.GetVsock())
	case runvv1.RuncIO_Unix_case:
		forwarder, err = NewUnixForwarder(ctx, rvio.GetUnix())
	default:
		return nil, fmt.Errorf("unknown io type: %v", rvio.WhichIo())
	}
	if err != nil {
		return nil, err
	}

	self := &ProxyIO{
		req:  rvio,
		pipe: p,
	}

	stdinListener, stdoutListener, stderrListener, err := ForwardListeners(ctx, forwarder, p.Stdin(), p.Stdout(), p.Stderr())
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
func (s *ProxyIO) Stderr() io.ReadCloser {
	return s.pipe.Stderr()
}

// Stdin implements runtime.IO.
func (s *ProxyIO) Stdin() io.WriteCloser {
	return s.pipe.Stdin()
}

// Stdout implements runtime.IO.
func (s *ProxyIO) Stdout() io.ReadCloser {
	return s.pipe.Stdout()
}

func (s *ProxyIO) Close() error {
	go s.pipe.Close()
	for _, listener := range s.listeners {
		go listener.Close()
	}
	for _, dialer := range s.dialers {
		go dialer.Close()
	}
	return nil
}

func (s *ProxyIO) Set(cmd *exec.Cmd) {
	s.pipe.Set(cmd)
}

type dialForwarder interface {
	StdinDialer(context.Context) (net.Conn, error)
	StdoutDialer(context.Context) (net.Conn, error)
	StderrDialer(context.Context) (net.Conn, error)
}

type listenForwarder interface {
	StdinListener(context.Context) (net.Listener, error)
	StdoutListener(context.Context) (net.Listener, error)
	StderrListener(context.Context) (net.Listener, error)
}

type vsockForwarder struct {
	stdinPort  uint64
	stdoutPort uint64
	stderrPort uint64
	listenFunc func(context.Context, uint64) (net.Listener, error)
	dialFunc   func(context.Context, uint64) (net.Conn, error)
}

func (v *vsockForwarder) StderrListener(ctx context.Context) (net.Listener, error) {
	return v.listenFunc(ctx, v.stderrPort)
}

func (v *vsockForwarder) StdinListener(ctx context.Context) (net.Listener, error) {
	return v.listenFunc(ctx, v.stdinPort)
}

func (v *vsockForwarder) StdoutListener(ctx context.Context) (net.Listener, error) {
	return v.listenFunc(ctx, v.stdoutPort)
}

func (v *vsockForwarder) StderrDialer(ctx context.Context) (net.Conn, error) {
	return v.dialFunc(ctx, v.stderrPort)
}

func (v *vsockForwarder) StdinDialer(ctx context.Context) (net.Conn, error) {
	return v.dialFunc(ctx, v.stdinPort)
}

func (v *vsockForwarder) StdoutDialer(ctx context.Context) (net.Conn, error) {
	return v.dialFunc(ctx, v.stdoutPort)
}

func NewVsockLibForwarder(ctx context.Context, rvio *runvv1.RuncVsockIO) (*vsockForwarder, error) {
	if rvio.GetVsockContextId() == 0 {
		return nil, fmt.Errorf("vsock context id is required, you likely meant to set it to '3' for the mvp")
	}

	return &vsockForwarder{
		listenFunc: func(ctx context.Context, port uint64) (net.Listener, error) {
			return vsock.ListenContextID(rvio.GetVsockContextId(), uint32(port), nil)
		},
		dialFunc: func(ctx context.Context, port uint64) (net.Conn, error) {
			panic("not expected to be called for mvp")
			// return vsock.Dial(rvio.GetVsockContextId(), uint32(port), nil)
		},
		stdinPort:  rvio.GetStdinVsockPort(),
		stdoutPort: rvio.GetStdoutVsockPort(),
		stderrPort: rvio.GetStderrVsockPort(),
	}, nil
}

func NewVsockCustomForwarder(ctx context.Context, rvio *runvv1.RuncVsockIO, dialFunc func(context.Context, uint32, uint64) (net.Conn, error), listenFunc func(context.Context, uint32, uint64) (net.Listener, error)) (*vsockForwarder, error) {
	if rvio.GetVsockContextId() == 0 {
		return nil, fmt.Errorf("vsock context id is required, you likely meant to set it to '2' for the mvp")
	}

	return &vsockForwarder{
		listenFunc: func(ctx context.Context, port uint64) (net.Listener, error) {
			panic("not expected to be called for mvp")
			// return listenFunc(ctx, rvio.GetVsockContextId(), port)
		},
		dialFunc: func(ctx context.Context, port uint64) (net.Conn, error) {
			return dialFunc(ctx, rvio.GetVsockContextId(), port)
		},
		stdinPort:  rvio.GetStdinVsockPort(),
		stdoutPort: rvio.GetStdoutVsockPort(),
		stderrPort: rvio.GetStderrVsockPort(),
	}, nil
}

func forwardWriteListener(ctx context.Context, fn func(context.Context) (net.Listener, error), w io.Writer) (net.Listener, error) {
	listener, err := fn(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				buf := make([]byte, 1024)
				_, _ = io.CopyBuffer(w, conn, buf)
			}()
		}
	}()

	return listener, nil
}

func forwardReadListener(ctx context.Context, fn func(context.Context) (net.Listener, error), r io.Reader) (net.Listener, error) {
	listener, err := fn(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				buf := make([]byte, 1024)
				_, _ = io.CopyBuffer(conn, r, buf)
			}()
		}
	}()

	return listener, nil
}

func forwardWriteDialer(ctx context.Context, fn func(context.Context) (net.Conn, error), w io.Writer) (net.Conn, error) {
	conn, err := fn(ctx)
	if err != nil {
		return nil, err
	}
	go func() {
		defer conn.Close()
		buf := make([]byte, 1024)
		_, _ = io.CopyBuffer(w, conn, buf)
	}()
	return conn, nil
}

func forwardReadDialer(ctx context.Context, fn func(context.Context) (net.Conn, error), r io.Reader) (net.Conn, error) {
	conn, err := fn(ctx)
	if err != nil {
		return nil, err
	}
	go func() {
		defer conn.Close()
		buf := make([]byte, 1024)
		_, _ = io.CopyBuffer(conn, r, buf)
	}()
	return conn, nil
}

func ForwardListeners(ctx context.Context, f listenForwarder, stdin io.Writer, stdout io.Reader, stderr io.Reader) (net.Listener, net.Listener, net.Listener, error) {
	var err error
	var stdinListener, stdoutListener, stderrListener net.Listener

	if stdin != nil {
		stdinListener, err = forwardWriteListener(ctx, f.StdinListener, stdin)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if stdout != nil {
		stdoutListener, err = forwardReadListener(ctx, f.StdoutListener, stdout)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if stderr != nil {
		stderrListener, err = forwardReadListener(ctx, f.StderrListener, stderr)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return stdinListener, stdoutListener, stderrListener, nil
}

func ForwardDialers(ctx context.Context, f dialForwarder, stdin io.Writer, stdout io.Reader, stderr io.Reader) (net.Conn, net.Conn, net.Conn, error) {
	var err error
	var stdinConn, stdoutConn, stderrConn net.Conn

	if stdin != nil {
		stdinConn, err = forwardWriteDialer(ctx, f.StdinDialer, stdin)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if stdout != nil {
		stdoutConn, err = forwardReadDialer(ctx, f.StdoutDialer, stdout)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if stderr != nil {
		stderrConn, err = forwardReadDialer(ctx, f.StderrDialer, stderr)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return stdinConn, stdoutConn, stderrConn, nil
}

type unixForwarder struct {
	stdinPath  string
	stdoutPath string
	stderrPath string
	listenFunc func(context.Context, string) (net.Listener, error)
	dialFunc   func(context.Context, string) (net.Conn, error)
}

func NewUnixForwarder(ctx context.Context, rvio *runvv1.RuncUnixIO) (*unixForwarder, error) {
	return &unixForwarder{
		stdinPath:  rvio.GetStdinPath(),
		stdoutPath: rvio.GetStdoutPath(),
		stderrPath: rvio.GetStderrPath(),
		listenFunc: func(ctx context.Context, path string) (net.Listener, error) {
			return net.Listen("unix", path)
		},
		dialFunc: func(ctx context.Context, path string) (net.Conn, error) {
			return net.Dial("unix", path)
		},
	}, nil
}

func (u *unixForwarder) StderrListener(ctx context.Context) (net.Listener, error) {
	return u.listenFunc(ctx, u.stderrPath)
}

func (u *unixForwarder) StdinListener(ctx context.Context) (net.Listener, error) {
	return u.listenFunc(ctx, u.stdinPath)
}

func (u *unixForwarder) StdoutListener(ctx context.Context) (net.Listener, error) {
	return u.listenFunc(ctx, u.stdoutPath)
}

func (u *unixForwarder) StderrDialer(ctx context.Context) (net.Conn, error) {
	return u.dialFunc(ctx, u.stderrPath)
}

func (u *unixForwarder) StdinDialer(ctx context.Context) (net.Conn, error) {
	return u.dialFunc(ctx, u.stdinPath)
}

func (u *unixForwarder) StdoutDialer(ctx context.Context) (net.Conn, error) {
	return u.dialFunc(ctx, u.stdoutPath)
}
