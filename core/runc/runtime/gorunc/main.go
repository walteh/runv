package goruncruntime

import (
	"context"
	"path/filepath"

	"golang.org/x/sys/unix"

	gorunc "github.com/containerd/go-runc"

	"github.com/walteh/runm/core/runc/runtime"
)

var _ runtime.Runtime = (*GoRuncRuntime)(nil)
var _ runtime.RuntimeExtras = (*GoRuncRuntime)(nil)

type GoRuncRuntime struct {
	*gorunc.Runc
	sharedDirPathPrefix string
}

func (r *GoRuncRuntime) SharedDir() string {
	return r.sharedDirPathPrefix
}

func WrapdGoRuncRuntime(rt *gorunc.Runc) *GoRuncRuntime {
	return &GoRuncRuntime{
		Runc: rt,
	}
}

func (r *GoRuncRuntime) NewTempConsoleSocket(ctx context.Context) (runtime.ConsoleSocket, error) {
	return gorunc.NewTempConsoleSocket()
}

func (r *GoRuncRuntime) NewNullIO() (runtime.IO, error) {
	return gorunc.NewNullIO()
}

func (r *GoRuncRuntime) NewPipeIO(ctx context.Context, cioUID, ioGID int, opts ...gorunc.IOOpt) (runtime.IO, error) {
	return gorunc.NewPipeIO(cioUID, ioGID, opts...)
}

func (r *GoRuncRuntime) ReadPidFile(ctx context.Context, path string) (int, error) {
	return gorunc.ReadPidFile(path)
}

func (r *GoRuncRuntime) RuncRun(ctx context.Context, id, bundle string, options *gorunc.CreateOpts) (int, error) {
	return r.Runc.Run(ctx, id, bundle, options)
}

type GoRuncRuntimeCreator struct {
}

func (c *GoRuncRuntimeCreator) Create(ctx context.Context, sharedDir string, opts *runtime.RuntimeOptions) runtime.Runtime {
	r := WrapdGoRuncRuntime(&gorunc.Runc{
		Command:       opts.ProcessCreateConfig.Runtime,
		Log:           filepath.Join(sharedDir, runtime.LogFileBase),
		LogFormat:     gorunc.JSON,
		PdeathSignal:  unix.SIGKILL,
		Root:          filepath.Join(opts.ProcessCreateConfig.Options.Root, opts.Namespace),
		SystemdCgroup: opts.ProcessCreateConfig.Options.SystemdCgroup,
	})
	return r
}
