package stdio

import (
	"context"
	"net"

	"github.com/mdlayher/vsock"
	"github.com/pkg/errors"
)

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

func NewVsockForwarder(
	ctx context.Context,
	ctxId uint32,
	stdinPort, stdoutPort, stderrPort uint64,
	dialFunc func(context.Context, uint32, uint64) (net.Conn, error),
	listenFunc func(context.Context, uint32, uint64) (net.Listener, error),
) (*vsockForwarder, error) {
	if ctxId == 0 {
		return nil, errors.Errorf("vsock context id is required, you likely meant to set it to '2' for the mvp")
	}

	return &vsockForwarder{
		listenFunc: func(ctx context.Context, port uint64) (net.Listener, error) {
			panic("not expected to be called for mvp")
			// return listenFunc(ctx, rvio.GetVsockContextId(), port)
		},
		dialFunc: func(ctx context.Context, port uint64) (net.Conn, error) {
			return dialFunc(ctx, ctxId, port)
		},
		stdinPort:  stdinPort,
		stdoutPort: stdoutPort,
		stderrPort: stderrPort,
	}, nil
}

func NewVsockLibForwarder(ctx context.Context, ctxId uint32, stdinPort, stdoutPort, stderrPort uint64) (*vsockForwarder, error) {
	return NewVsockForwarder(
		ctx, ctxId, stdinPort, stdoutPort, stderrPort,
		func(ctx context.Context, ctxId uint32, port uint64) (net.Conn, error) {
			return vsock.Dial(ctxId, uint32(port), nil)
		}, func(ctx context.Context, ctxId uint32, port uint64) (net.Listener, error) {
			return vsock.ListenContextID(ctxId, uint32(port), nil)
		})
}
