package conversion

import (
	"context"
	"encoding/json"
	"os"

	gorunc "github.com/containerd/go-runc"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/walteh/runv/core/runc/runtime"
	runvv1 "github.com/walteh/runv/proto/v1"
)

func ConvertStatsOut(stats *runvv1.RuncStats) (*gorunc.Stats, error) {
	var runcStats gorunc.Stats
	if err := json.Unmarshal(stats.GetRawJson(), &runcStats); err != nil {
		return nil, err
	}
	return &runcStats, nil
}

// convertStats converts runc.Stats to runvv1.RuncStats
func ConvertStatsIn(stats *gorunc.Stats) (*runvv1.RuncStats, error) {
	rawJson, err := json.Marshal(stats)
	if err != nil {
		return nil, err
	}
	resp := &runvv1.RuncStats{}
	resp.SetRawJson(rawJson)
	return resp, nil
}

func ConvertCreateOptsIn(ctx context.Context, opts *runvv1.RuncCreateOptions) (*gorunc.CreateOpts, error) {
	var err error
	files := make([]*os.File, len(opts.GetExtraFiles()))
	for i, file := range opts.GetExtraFiles() {
		files[i], err = os.Open(file)
		if err != nil {
			return nil, err
		}
	}

	io, err := ConvertIOIn(ctx, opts.GetIo())
	if err != nil {
		return nil, err
	}

	return &gorunc.CreateOpts{
		PidFile:       opts.GetPidFile(),
		IO:            io,
		NoPivot:       opts.GetNoPivot(),
		NoNewKeyring:  opts.GetNoNewKeyring(),
		ConsoleSocket: NewPassThroughConsoleSocket(opts.GetConsoleSocket().GetPath()),
		Detach:        opts.GetDetach(),
		ExtraFiles:    files,
		Started:       make(chan int),
	}, nil
}

func ConvertCreateOptsOut(ctx context.Context, opts *gorunc.CreateOpts) (*runvv1.RuncCreateOptions, error) {
	var err error
	ioz, err := ConvertIOOut(ctx, opts.IO)
	if err != nil {
		return nil, err
	}

	files := make([]string, len(opts.ExtraFiles))
	for i, file := range opts.ExtraFiles {
		files[i] = file.Name()
	}

	cs := &runvv1.RuncConsoleSocket{}
	cs.SetPath(opts.ConsoleSocket.Path())

	res := &runvv1.RuncCreateOptions{}
	res.SetIo(ioz)
	res.SetPidFile(opts.PidFile)
	res.SetNoPivot(opts.NoPivot)
	res.SetNoNewKeyring(opts.NoNewKeyring)
	res.SetConsoleSocket(cs)
	res.SetDetach(opts.Detach)
	res.SetExtraFiles(files)

	return res, nil
}

func ConvertExecOptsIn(opts *runvv1.RuncExecOptions) *gorunc.ExecOpts {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertExecOptsOut(opts *gorunc.ExecOpts) *runvv1.RuncExecOptions {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertKillOptsIn(opts *runvv1.RuncKillOptions) *gorunc.KillOpts {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertKillOptsOut(opts *gorunc.KillOpts) *runvv1.RuncKillOptions {
	panic(runtime.ReflectNotImplementedError())
}

// checkpoint in out

func ConvertCheckpointOptsIn(opts *runvv1.RuncCheckpointOptions) *gorunc.CheckpointOpts {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertCheckpointOptsOut(opts *gorunc.CheckpointOpts) *runvv1.RuncCheckpointOptions {
	panic(runtime.ReflectNotImplementedError())
}

// restore in out
func ConvertRestoreOptsIn(opts *runvv1.RuncRestoreOptions) *gorunc.RestoreOpts {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertRestoreOptsOut(opts *gorunc.RestoreOpts) *runvv1.RuncRestoreOptions {
	panic(runtime.ReflectNotImplementedError())
}

// checkpoint actions in out

func ConvertCheckpointActionsIn(actions []runvv1.RuncCheckpointAction) []gorunc.CheckpointAction {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertCheckpointActionsOut(actions []gorunc.CheckpointAction) []runvv1.RuncCheckpointAction {
	panic(runtime.ReflectNotImplementedError())
}

// delete in out
func ConvertDeleteOptsIn(opts *runvv1.RuncDeleteOptions) *gorunc.DeleteOpts {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertDeleteOptsOut(opts *gorunc.DeleteOpts) *runvv1.RuncDeleteOptions {
	panic(runtime.ReflectNotImplementedError())
}

// linux resources in out

func ConvertLinuxResourcesIn(resources *runvv1.RuncLinuxResources) (*specs.LinuxResources, error) {
	var linuxResources specs.LinuxResources
	if err := json.Unmarshal(resources.GetRawJson(), &linuxResources); err != nil {
		return nil, err
	}
	return &linuxResources, nil
}

func ConvertLinuxResourcesOut(resources *specs.LinuxResources) (*runvv1.RuncLinuxResources, error) {
	rawJson, err := json.Marshal(resources)
	if err != nil {
		return nil, err
	}
	resp := &runvv1.RuncLinuxResources{}
	resp.SetRawJson(rawJson)
	return resp, nil
}

func ConvertProcessSpecIn(resources *runvv1.RuncProcessSpec) (*specs.Process, error) {
	var ProcessSpec specs.Process
	if err := json.Unmarshal(resources.GetRawJson(), &ProcessSpec); err != nil {
		return nil, err
	}
	return &ProcessSpec, nil
}

func ConvertProcessSpecOut(resources *specs.Process) (*runvv1.RuncProcessSpec, error) {
	rawJson, err := json.Marshal(resources)
	if err != nil {
		return nil, err
	}
	resp := &runvv1.RuncProcessSpec{}
	resp.SetRawJson(rawJson)
	return resp, nil
}
