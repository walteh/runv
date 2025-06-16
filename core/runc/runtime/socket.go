package runtime

import (
	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/mdlayher/vsock"
	"go.uber.org/atomic"
)

type FileConn interface {
	syscall.Conn
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
}

var _ AllocatedSocket = &HostAllocatedSocket{}

type HostAllocatedSocket struct {
	conn        *net.UnixConn
	path        string
	referenceId string
}

func (h *HostAllocatedSocket) isAllocatedSocket() {}

func (h *HostAllocatedSocket) Close() error {
	return h.conn.Close()
}

func (h *HostAllocatedSocket) Conn() FileConn {
	return h.conn
}

func (h *HostAllocatedSocket) Path() string {
	return h.path
}

func (h *HostAllocatedSocket) Ready() error {
	return nil
}

func NewHostAllocatedVsockSocket(ctx context.Context, port uint32, refId string, proxier VsockProxier) (*HostAllocatedSocket, error) {
	conn, path, err := proxier.Proxy(ctx, port)
	if err != nil {
		return nil, err
	}
	return &HostAllocatedSocket{conn: conn, path: path, referenceId: refId}, nil
}

func NewHostAllocatedUnixSocket(ctx context.Context, path string, refId string) (*HostAllocatedSocket, error) {
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, err
	}
	return &HostAllocatedSocket{conn: conn, path: path, referenceId: refId}, nil
}

func NewHostAllocatedSocketFromId(ctx context.Context, id string, proxier VsockProxier) (*HostAllocatedSocket, error) {
	switch {
	case strings.HasPrefix(id, "socket:vsock:"):
		port, err := strconv.Atoi(strings.TrimPrefix(id, "socket:vsock:"))
		if err != nil {
			return nil, err
		}
		return NewHostAllocatedVsockSocket(ctx, uint32(port), id, proxier)
	case strings.HasPrefix(id, "socket:unix:"):
		path := strings.TrimPrefix(id, "socket:unix:")
		return NewHostAllocatedUnixSocket(ctx, path, id)
	}
	return nil, fmt.Errorf("invalid socket type: %s", id)
}

// BindConsoleToSocket implements runtime.SocketAllocator.
func BindConsoleToSocket(ctx context.Context, cons ConsoleSocket, sock AllocatedSocket) error {
	// open up the console socket path, and create a pipe to it
	consConn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: cons.Path(), Net: "unix"})
	if err != nil {
		return err
	}
	sockConn := sock.Conn()

	wg := sync.WaitGroup{}
	wg.Add(2)

	// create a goroutine to read from the pipe and write to the socket
	go func() {
		defer wg.Done()
		wg.Add(1)
		io.Copy(consConn, sockConn)
	}()

	// create a goroutine to read from the socket and write to the console
	go func() {
		defer wg.Done()
		wg.Add(1)
		io.Copy(sockConn, consConn)
	}()

	go func() {
		wg.Wait()
		consConn.Close()
		sockConn.Close()
	}()

	// return the pipe
	return nil
}

// BindIOToSockets implements SocketAllocator.
func BindIOToSockets(ctx context.Context, ios IO, stdin, stdout, stderr AllocatedSocket) error {

	if stdin != nil {
		go func() {
			io.Copy(ios.Stdin(), stdin.Conn())
		}()
	}
	if stdout != nil {
		go func() {
			io.Copy(stdout.Conn(), ios.Stdout())
		}()
	}
	if stderr != nil {
		go func() {
			io.Copy(stderr.Conn(), ios.Stderr())
		}()
	}

	return nil
}

type GuestAllocatedUnixSocket struct {
	listener    *net.UnixListener
	conn        *net.UnixConn
	path        string
	ready       chan struct{}
	readyErr    error
	referenceId string
}

func (g *GuestAllocatedUnixSocket) isAllocatedSocket() {}

func (g *GuestAllocatedUnixSocket) Close() error {
	return g.conn.Close()
}

func (g *GuestAllocatedUnixSocket) Conn() *net.UnixConn {
	return g.conn
}

func (g *GuestAllocatedUnixSocket) Path() string {
	return g.path
}

func (g *GuestAllocatedUnixSocket) Ready() error {
	<-g.ready
	return g.readyErr
}

func NewGuestAllocatedUnixSocket(ctx context.Context, path string) (*GuestAllocatedUnixSocket, error) {
	conn, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, err
	}

	refId := NewUnixSocketReferenceId(path)

	guestConn := &GuestAllocatedUnixSocket{
		listener:    conn,
		path:        path,
		referenceId: refId,
		ready:       make(chan struct{}),
	}

	go func() {
		defer close(guestConn.ready)
		conn, err := conn.AcceptUnix()
		if err != nil {
			return
		}
		guestConn.conn = conn
	}()

	return guestConn, nil
}

type GuestAllocatedVsockSocket struct {
	listener    *vsock.Listener
	conn        *vsock.Conn
	ready       chan struct{}
	readyErr    error
	path        string
	referenceId string
}

func (g *GuestAllocatedVsockSocket) isAllocatedSocket() {}

func (g *GuestAllocatedVsockSocket) Close() error {
	return g.conn.Close()
}

func (g *GuestAllocatedVsockSocket) Conn() FileConn {
	return g.conn
}

func (g *GuestAllocatedVsockSocket) Ready() error {
	<-g.ready
	return g.readyErr
}

func NewGuestAllocatedVsockSocket(ctx context.Context, cid uint32, port uint32) (*GuestAllocatedVsockSocket, error) {
	listener, err := vsock.ListenContextID(cid, port, nil)
	if err != nil {
		return nil, err
	}
	refId := NewVsockSocketReferenceId(port)
	guestConn := &GuestAllocatedVsockSocket{
		listener:    listener,
		path:        fmt.Sprintf("vsock:%d", port),
		referenceId: refId,
		ready:       make(chan struct{}),
	}

	go func() {
		defer close(guestConn.ready)
		conn, err := listener.Accept()
		if err != nil {
			guestConn.readyErr = err
			return
		}
		guestConn.conn = conn.(*vsock.Conn)
	}()

	return guestConn, nil
}

var _ SocketAllocator = (*GuestUnixSocketAllocator)(nil)

type GuestUnixSocketAllocator struct {
	socketDir string
}

func NewGuestUnixSocketAllocator(socketDir string) *GuestUnixSocketAllocator {
	return &GuestUnixSocketAllocator{socketDir: socketDir}
}

var guestUnixSocketCounter = atomic.NewInt64(0)

// AllocateSocket implements SocketAllocator.
func (g *GuestUnixSocketAllocator) AllocateSocket(ctx context.Context) (AllocatedSocket, error) {

	unixSockPath := filepath.Join(g.socketDir, fmt.Sprintf("runv-%02d.sock", guestUnixSocketCounter.Add(1)))
	rid := NewUnixSocketReferenceId(unixSockPath)
	unixSock, err := NewHostAllocatedUnixSocket(ctx, unixSockPath, rid)
	if err != nil {
		return nil, err
	}
	return unixSock, nil
}
