package conversion

import (
	"context"
	"fmt"
	"net"

	gorunc "github.com/containerd/go-runc"
	"github.com/mdlayher/vsock"
	"github.com/walteh/runv/core/runc/runtime"
	"github.com/walteh/runv/core/runc/stdio"
	runvv1 "github.com/walteh/runv/proto/v1"
)

func ConvertIOFromProto(ctx context.Context, req *runvv1.RuncIO) (runtime.IO, error) {
	return NewServerIOFromClient(ctx, req)
}

func ConvertIOToProto(ctx context.Context, rvio runtime.IO) (*runvv1.RuncIO, error) {
	res := &runvv1.RuncIO{}

	switch iot := rvio.(type) {
	case *stdio.HostVsockProxyIo:
		rviod := &runvv1.RuncVsockIO{}
		rviod.SetStdinVsockPort(iot.StdinPort)
		rviod.SetStdoutVsockPort(iot.StdoutPort)
		rviod.SetStderrVsockPort(iot.StderrPort)
		rviod.SetVsockContextId(3)
		res.SetVsock(rviod)
	case *stdio.HostUnixProxyIo:
		rviod := &runvv1.RuncUnixIO{}
		rviod.SetStdinPath(iot.StdinPath)
		rviod.SetStdoutPath(iot.StdoutPath)
		rviod.SetStderrPath(iot.StderrPath)
		res.SetUnix(rviod)
	case *stdio.HostNullIo:
		res.SetNull(&runvv1.RuncNullIO{})
	default:
		return nil, fmt.Errorf("io is not a vsock proxy io")
	}

	return res, nil
}

func NewServerIOFromClient(ctx context.Context, rvio *runvv1.RuncIO) (runtime.IO, error) {
	var err error
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
	var forwarder stdio.ListenForwarders

	switch rvio.WhichIo() {
	case runvv1.RuncIO_Vsock_case:
		ctxId := rvio.GetVsock().GetVsockContextId()
		stdinPort := rvio.GetVsock().GetStdinVsockPort()
		stdoutPort := rvio.GetVsock().GetStdoutVsockPort()
		stderrPort := rvio.GetVsock().GetStderrVsockPort()
		dialFunc := func(_ context.Context, ctxId uint32, port uint64) (net.Conn, error) {
			return vsock.Dial(ctxId, uint32(port), nil)
		}
		listenFunc := func(_ context.Context, ctxId uint32, port uint64) (net.Listener, error) {
			return vsock.ListenContextID(ctxId, uint32(port), nil)
		}
		forwarder, err = stdio.NewVsockForwarder(ctx, ctxId, stdinPort, stdoutPort, stderrPort, dialFunc, listenFunc)
	case runvv1.RuncIO_Unix_case:
		stdinPath := rvio.GetUnix().GetStdinPath()
		stdoutPath := rvio.GetUnix().GetStdoutPath()
		stderrPath := rvio.GetUnix().GetStderrPath()
		forwarder, err = stdio.NewUnixForwarder(ctx, stdinPath, stdoutPath, stderrPath)
	case runvv1.RuncIO_Null_case:
		// no need to forward null io
		return gorunc.NewNullIO()
	default:
		return nil, fmt.Errorf("unknown io type: %v", rvio.WhichIo())
	}
	if err != nil {
		return nil, err
	}

	guestIO, err := stdio.NewGuestIO(ctx, forwarder, opts...)
	if err != nil {
		return nil, err
	}

	return guestIO, nil
}
