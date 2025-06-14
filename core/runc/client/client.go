package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	gorunc "github.com/containerd/go-runc"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/walteh/runv/core/runc/conversion"
	"github.com/walteh/runv/core/runc/runtime"
	runvv1 "github.com/walteh/runv/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/durationpb"
)

var _ runtime.Runtime = (*Client)(nil)

// Client is a client for the runc service.

type Client struct {
	rpc  runvv1.RuncServiceClient
	conn *grpc.ClientConn
}

// ReadPidFile implements runtime.Runtime.
func (c *Client) ReadPidFile(path string) (int, error) {
	panic("unimplemented")
}

// LogFilePath implements runtime.Runtime.
func (c *Client) LogFilePath() string {
	panic("unimplemented")
}

// NewTempConsoleSocket implements runtime.Runtime.
func (c *Client) NewTempConsoleSocket() (runtime.Socket, error) {
	panic("unimplemented")
}

// Update implements runtime.Runtime.
func (c *Client) Update(ctx context.Context, id string, resources *specs.LinuxResources) error {
	panic("unimplemented")
}

// NewNullIO implements runtime.Runtime.
func (c *Client) NewNullIO() (runtime.IO, error) {
	panic("unimplemented")
}

// NewPipeIO implements runtime.Runtime.
func (c *Client) NewPipeIO(ioUID, ioGID int, opts ...gorunc.IOOpt) (runtime.IO, error) {
	panic("unimplemented")
}

// NewRuncClient creates a new client for the runc service.
func NewClient(target string, opts ...grpc.DialOption) (*Client, error) {
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to runc service: %w", err)
	}

	return NewClientFromConn(conn)
}

// NewClientFromConn creates a new client from an existing connection.
func NewClientFromConn(conn *grpc.ClientConn) (*Client, error) {

	client := &Client{
		rpc:  runvv1.NewRuncServiceClient(conn),
		conn: conn,
	}

	return client, nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Ping checks if the runc service is alive.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.rpc.Ping(ctx, &runvv1.PingRequest{})
	return err
}

// List returns all containers created inside the provided runc root directory.
func (c *Client) List(ctx context.Context) ([]*gorunc.Container, error) {
	panic("unimplemented")
}

// State returns the state for the container provided by id.
func (c *Client) State(ctx context.Context, id string) (*gorunc.Container, error) {
	panic("unimplemented")
}

