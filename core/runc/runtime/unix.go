package runtime

import (
	"context"
	"net"
)

type unixForwarder struct {
	stdinPath  string
	stdoutPath string
	stderrPath string
	listenFunc func(context.Context, string) (net.Listener, error)
	dialFunc   func(context.Context, string) (net.Conn, error)
}

func NewUnixForwarder(ctx context.Context, stdinPath, stdoutPath, stderrPath string) (*unixForwarder, error) {
	return &unixForwarder{
		stdinPath:  stdinPath,
		stdoutPath: stdoutPath,
		stderrPath: stderrPath,
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
