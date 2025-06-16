package client

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/walteh/runv/core/runc/conversion"
	"github.com/walteh/runv/core/runc/runtime"
	runvv1 "github.com/walteh/runv/proto/v1"
	"google.golang.org/protobuf/types/known/durationpb"

	gorunc "github.com/containerd/go-runc"
)

var _ runtime.RuntimeExtras = (*Client)(nil)

// Stats returns the stats for a container like cpu, memory, and io.
func (c *Client) Stats(ctx context.Context, id string) (*gorunc.Stats, error) {
	req := &runvv1.RuncStatsRequest{}
	req.SetId(id)

	resp, err := c.runtimeExtras.Stats(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetGoError() != "" {
		return nil, errors.New(resp.GetGoError())
	}
	stats, err := conversion.ConvertStatsFromProto(resp.GetStats())
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// List returns all containers created inside the provided runc root directory.
func (c *Client) List(ctx context.Context) ([]*gorunc.Container, error) {
	panic("unimplemented")
}

// State returns the state for the container provided by id.
func (c *Client) State(ctx context.Context, id string) (*gorunc.Container, error) {
	panic("unimplemented")
}

// Run runs the create, start, delete lifecycle of the container.
func (c *Client) Run(ctx context.Context, id, bundle string, options *gorunc.CreateOpts) (int, error) {
	panic("unimplemented")
}

func (c *Client) Events(ctx context.Context, id string, duration time.Duration) (chan *gorunc.Event, error) {
	req := &runvv1.RuncEventsRequest{}
	req.SetId(id)
	req.SetDuration(durationpb.New(duration))

	stream, err := c.runtimeExtras.Events(ctx, req)
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

			eventz, err := conversion.ConvertEventFromProto(event)
			if err != nil {
				slog.Error("failed to convert event", "error", err)
				return
			}

			events <- eventz
		}
	}()

	return events, nil
}

// Top lists all the processes inside the container returning the full ps data.
func (c *Client) Top(ctx context.Context, id string, psOptions string) (*gorunc.TopResults, error) {
	req := &runvv1.RuncTopRequest{}
	req.SetId(id)
	req.SetPsOptions(psOptions)

	resp, err := c.runtimeExtras.Top(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetGoError() != "" {
		return nil, errors.New(resp.GetGoError())
	}

	results := conversion.ConvertTopResultsFromProto(resp.GetResults())

	return results, nil
}

// Version returns the runc and runtime-spec versions.
func (c *Client) Version(ctx context.Context) (gorunc.Version, error) {
	resp, err := c.runtimeExtras.Version(ctx, &runvv1.RuncVersionRequest{})
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
