package runtime

import (
	"context"
	"io"
	"net"
)

type DialForwarder interface {
	StdinDialer(context.Context) (net.Conn, error)
	StdoutDialer(context.Context) (net.Conn, error)
	StderrDialer(context.Context) (net.Conn, error)
}

type ListenForwarders interface {
	StdinListener(context.Context) (net.Listener, error)
	StdoutListener(context.Context) (net.Listener, error)
	StderrListener(context.Context) (net.Listener, error)
}

func ForwardListeners(ctx context.Context, f ListenForwarders, stdin io.Writer, stdout io.Reader, stderr io.Reader) (net.Listener, net.Listener, net.Listener, error) {
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

func ForwardDialers(ctx context.Context, f DialForwarder, stdin io.ReadCloser, stdout io.WriteCloser, stderr io.WriteCloser) (net.Conn, net.Conn, net.Conn, error) {
	var err error
	var stdinConn, stdoutConn, stderrConn net.Conn

	if stdin != nil {
		stdinConn, err = forwardReadDialer(ctx, f.StdinDialer, stdin)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if stdout != nil {
		stdoutConn, err = forwardWriteDialer(ctx, f.StdoutDialer, stdout)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if stderr != nil {
		stderrConn, err = forwardWriteDialer(ctx, f.StderrDialer, stderr)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return stdinConn, stdoutConn, stderrConn, nil
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
