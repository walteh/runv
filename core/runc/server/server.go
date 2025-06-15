package server

import (
	"context"
	"log/slog"

	runc "github.com/containerd/go-runc"
	"github.com/walteh/runv/core/runc/conversion"
	"github.com/walteh/runv/core/runc/runtime"
	runvv1 "github.com/walteh/runv/proto/v1"
	"google.golang.org/grpc"
)

type Server struct {
	runtime       runtime.Runtime
	runtimeExtras runtime.RuntimeExtras
}

// LogFilePath implements runvv1.RuncServiceServer.
func (s *Server) LogFilePath(context.Context, *runvv1.RuncLogFilePathRequest) (*runvv1.RuncLogFilePathResponse, error) {
	resp := &runvv1.RuncLogFilePathResponse{}
	resp.SetPath(s.runtime.LogFilePath() + "hi")
	slog.Info("LogFilePath", "path", resp.GetPath())
	return resp, nil
}

func NewServer(runtime runtime.Runtime, runtimeExtras runtime.RuntimeExtras) *Server {
	srv := &Server{
		runtime:       runtime,
		runtimeExtras: runtimeExtras,
	}
	return srv
}

var _ runvv1.RuncServiceServer = (*Server)(nil)

// Register registers the server with a gRPC server.
func (s *Server) Register(grpcServer *grpc.Server) {
	runvv1.RegisterRuncServiceServer(grpcServer, s)
}

