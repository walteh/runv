package grpcruntime

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	gorunc "github.com/containerd/go-runc"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/walteh/runv/core/runc/conversion"
	"github.com/walteh/runv/core/runc/runtime"
	runvv1 "github.com/walteh/runv/proto/v1"
)

var _ runtime.Runtime = (*GRPCClientRuntime)(nil)

func (c *GRPCClientRuntime) SharedDir() string {
	return c.sharedDirPathPrefix
}

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

	sock, err := c.socketAllocator.AllocateSocketStream(ctx, &runvv1.AllocateSocketStreamRequest{})
	if err != nil {
		return nil, err
	}

	refId, err := sock.Recv()
	if err != nil {
		return nil, err
	}

	hsock, err := runtime.NewHostAllocatedSocketFromId(ctx, refId.GetSocketReferenceId(), c.vsockProxier)
	if err != nil {
		return nil, err
	}

	ready := make(chan error)
	go func() {
		if err := hsock.Ready(); err != nil {
			ready <- err
			return
		}
		if err := sock.CloseSend(); err != nil {
			ready <- err
			return
		}
		ready <- nil
	}()

	select {
	case <-sock.Context().Done():
		return nil, fmt.Errorf("context done before socket was ready: %w", sock.Context().Err())
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for socket to be ready")
	case err := <-ready:
		if err != nil {
			return nil, err
		}
	}

	req := &runvv1.BindConsoleToSocketRequest{}
	req.SetConsoleReferenceId(cons.GetConsoleReferenceId())
	req.SetSocketReferenceId(refId.GetSocketReferenceId())

	// bind the two together

	_, err = c.socketAllocator.BindConsoleToSocket(ctx, req)
	if err != nil {
		return nil, err
	}

	consock, err := runtime.NewHostConsoleSocket(ctx, hsock, c.vsockProxier)
	if err != nil {
		return nil, err
	}

	c.state.StoreOpenConsole(cons.GetConsoleReferenceId(), consock)
	c.state.StoreOpenSocket(refId.GetSocketReferenceId(), hsock)

	// socket is allocated, we just have an id
	// so now we need to creater a new socket

	return consock, nil
}

// ReadPidFile implements runtime.Runtime.
func (c *GRPCClientRuntime) ReadPidFile(ctx context.Context, path string) (int, error) {
	req := &runvv1.RuncReadPidFileRequest{}
	req.SetPath(path)
	resp, err := c.runtime.ReadPidFile(ctx, req)
	if err != nil {
		return -1, err
	}
	if resp.GetGoError() != "" {
		return -1, errors.New(resp.GetGoError())
	}
	return int(resp.GetPid()), nil
}

// LogFilePath implements runtime.Runtime.
func (c *GRPCClientRuntime) LogFilePath(ctx context.Context) (string, error) {
	return filepath.Join(c.sharedDirPathPrefix, runtime.LogFileBase), nil
}

// Update implements runtime.Runtime.
func (c *GRPCClientRuntime) Update(ctx context.Context, id string, resources *specs.LinuxResources) error {
	panic("unimplemented")
}

// NewNullIO implements runtime.Runtime.
func (c *GRPCClientRuntime) NewNullIO() (runtime.IO, error) {
	return runtime.NewHostNullIo()
}

// NewPipeIO implements runtime.Runtime.
func (c *GRPCClientRuntime) NewPipeIO(ctx context.Context, ioUID, ioGID int, opts ...gorunc.IOOpt) (runtime.IO, error) {

	ropts := gorunc.IOOption{}
	for _, opt := range opts {
		opt(&ropts)
	}

	count := 0
	if ropts.OpenStderr {
		count++
	}
	if ropts.OpenStdout {
		count++
	}
	if ropts.OpenStdin {
		count++
	}

	if count == 0 {
		return nil, errors.New("no sockets to allocate")
	}

	req := &runvv1.AllocateSocketsRequest{}
	req.SetCount(uint32(count))

	iov, err := c.socketAllocator.AllocateSockets(ctx, req)
	if err != nil {
		return nil, err
	}

	ioReq := &runvv1.AllocateIORequest{}
	ioReq.SetOpenStdin(ropts.OpenStdin)
	ioReq.SetOpenStdout(ropts.OpenStdout)
	ioReq.SetOpenStderr(ropts.OpenStderr)

	sock, err := c.socketAllocator.AllocateIO(ctx, ioReq)
	if err != nil {
		return nil, err
	}

	count2 := 0

	bindReq := &runvv1.BindIOToSocketsRequest{}
	bindReq.SetIoReferenceId(sock.GetIoReferenceId())

	if ropts.OpenStdin {
		bindReq.SetStdinSocketReferenceId(iov.GetSocketReferenceIds()[count2])
		count2++
	}
	if ropts.OpenStdout {
		bindReq.SetStdoutSocketReferenceId(iov.GetSocketReferenceIds()[count2])
		count2++
	}
	if ropts.OpenStderr {
		bindReq.SetStderrSocketReferenceId(iov.GetSocketReferenceIds()[count2])
	}

	_, err = c.socketAllocator.BindIOToSockets(ctx, bindReq)
	if err != nil {
		return nil, err
	}

	var stdinRef, stdoutRef, stderrRef string

	if ropts.OpenStdin {
		stdinRef = bindReq.GetStdinSocketReferenceId()
	}
	if ropts.OpenStdout {
		stdoutRef = bindReq.GetStdoutSocketReferenceId()
	}
	if ropts.OpenStderr {
		stderrRef = bindReq.GetStderrSocketReferenceId()
	}

	var stdinAllocated, stdoutAllocated, stderrAllocated runtime.AllocatedSocket

	if stdinRef != "" {
		stdinAllocated, err = runtime.NewHostAllocatedSocketFromId(ctx, stdinRef, c.vsockProxier)
		if err != nil {
			return nil, err
		}
	}

	if stdoutRef != "" {
		stdoutAllocated, err = runtime.NewHostAllocatedSocketFromId(ctx, stdoutRef, c.vsockProxier)
		if err != nil {
			return nil, err
		}
	}

	if stderrRef != "" {
		stderrAllocated, err = runtime.NewHostAllocatedSocketFromId(ctx, stderrRef, c.vsockProxier)
		if err != nil {
			return nil, err
		}
	}

	ioz := runtime.NewHostUnixProxyIo(ctx, stdinAllocated, stdoutAllocated, stderrAllocated)

	return ioz, nil
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
