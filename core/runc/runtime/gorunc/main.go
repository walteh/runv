package goruncruntime

import (
	"context"
	"path/filepath"

	gorunc "github.com/containerd/go-runc"
	"github.com/walteh/runv/core/runc/runtime"
	"golang.org/x/sys/unix"
)

var _ runtime.Runtime = (*GoRuncRuntime)(nil)
var _ runtime.RuntimeExtras = (*GoRuncRuntime)(nil)

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

func (r *GoRuncRuntime) NewTempConsoleSocket(ctx context.Context) (runtime.ConsoleSocket, error) {
	return gorunc.NewTempConsoleSocket()
}

func (r *GoRuncRuntime) NewNullIO() (runtime.IO, error) {
	return gorunc.NewNullIO()
}

func (r *GoRuncRuntime) NewPipeIO(ioUID, ioGID int, opts ...gorunc.IOOpt) (runtime.IO, error) {
	return gorunc.NewPipeIO(ioUID, ioGID, opts...)
}

func (r *GoRuncRuntime) ReadPidFile(path string) (int, error) {
	return gorunc.ReadPidFile(path)
}

type GoRuncRuntimeCreator struct{}

func (c *GoRuncRuntimeCreator) Create(ctx context.Context, opts *runtime.RuntimeOptions) runtime.Runtime {
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
