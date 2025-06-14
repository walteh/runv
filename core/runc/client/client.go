package client

import (
	"context"
	"fmt"
	"time"

	runvv1 "github.com/walteh/runv/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a client for the runc service.
//
//go:opts
type Client struct {
	runcClient runvv1.RuncServiceClient
	conn       *grpc.ClientConn
}

// NewRuncClient creates a new client for the runc service.
func NewRuncClient(target string, opts ...grpc.DialOption) (*Client, error) {
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to runc service: %w", err)
	}

	client := NewClient(
		WithRuncClient(runvv1.NewRuncServiceClient(conn)),
		WithConn(conn),
	)

	return &client, nil
}

// NewClientFromConn creates a new client from an existing connection.
func NewClientFromConn(conn *grpc.ClientConn) (*Client, error) {
	client := NewClient(
		WithRuncClient(runvv1.NewRuncServiceClient(conn)),
		WithConn(conn),
	)

	return &client, nil
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
	_, err := c.runcClient.Ping(ctx, &runvv1.PingRequest{})
	return err
}

// List returns all containers created inside the provided runc root directory.
func (c *Client) List(ctx context.Context, root string) ([]*runvv1.RuncContainer, error) {
	req := &runvv1.RuncListRequest{}
	req.SetRoot(root)

	resp, err := c.runcClient.List(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetGoError() != "" {
		return nil, fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return resp.GetContainers(), nil
}

// State returns the state for the container provided by id.
func (c *Client) State(ctx context.Context, id string) (*runvv1.RuncContainer, error) {
	req := &runvv1.RuncStateRequest{}
	req.SetId(id)

	resp, err := c.runcClient.State(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetGoError() != "" {
		return nil, fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return resp.GetContainer(), nil
}

// Create creates a new container.
func (c *Client) Create(ctx context.Context, id, bundle string, options ...CreateOption) error {
	req := &runvv1.RuncCreateRequest{}
	req.SetId(id)
	req.SetBundle(bundle)

	for _, opt := range options {
		opt(req)
	}

	resp, err := c.runcClient.Create(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return nil
}

// Start starts an already created container.
func (c *Client) Start(ctx context.Context, id string) error {
	req := &runvv1.RuncStartRequest{}
	req.SetId(id)

	resp, err := c.runcClient.Start(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return nil
}

// Run runs the create, start, delete lifecycle of the container.
func (c *Client) Run(ctx context.Context, id, bundle string, options ...RunOption) (int32, error) {
	req := &runvv1.RuncRunRequest{}
	req.SetId(id)
	req.SetBundle(bundle)

	for _, opt := range options {
		opt(req)
	}

	resp, err := c.runcClient.Run(ctx, req)
	if err != nil {
		return -1, err
	}
	if resp.GetGoError() != "" {
		return resp.GetStatus(), fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return resp.GetStatus(), nil
}

// Delete deletes a container.
func (c *Client) Delete(ctx context.Context, id string, force bool, extraArgs ...string) error {
	req := &runvv1.RuncDeleteRequest{}
	req.SetId(id)
	req.SetForce(force)
	req.SetExtraArgs(extraArgs)

	resp, err := c.runcClient.Delete(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return nil
}

// Kill sends the specified signal to the container.
func (c *Client) Kill(ctx context.Context, id string, signal int32, all bool, extraArgs ...string) error {
	req := &runvv1.RuncKillRequest{}
	req.SetId(id)
	req.SetSignal(signal)
	req.SetAll(all)
	req.SetExtraArgs(extraArgs)

	resp, err := c.runcClient.Kill(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return nil
}

// Stats returns the stats for a container like cpu, memory, and io.
func (c *Client) Stats(ctx context.Context, id string) (*runvv1.RuncStats, error) {
	req := &runvv1.RuncStatsRequest{}
	req.SetId(id)

	resp, err := c.runcClient.Stats(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetGoError() != "" {
		return nil, fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return resp.GetStats(), nil
}

// Pause pauses the container with the provided id.
func (c *Client) Pause(ctx context.Context, id string) error {
	req := &runvv1.RuncPauseRequest{}
	req.SetId(id)

	resp, err := c.runcClient.Pause(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return nil
}

// Resume resumes the container with the provided id.
func (c *Client) Resume(ctx context.Context, id string) error {
	req := &runvv1.RuncResumeRequest{}
	req.SetId(id)

	resp, err := c.runcClient.Resume(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return nil
}

// Ps lists all the processes inside the container returning their pids.
func (c *Client) Ps(ctx context.Context, id string) ([]int32, error) {
	req := &runvv1.RuncPsRequest{}
	req.SetId(id)

	resp, err := c.runcClient.Ps(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetGoError() != "" {
		return nil, fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return resp.GetPids(), nil
}

// Top lists all the processes inside the container returning the full ps data.
func (c *Client) Top(ctx context.Context, id string, psOptions string) ([]string, [][]string, error) {
	req := &runvv1.RuncTopRequest{}
	req.SetId(id)
	req.SetPsOptions(psOptions)

	resp, err := c.runcClient.Top(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	if resp.GetGoError() != "" {
		return nil, nil, fmt.Errorf("server error: %s", resp.GetGoError())
	}

	processes := make([][]string, len(resp.GetProcesses()))
	for i, process := range resp.GetProcesses() {
		processes[i] = process.GetData()
	}

	return resp.GetHeaders(), processes, nil
}

// Version returns the runc and runtime-spec versions.
func (c *Client) Version(ctx context.Context) (string, string, string, error) {
	resp, err := c.runcClient.Version(ctx, &runvv1.RuncVersionRequest{})
	if err != nil {
		return "", "", "", err
	}
	if resp.GetGoError() != "" {
		return "", "", "", fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return resp.GetRunc(), resp.GetCommit(), resp.GetSpec(), nil
}

// Exec executes an additional process inside the container.
func (c *Client) Exec(ctx context.Context, id string, spec *runvv1.RuncProcessSpec, options ...ExecOption) error {
	req := &runvv1.RuncExecRequest{}
	req.SetId(id)
	req.SetSpec(spec)

	for _, opt := range options {
		opt(req)
	}

	resp, err := c.runcClient.Exec(ctx, req)
	if err != nil {
		return err
	}
	if resp.GetGoError() != "" {
		return fmt.Errorf("server error: %s", resp.GetGoError())
	}
	return nil
}
