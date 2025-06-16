package runtime

import (
	"fmt"

	"go.uber.org/atomic"
)

var consoleReferenceCounter = atomic.NewUint64(0)
var ioReferenceCounter = atomic.NewUint64(0)

type vsockAllocatedReferenceIdSocket struct {
	port int
}

type unixAllocatedReferenceIdSocket struct {
	path string
}

func (v *vsockAllocatedReferenceIdSocket) Close() error {
	return nil
}

func (v *vsockAllocatedReferenceIdSocket) Port() int {
	return v.port
}

func NewSocketReferenceId(allocatedSocket AllocatedSocket) string {
	switch v := allocatedSocket.(type) {
	case VsockAllocatedSocket:
		return fmt.Sprintf("socket:vsock:%d", v.Port())
	case UnixAllocatedSocket:
		return fmt.Sprintf("socket:unix:%s", v.Path())
	}
	return ""
}

func NewUnixSocketReferenceId(path string) string {
	return fmt.Sprintf("socket:unix:%s", path)
}

func NewConsoleReferenceId() string {
	return fmt.Sprintf("console:%d", consoleReferenceCounter.Add(1))
}

func NewIoReferenceId() string {
	return fmt.Sprintf("io:%d", ioReferenceCounter.Add(1))
}
