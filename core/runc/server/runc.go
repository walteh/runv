package server

import (
	"context"
	"fmt"

	"github.com/walteh/runv/core/runc/conversion"
	"github.com/walteh/runv/core/runc/runtime"
	"github.com/walteh/runv/core/runc/state"
	runvv1 "github.com/walteh/runv/proto/v1"
)

var _ runvv1.RuncServiceServer = (*Server)(nil)

// Ping implements the RuncServiceServer Ping method.
func (s *Server) Ping(ctx context.Context, req *runvv1.PingRequest) (*runvv1.PingResponse, error) {
	return &runvv1.PingResponse{}, nil
}

func (s *Server) NewTempConsoleSocket(ctx context.Context, req *runvv1.RuncNewTempConsoleSocketRequest) (*runvv1.RuncNewTempConsoleSocketResponse, error) {

	socket, err := s.runtime.NewTempConsoleSocket(ctx)
	if err != nil {
		return nil, err
	}

	referenceId := runtime.NewConsoleReferenceId()
	s.state.StoreOpenConsole(referenceId, socket)

	resp := &runvv1.RuncNewTempConsoleSocketResponse{}
	resp.SetConsoleReferenceId(referenceId)
	return resp, nil
}

func NewServer(r runtime.Runtime, runtimeExtras runtime.RuntimeExtras) *Server {
	srv := &Server{
		runtime:       r,
		runtimeExtras: runtimeExtras,
		state:         state.NewState(),
	}
	return srv
}

// Create implements the RuncServiceServer Create method.
func (s *Server) Create(ctx context.Context, req *runvv1.RuncCreateRequest) (*runvv1.RuncCreateResponse, error) {
	resp := &runvv1.RuncCreateResponse{}

	opts, err := conversion.ConvertCreateOptsFromProto(ctx, req.GetOptions(), s.state)
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

// Delete implements the RuncServiceServer Delete method.
func (s *Server) Delete(ctx context.Context, req *runvv1.RuncDeleteRequest) (*runvv1.RuncDeleteResponse, error) {
	resp := &runvv1.RuncDeleteResponse{}

	opts := conversion.ConvertDeleteOptsFromProto(req.GetOptions())

	err := s.runtime.Delete(ctx, req.GetId(), opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Kill implements the RuncServiceServer Kill method.
func (s *Server) Kill(ctx context.Context, req *runvv1.RuncKillRequest) (*runvv1.RuncKillResponse, error) {
	resp := &runvv1.RuncKillResponse{}

	opts := conversion.ConvertKillOptsFromProto(req.GetOptions())

	err := s.runtime.Kill(ctx, req.GetId(), int(req.GetSignal()), opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}
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

// Exec implements the RuncServiceServer Exec method.
func (s *Server) Exec(ctx context.Context, req *runvv1.RuncExecRequest) (*runvv1.RuncExecResponse, error) {
	resp := &runvv1.RuncExecResponse{}

	if req.GetSpec() == nil {
		return nil, fmt.Errorf("spec is required")
	}

	processSpec, err := conversion.ConvertProcessSpecFromProto(req.GetSpec())
	if err != nil {
		return nil, err
	}

	opts, err := conversion.ConvertExecOptsFromProto(req.GetOptions(), s.state)
	if err != nil {
		return nil, err
	}

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

// Restore implements runvv1.RuncServiceServer.
func (s *Server) Restore(context.Context, *runvv1.RuncRestoreRequest) (*runvv1.RuncRestoreResponse, error) {
	return nil, runtime.ReflectNotImplementedError()
}

// Update implements runvv1.RuncServiceServer.
func (s *Server) Update(context.Context, *runvv1.RuncUpdateRequest) (*runvv1.RuncUpdateResponse, error) {
	return nil, runtime.ReflectNotImplementedError()
}
