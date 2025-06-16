package file

import (
	"net"

	"github.com/containerd/console"
)

// NewHostFile dials the given Unix socket path (e.g. vsudd proxy).
func NewUnixConnFile(socketPath string) (console.File, error) {
	addr := &net.UnixAddr{Name: socketPath, Net: "unix"}
	c, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, err
	}
	return NewFileFromConn(socketPath, c), nil
}
