package file

import (
	"net"
	"os"
	"syscall"

	"github.com/containerd/console"
)

type FileConn interface {
	syscall.Conn
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
}

type FileFromConn struct {
	name string
	conn FileConn
}

var _ console.File = &FileFromConn{}

func NewFileFromConn(name string, conn FileConn) *FileFromConn {
	return &FileFromConn{name: name, conn: conn}
}

func (f *FileFromConn) Name() string {
	return f.name
}

func (f *FileFromConn) Read(p []byte) (n int, err error) {
	return f.conn.Read(p)
}

func (f *FileFromConn) Write(p []byte) (n int, err error) {
	return f.conn.Write(p)
}

func (f *FileFromConn) Close() error {
	return f.conn.Close()
}

func (f *FileFromConn) Fd() uintptr {
	raw, err := f.conn.SyscallConn()
	if err != nil {
		panic(err)
	}
	var fd uintptr
	raw.Control(func(sfd uintptr) {
		fd = sfd
	})
	return fd
}

type FileFromFd struct {
	name string
	fd   uintptr
	conn net.Conn
}

var _ console.File = &FileFromFd{}

func NewFileConnFromRawFd(name string, fd uintptr) (*FileFromFd, error) {
	conn, err := net.FileConn(os.NewFile(uintptr(fd), ""))
	if err != nil {
		return nil, err
	}
	return &FileFromFd{name: name, fd: fd, conn: conn}, nil
}

func NewFileConnFromRawFdRawNetConn(name string, fd uintptr, conn net.Conn) *FileFromFd {
	return &FileFromFd{name: name, fd: fd, conn: conn}
}

func (f *FileFromFd) Read(p []byte) (n int, err error) {
	return f.conn.Read(p)
}

func (f *FileFromFd) Write(p []byte) (n int, err error) {
	return f.conn.Write(p)
}

func (f *FileFromFd) Close() error {
	return f.conn.Close()
}

func (f *FileFromFd) Fd() uintptr {
	return f.fd
}

func (f *FileFromFd) Name() string {
	return f.name
}
