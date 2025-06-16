package runtime

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
)

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

func (h *HostAllocatedSocket) Conn() *net.UnixConn {
	return h.conn
}

func (h *HostAllocatedSocket) Path() string {
	return h.path
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
