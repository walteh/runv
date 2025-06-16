package grpcruntime

import (
	"context"
	"errors"

	gorunc "github.com/containerd/go-runc"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/walteh/runv/core/runc/conversion"
	"github.com/walteh/runv/core/runc/runtime"
	"github.com/walteh/runv/core/runc/stdio"
	runvv1 "github.com/walteh/runv/proto/v1"
)

var _ runtime.Runtime = (*GRPCClientRuntime)(nil)

// Ping checks if the runc service is alive.
func (c *GRPCClientRuntime) Ping(ctx context.Context) error {
	_, err := c.runtime.Ping(ctx, &runvv1.PingRequest{})
	return err
}

// NewTempConsoleSocket implements runtime.Runtime.
func (c *GRPCClientRuntime) NewTempConsoleSocket(ctx context.Context) (runtime.ConsoleSocket, error) {

	cons, err := c.runtime.NewTempConsoleSocket(ctx, &runvv1.RuncNewTempConsoleSocketRequest{})
	if err != nil {
		return nil, err
	}
	if cons.GetGoError() != "" {
		return nil, errors.New(cons.GetGoError())
	}

	sock, err := c.socketAllocator.AllocateSocket(ctx, &runvv1.AllocateSocketRequest{})
	if err != nil {
		return nil, err
	}

	req := &runvv1.BindConsoleToSocketRequest{}
	req.SetConsoleReferenceId(cons.GetConsoleReferenceId())
	req.SetSocketReferenceId(sock.GetSocketReferenceId())

	// bind the two together

	_, err = c.socketAllocator.BindConsoleToSocket(ctx, req)
	if err != nil {
		return nil, err
	}

	hsock, err := runtime.NewHostAllocatedSocketFromId(ctx, sock.GetSocketReferenceId(), c.vsockProxier)
	if err != nil {
		return nil, err
	}

	consock, err := runtime.NewHostConsoleSocket(ctx, hsock, c.vsockProxier)
	if err != nil {
		return nil, err
	}

	c.state.StoreOpenConsole(cons.GetConsoleReferenceId(), consock)
	c.state.StoreOpenSocket(sock.GetSocketReferenceId(), hsock)

	// socket is allocated, we just have an id
	// so now we need to creater a new socket

	return consock, nil
}

// ReadPidFile implements runtime.Runtime.
func (c *GRPCClientRuntime) ReadPidFile(path string) (int, error) {
	panic("unimplemented")
}

// LogFilePath implements runtime.Runtime.
func (c *GRPCClientRuntime) LogFilePath() string {
	resp, err := c.runtime.LogFilePath(context.Background(), &runvv1.RuncLogFilePathRequest{})
	if err != nil {
		return ""
	}
	return resp.GetPath()
}

// Update implements runtime.Runtime.
func (c *GRPCClientRuntime) Update(ctx context.Context, id string, resources *specs.LinuxResources) error {
	panic("unimplemented")
}

// NewNullIO implements runtime.Runtime.
func (c *GRPCClientRuntime) NewNullIO() (runtime.IO, error) {
	return stdio.NewHostNullIo()
}

// NewPipeIO implements runtime.Runtime.
func (c *GRPCClientRuntime) NewPipeIO(ioUID, ioGID int, opts ...gorunc.IOOpt) (runtime.IO, error) {
	return stdio.NewHostVsockProxyIo(context.Background(), opts...)
}

// Create creates a new container.
func (c *GRPCClientRuntime) Create(ctx context.Context, id, bundle string, options *gorunc.CreateOpts) error {
	conv, err := conversion.ConvertCreateOptsToProto(ctx, options)
	if err != nil {
		return err
	}

	req := &runvv1.RuncCreateRequest{}
	req.SetId(id)
	req.SetBundle(bundle)
	req.SetOptions(conv)

	resp, err := c.runtime.Create(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

// Start starts an already created container.
func (c *GRPCClientRuntime) Start(ctx context.Context, id string) error {
	req := &runvv1.RuncStartRequest{}
	req.SetId(id)

	resp, err := c.runtime.Start(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

// Delete deletes a container.
func (c *GRPCClientRuntime) Delete(ctx context.Context, id string, opts *gorunc.DeleteOpts) error {
	req := &runvv1.RuncDeleteRequest{}
	req.SetId(id)
	req.SetOptions(conversion.ConvertDeleteOptsToProto(opts))

	resp, err := c.runtime.Delete(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

// Kill sends the specified signal to the container.
func (c *GRPCClientRuntime) Kill(ctx context.Context, id string, signal int, opts *gorunc.KillOpts) error {
	req := &runvv1.RuncKillRequest{}
	req.SetId(id)
	req.SetSignal(int32(signal))
	req.SetOptions(conversion.ConvertKillOptsToProto(opts))

	resp, err := c.runtime.Kill(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

// Pause pauses the container with the provided id.
func (c *GRPCClientRuntime) Pause(ctx context.Context, id string) error {
	req := &runvv1.RuncPauseRequest{}
	req.SetId(id)

	resp, err := c.runtime.Pause(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

// Resume resumes the container with the provided id.
func (c *GRPCClientRuntime) Resume(ctx context.Context, id string) error {
	req := &runvv1.RuncResumeRequest{}
	req.SetId(id)

	resp, err := c.runtime.Resume(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

// Ps lists all the processes inside the container returning their pids.
func (c *GRPCClientRuntime) Ps(ctx context.Context, id string) ([]int, error) {
	req := &runvv1.RuncPsRequest{}
	req.SetId(id)

	resp, err := c.runtime.Ps(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetGoError() != "" {
		return nil, errors.New(resp.GetGoError())
	}
	pids := make([]int, len(resp.GetPids()))
	for i, pid := range resp.GetPids() {
		pids[i] = int(pid)
	}
	return pids, nil
}

// Exec executes an additional process inside the container.
func (c *GRPCClientRuntime) Exec(ctx context.Context, id string, spec specs.Process, options *gorunc.ExecOpts) error {
	req := &runvv1.RuncExecRequest{}
	req.SetId(id)

	specOut, err := conversion.ConvertProcessSpecToProto(&spec)
	if err != nil {
		return err
	}
	req.SetSpec(specOut)

	req.SetOptions(conversion.ConvertExecOptsToProto(options))

	resp, err := c.runtime.Exec(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

func (c *GRPCClientRuntime) Checkpoint(ctx context.Context, id string, options *gorunc.CheckpointOpts, actions ...gorunc.CheckpointAction) error {
	req := &runvv1.RuncCheckpointRequest{}
	req.SetId(id)
	req.SetOptions(conversion.ConvertCheckpointOptsToProto(options))
	req.SetActions(conversion.ConvertCheckpointActionsToProto(actions...))

	resp, err := c.runtime.Checkpoint(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

func (c *GRPCClientRuntime) Restore(ctx context.Context, id, bundle string, options *gorunc.RestoreOpts) (int, error) {
	req := &runvv1.RuncRestoreRequest{}
	req.SetId(id)
	req.SetBundle(bundle)
	req.SetOptions(conversion.ConvertRestoreOptsToProto(options))

	resp, err := c.runtime.Restore(ctx, req)
	if err != nil {
		return -1, err
	}
	if resp.GetGoError() != "" {
		return -1, errors.New(resp.GetGoError())
	}
	return int(resp.GetStatus()), nil
}
