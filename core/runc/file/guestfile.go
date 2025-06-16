package file

import (
	"fmt"

	"github.com/containerd/console"
	"github.com/mdlayher/vsock"
)

// NewVsockFile dials the given VSock CID and port.
func NewVsockLibFileConn(cid, port uint32) (console.File, error) {
	c, err := vsock.Dial(cid, port, nil)
	if err != nil {
		return nil, err
	}

	name := fmt.Sprintf("vsock://%d:%d", cid, port)
	return NewFileFromConn(name, c), nil
}
