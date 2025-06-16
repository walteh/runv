package server

import (
	"context"

	"github.com/walteh/runv/core/runc/conversion"
	"github.com/walteh/runv/core/runc/runtime"
	runvv1 "github.com/walteh/runv/proto/v1"
	"google.golang.org/grpc"
)

var _ runvv1.RuncExtrasServiceServer = (*Server)(nil)

////////////////////////////////////////////////////////////
// RuntimeExtras
////////////////////////////////////////////////////////////

func (s *Server) Run(ctx context.Context, req *runvv1.RuncRunRequest) (*runvv1.RuncRunResponse, error) {
	resp := &runvv1.RuncRunResponse{}

	opts, err := conversion.ConvertCreateOptsFromProto(ctx, req.GetOptions(), s.state)
	if err != nil {
		return nil, err
	}

	status, err := s.runtimeExtras.Run(ctx, req.GetId(), req.GetBundle(), opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}
	resp.SetStatus(int32(status))
	return resp, nil
}

// Events implements runvv1.RuncServiceServer.
func (s *Server) Events(*runvv1.RuncEventsRequest, grpc.ServerStreamingServer[runvv1.RuncEvent]) error {
	return runtime.ReflectNotImplementedError()
}

// Stats implements the RuncServiceServer Stats method.
func (s *Server) Stats(ctx context.Context, req *runvv1.RuncStatsRequest) (*runvv1.RuncStatsResponse, error) {
	resp := &runvv1.RuncStatsResponse{}

	stats, err := s.runtimeExtras.Stats(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	runcStats, err := conversion.ConvertStatsToProto(stats)
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}
	resp.SetStats(runcStats)
	return resp, nil
}

// Top implements the RuncServiceServer Top method.
func (s *Server) Top(ctx context.Context, req *runvv1.RuncTopRequest) (*runvv1.RuncTopResponse, error) {
	resp := &runvv1.RuncTopResponse{}

	topResults, err := s.runtimeExtras.Top(ctx, req.GetId(), req.GetPsOptions())
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	resp.SetResults(conversion.ConvertTopResultsToProto(topResults))

	return resp, nil
}

// State implements the RuncServiceServer State method.
func (s *Server) State(ctx context.Context, req *runvv1.RuncStateRequest) (*runvv1.RuncStateResponse, error) {
	resp := &runvv1.RuncStateResponse{}

	container, err := s.runtimeExtras.State(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	containerz, err := conversion.ConvertContainerToProto(container)
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	resp.SetContainer(containerz)
	return resp, nil
}

// List implements the RuncServiceServer List method.
func (s *Server) List(ctx context.Context, req *runvv1.RuncListRequest) (*runvv1.RuncListResponse, error) {
	resp := &runvv1.RuncListResponse{}

	containers, err := s.runtimeExtras.List(ctx)
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	runcContainers := make([]*runvv1.RuncContainer, len(containers))
	for i, container := range containers {
		c := &runvv1.RuncContainer_builder{
			Id:               container.ID,
			Pid:              int32(container.Pid),
			Status:           container.Status,
			Bundle:           container.Bundle,
			Rootfs:           container.Rootfs,
			CreatedTimestamp: container.Created.UnixNano(),
			Annotations:      container.Annotations,
		}
		runcContainers[i] = c.Build()
	}

	resp.SetContainers(runcContainers)
	return resp, nil
}

// Version implements the RuncServiceServer Version method.
func (s *Server) Version(ctx context.Context, req *runvv1.RuncVersionRequest) (*runvv1.RuncVersionResponse, error) {
	resp := &runvv1.RuncVersionResponse{}

	version, err := s.runtimeExtras.Version(ctx)
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	resp.SetRunc(version.Runc)
	resp.SetCommit(version.Commit)
	resp.SetSpec(version.Spec)

	return resp, nil
}
