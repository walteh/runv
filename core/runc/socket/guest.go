package socket

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/walteh/runv/core/runc/runtime"
)

var _ runtime.SocketAllocator = (*GuestUnixSocketAllocator)(nil)

type GuestUnixSocketAllocator struct {
	socketDir string
}

// AllocateSocket implements runtime.SocketAllocator.
func (g *GuestUnixSocketAllocator) AllocateSocket(ctx context.Context) (runtime.AllocatedSocket, error) {
	dir, err := os.MkdirTemp(g.socketDir, "runv-unix-sock-")
	if err != nil {
		return nil, err
	}
	unixSockPath := filepath.Join(dir, fmt.Sprintf("runv-unix-sock-%d.sock", time.Now().UnixNano()))
	rid := runtime.NewUnixSocketReferenceId(unixSockPath)
	unixSock, err := runtime.NewHostAllocatedUnixSocket(ctx, unixSockPath, rid)
	if err != nil {
		return nil, err
	}
	return unixSock, nil
}

type GuestVsockSocketAllocator struct {
	vsockDir string
}