// Create creates a new container.
func (c *Client) Create(ctx context.Context, id, bundle string, options *gorunc.CreateOpts) error {
	req := &runvv1.RuncCreateRequest{}
	req.SetId(id)
	req.SetBundle(bundle)

	resp, err := c.rpc.Create(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

// Start starts an already created container.
func (c *Client) Start(ctx context.Context, id string) error {
	req := &runvv1.RuncStartRequest{}
	req.SetId(id)

	resp, err := c.rpc.Start(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

// Run runs the create, start, delete lifecycle of the container.
func (c *Client) Run(ctx context.Context, id, bundle string, options *gorunc.CreateOpts) (int, error) {
	panic("unimplemented")
}

// Delete deletes a container.
func (c *Client) Delete(ctx context.Context, id string, opts *gorunc.DeleteOpts) error {
	req := &runvv1.RuncDeleteRequest{}
	req.SetId(id)
	req.SetOptions(conversion.ConvertDeleteOptsOut(opts))

	resp, err := c.rpc.Delete(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

// Kill sends the specified signal to the container.
func (c *Client) Kill(ctx context.Context, id string, signal int, opts *gorunc.KillOpts) error {
	req := &runvv1.RuncKillRequest{}
	req.SetId(id)
	req.SetSignal(int32(signal))
	req.SetOptions(conversion.ConvertKillOptsOut(opts))

	resp, err := c.rpc.Kill(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

// Stats returns the stats for a container like cpu, memory, and io.
func (c *Client) Stats(ctx context.Context, id string) (*gorunc.Stats, error) {
	req := &runvv1.RuncStatsRequest{}
	req.SetId(id)

	resp, err := c.rpc.Stats(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetGoError() != "" {
		return nil, errors.New(resp.GetGoError())
	}
	return convertStats(resp.GetStats()), nil
}

// Pause pauses the container with the provided id.
func (c *Client) Pause(ctx context.Context, id string) error {
	req := &runvv1.RuncPauseRequest{}
	req.SetId(id)

	resp, err := c.rpc.Pause(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

// Resume resumes the container with the provided id.
func (c *Client) Resume(ctx context.Context, id string) error {
	req := &runvv1.RuncResumeRequest{}
	req.SetId(id)

	resp, err := c.rpc.Resume(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

// Ps lists all the processes inside the container returning their pids.
func (c *Client) Ps(ctx context.Context, id string) ([]int, error) {
	req := &runvv1.RuncPsRequest{}
	req.SetId(id)

	resp, err := c.rpc.Ps(ctx, req)
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

// Top lists all the processes inside the container returning the full ps data.
func (c *Client) Top(ctx context.Context, id string, psOptions string) ([]string, [][]string, error) {
	req := &runvv1.RuncTopRequest{}
	req.SetId(id)
	req.SetPsOptions(psOptions)

	resp, err := c.rpc.Top(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	if resp.GetGoError() != "" {
		return nil, nil, errors.New(resp.GetGoError())
	}

	processes := make([][]string, len(resp.GetProcesses()))
	for i, process := range resp.GetProcesses() {
		processes[i] = process.GetData()
	}

	return resp.GetHeaders(), processes, nil
}

// Version returns the runc and runtime-spec versions.
func (c *Client) Version(ctx context.Context) (gorunc.Version, error) {
	resp, err := c.rpc.Version(ctx, &runvv1.RuncVersionRequest{})
	if err != nil {
		return gorunc.Version{}, err
	}
	if resp.GetGoError() != "" {
		return gorunc.Version{}, errors.New(resp.GetGoError())
	}
	return gorunc.Version{
		Runc:   resp.GetRunc(),
		Spec:   resp.GetSpec(),
		Commit: resp.GetCommit(),
	}, nil
}

// Exec executes an additional process inside the container.
func (c *Client) Exec(ctx context.Context, id string, spec specs.Process, options *gorunc.ExecOpts) error {
	req := &runvv1.RuncExecRequest{}
	req.SetId(id)
	req.SetOptions(conversion.ConvertExecOptsOut(options))

	specOut, err := conversion.ConvertProcessSpecOut(&spec)
	if err != nil {
		return err
	}
	req.SetSpec(specOut)

	resp, err := c.rpc.Exec(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

func (c *Client) Checkpoint(ctx context.Context, id string, options *gorunc.CheckpointOpts, actions ...gorunc.CheckpointAction) error {
	req := &runvv1.RuncCheckpointRequest{}
	req.SetId(id)
	req.SetOptions(convertCheckpointOpts(options))
	req.SetActions(convertCheckpointActions(actions...))

	resp, err := c.rpc.Checkpoint(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return errors.New(resp.GetGoError())
	}
	return nil
}

func (c *Client) Events(ctx context.Context, id string, duration time.Duration) (chan *gorunc.Event, error) {
	req := &runvv1.RuncEventsRequest{}
	req.SetId(id)
	req.SetDuration(durationpb.New(duration))

	stream, err := c.rpc.Events(ctx, req)
	if err != nil {
		return nil, err
	}

	events := make(chan *gorunc.Event)

	go func() {
		defer stream.CloseSend()
		defer close(events)

		for {
			event, err := stream.Recv()
			if err != nil {
				slog.Error("failed to receive event", "error", err)
				return
			}

			events <- convertEvent(event)
		}
	}()

	return events, nil
}

func (c *Client) Restore(ctx context.Context, id, bundle string, options *gorunc.RestoreOpts) (int, error) {
	req := &runvv1.RuncRestoreRequest{}
	req.SetId(id)
	req.SetBundle(bundle)
	req.SetOptions(conversion.ConvertRestoreOptsOut(options))

	resp, err := c.rpc.Restore(ctx, req)
	if err != nil {
		return -1, err
	}
	if resp.GetGoError() != "" {
		return -1, errors.New(resp.GetGoError())
	}
	return int(resp.GetStatus()), nil
}

func convertCheckpointOpts(opts *gorunc.CheckpointOpts) *runvv1.RuncCheckpointOptions {
	output := &runvv1.RuncCheckpointOptions{}

	output.SetImagePath(opts.ImagePath)
	output.SetWorkDir(opts.WorkDir)
	output.SetParentPath(opts.ParentPath)
	output.SetAllowOpenTcp(opts.AllowOpenTCP)
	output.SetAllowExternalUnixSockets(opts.AllowExternalUnixSockets)
	output.SetAllowTerminal(opts.AllowTerminal)
	output.SetCriuPageServer(opts.CriuPageServer)
	output.SetFileLocks(opts.FileLocks)
	output.SetCgroups(string(opts.Cgroups))
	output.SetEmptyNamespaces(opts.EmptyNamespaces)
	output.SetLazyPages(opts.LazyPages)
	output.SetStatusFile(opts.StatusFile.Name())
	output.SetExtraArgs(opts.ExtraArgs)

	return output
}

func convertCheckpointActions(actions ...gorunc.CheckpointAction) []*runvv1.RuncCheckpointAction {
	output := make([]*runvv1.RuncCheckpointAction, len(actions))
	for i, action := range actions {
		tmp := &runvv1.RuncCheckpointAction{}
		tmp.SetAction(action([]string{}))
		output[i] = tmp
	}
	return output
}

func convertStats(stats *runvv1.RuncStats) *gorunc.Stats {
	if stats == nil {
		return nil
	}

	return &gorunc.Stats{}
}

func convertEvent(event *runvv1.RuncEvent) *gorunc.Event {
	var errz error
	if event.GetErr() != "" {
		errz = errors.New(event.GetErr())
	}

	return &gorunc.Event{
		Type:  event.GetType(),
		ID:    event.GetId(),
		Stats: convertStats(event.GetStats()),
		Err:   errz,
	}
}