// Ping implements the RuncServiceServer Ping method.
func (s *Server) Ping(ctx context.Context, req *runvv1.PingRequest) (*runvv1.PingResponse, error) {
	return &runvv1.PingResponse{}, nil
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

// State implements the RuncServiceServer State method.
func (s *Server) State(ctx context.Context, req *runvv1.RuncStateRequest) (*runvv1.RuncStateResponse, error) {
	resp := &runvv1.RuncStateResponse{}

	container, err := s.runtimeExtras.State(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	c := &runvv1.RuncContainer{}
	c.SetId(container.ID)
	c.SetPid(int32(container.Pid))
	c.SetStatus(container.Status)
	c.SetBundle(container.Bundle)
	c.SetRootfs(container.Rootfs)
	c.SetCreatedTimestamp(container.Created.UnixNano())
	c.SetAnnotations(container.Annotations)

	resp.SetContainer(c)
	return resp, nil
}

// Create implements the RuncServiceServer Create method.
func (s *Server) Create(ctx context.Context, req *runvv1.RuncCreateRequest) (*runvv1.RuncCreateResponse, error) {
	resp := &runvv1.RuncCreateResponse{}

	opts, err := conversion.ConvertCreateOptsIn(req.GetOptions())
	if err != nil {
		return nil, err
	}

	err = s.runtime.Create(ctx, req.GetId(), req.GetBundle(), opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Start implements the RuncServiceServer Start method.
func (s *Server) Start(ctx context.Context, req *runvv1.RuncStartRequest) (*runvv1.RuncStartResponse, error) {
	resp := &runvv1.RuncStartResponse{}

	err := s.runtime.Start(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Run implements the RuncServiceServer Run method.
func (s *Server) Run(ctx context.Context, req *runvv1.RuncRunRequest) (*runvv1.RuncRunResponse, error) {
	resp := &runvv1.RuncRunResponse{}

	opts := &runc.CreateOpts{
		PidFile:      req.GetPidFile(),
		NoPivot:      req.GetNoPivot(),
		NoNewKeyring: req.GetNoNewKeyring(),
		ExtraArgs:    req.GetExtraArgs(),
	}

	if req.GetDetach() {
		opts.Detach = true
	}

	status, err := s.runtimeExtras.Run(ctx, req.GetId(), req.GetBundle(), opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}
	resp.SetStatus(int32(status))
	return resp, nil
}

// Delete implements the RuncServiceServer Delete method.
func (s *Server) Delete(ctx context.Context, req *runvv1.RuncDeleteRequest) (*runvv1.RuncDeleteResponse, error) {
	resp := &runvv1.RuncDeleteResponse{}

	opts := conversion.ConvertDeleteOptsIn(req.GetOptions())

	err := s.runtime.Delete(ctx, req.GetId(), opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Kill implements the RuncServiceServer Kill method.
func (s *Server) Kill(ctx context.Context, req *runvv1.RuncKillRequest) (*runvv1.RuncKillResponse, error) {
	resp := &runvv1.RuncKillResponse{}

	opts := conversion.ConvertKillOptsIn(req.GetOptions())

	err := s.runtime.Kill(ctx, req.GetId(), int(req.GetSignal()), opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Stats implements the RuncServiceServer Stats method.
func (s *Server) Stats(ctx context.Context, req *runvv1.RuncStatsRequest) (*runvv1.RuncStatsResponse, error) {
	resp := &runvv1.RuncStatsResponse{}

	stats, err := s.runtimeExtras.Stats(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	runcStats := conversion.ConvertStatsIn(stats)
	resp.SetStats(runcStats)
	return resp, nil
}

// Pause implements the RuncServiceServer Pause method.
func (s *Server) Pause(ctx context.Context, req *runvv1.RuncPauseRequest) (*runvv1.RuncPauseResponse, error) {
	resp := &runvv1.RuncPauseResponse{}

	err := s.runtime.Pause(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Resume implements the RuncServiceServer Resume method.
func (s *Server) Resume(ctx context.Context, req *runvv1.RuncResumeRequest) (*runvv1.RuncResumeResponse, error) {
	resp := &runvv1.RuncResumeResponse{}

	err := s.runtime.Resume(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Ps implements the RuncServiceServer Ps method.
func (s *Server) Ps(ctx context.Context, req *runvv1.RuncPsRequest) (*runvv1.RuncPsResponse, error) {
	resp := &runvv1.RuncPsResponse{}

	pids, err := s.runtime.Ps(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	pidsList := make([]int32, len(pids))
	for i, pid := range pids {
		pidsList[i] = int32(pid)
	}

	resp.SetPids(pidsList)
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

	resp.SetHeaders(topResults.Headers)

	processesList := make([]*runvv1.RuncProcessData, len(topResults.Processes))
	for i, process := range topResults.Processes {
		p := &runvv1.RuncProcessData{}
		p.SetData(process)
		processesList[i] = p
	}

	resp.SetProcesses(processesList)
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

// Exec implements the RuncServiceServer Exec method.
func (s *Server) Exec(ctx context.Context, req *runvv1.RuncExecRequest) (*runvv1.RuncExecResponse, error) {
	resp := &runvv1.RuncExecResponse{}

	if req.GetSpec() == nil {
		resp.SetGoError("spec is required")
		return resp, nil
	}

	processSpec, err := conversion.ConvertProcessSpecIn(req.GetSpec())
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	opts := conversion.ConvertExecOptsIn(req.GetOptions())

	err = s.runtime.Exec(ctx, req.GetId(), *processSpec, opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}

	return resp, nil
}

// Checkpoint implements runvv1.RuncServiceServer.
func (s *Server) Checkpoint(context.Context, *runvv1.RuncCheckpointRequest) (*runvv1.RuncCheckpointResponse, error) {
	return nil, runtime.ReflectNotImplementedError()
}

// Events implements runvv1.RuncServiceServer.
func (s *Server) Events(*runvv1.RuncEventsRequest, grpc.ServerStreamingServer[runvv1.RuncEvent]) error {
	return runtime.ReflectNotImplementedError()
}

// Restore implements runvv1.RuncServiceServer.
func (s *Server) Restore(context.Context, *runvv1.RuncRestoreRequest) (*runvv1.RuncRestoreResponse, error) {
	return nil, runtime.ReflectNotImplementedError()
}

// Update implements runvv1.RuncServiceServer.
func (s *Server) Update(context.Context, *runvv1.RuncUpdateRequest) (*runvv1.RuncUpdateResponse, error) {
	return nil, runtime.ReflectNotImplementedError()
}
