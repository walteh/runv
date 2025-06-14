package runtime

import (
	"context"
	"path/filepath"

	gorunc "github.com/containerd/go-runc"
	"golang.org/x/sys/unix"
)

var _ Runtime = (*GoRuncRuntime)(nil)
var _ RuntimeExtras = (*GoRuncRuntime)(nil)

type GoRuncRuntime struct {
	*gorunc.Runc
}

func WrapdGoRuncRuntime(rt *gorunc.Runc) *GoRuncRuntime {
	return &GoRuncRuntime{
		Runc: rt,
	}
}

func (r *GoRuncRuntime) LogFilePath() string {
	return r.Runc.Log
}

func (r *GoRuncRuntime) NewTempConsoleSocket() (*gorunc.Socket, error) {
	return gorunc.NewTempConsoleSocket()
}

type GoRuncRuntimeCreator struct{}

func (c *GoRuncRuntimeCreator) Create(ctx context.Context, opts *RuntimeOptions) Runtime {
	r := WrapdGoRuncRuntime(&gorunc.Runc{
		Command:       opts.Runtime,
		Log:           filepath.Join(opts.Path, "log.json"),
		LogFormat:     gorunc.JSON,
		PdeathSignal:  unix.SIGKILL,
		Root:          filepath.Join(opts.Root, opts.Namespace),
		SystemdCgroup: opts.SystemdCgroup,
	})
	return r
}
