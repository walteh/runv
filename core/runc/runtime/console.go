package runtime

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/containerd/console"
	"github.com/walteh/runv/core/runc/file"
	"golang.org/x/sys/unix"
)

var _ ConsoleSocket = &HostConsoleSocket{}

type HostConsoleSocket struct {
	socket AllocatedSocket
	path   string
	conn   *net.UnixConn
	// unusedfd uintptr
}

func (h *HostConsoleSocket) FileConn() file.FileConn {
	return h.conn
}

func (h *HostConsoleSocket) Close() error {
	return h.conn.Close()
}

func (h *HostConsoleSocket) Path() string {
	return h.path
}

func (h *HostConsoleSocket) ReceiveMaster() (console.Console, error) {
	f, err := RecvFd(h.conn)
	if err != nil {
		return nil, err
	}
	return console.ConsoleFromFile(f)
}

func NewHostUnixConsoleSocket(ctx context.Context, socket UnixAllocatedSocket) (*HostConsoleSocket, error) {
	return &HostConsoleSocket{socket: socket, path: socket.Path(), conn: socket.Conn().(*net.UnixConn)}, nil
}

func NewHostVsockFdConsoleSocket(ctx context.Context, socket VsockAllocatedSocket, proxier VsockProxier) (*HostConsoleSocket, error) {
	conn, path, err := proxier.Proxy(ctx, socket.Port())
	if err != nil {
		return nil, err
	}
	return &HostConsoleSocket{socket: socket, path: path, conn: conn}, nil
}

func NewHostConsoleSocket(ctx context.Context, socket AllocatedSocket, proxier VsockProxier) (*HostConsoleSocket, error) {
	switch v := socket.(type) {
	case UnixAllocatedSocket:
		return NewHostUnixConsoleSocket(ctx, v)
	case VsockAllocatedSocket:
		return NewHostVsockFdConsoleSocket(ctx, v, proxier)
	default:
		return nil, fmt.Errorf("invalid socket type: %T", socket)
	}
}

func RecvFd(socket *net.UnixConn) (*os.File, error) {
	const MaxNameLen = 4096
	oobSpace := unix.CmsgSpace(4)

	name := make([]byte, MaxNameLen)
	oob := make([]byte, oobSpace)

	n, oobn, _, _, err := socket.ReadMsgUnix(name, oob)
	if err != nil {
		return nil, err
	}

	if n >= MaxNameLen || oobn != oobSpace {
		return nil, fmt.Errorf("recvfd: incorrect number of bytes read (n=%d oobn=%d)", n, oobn)
	}

	// Truncate.
	name = name[:n]
	oob = oob[:oobn]

	scms, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return nil, err
	}
	if len(scms) != 1 {
		return nil, fmt.Errorf("recvfd: number of SCMs is not 1: %d", len(scms))
	}
	scm := scms[0]

	fds, err := unix.ParseUnixRights(&scm)
	if err != nil {
		return nil, err
	}
	if len(fds) != 1 {
		return nil, fmt.Errorf("recvfd: number of fds is not 1: %d", len(fds))
	}
	fd := uintptr(fds[0])

	return os.NewFile(fd, string(name)), nil
}
