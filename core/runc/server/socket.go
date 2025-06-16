package server

import (
	"context"
	"fmt"

	"github.com/walteh/runv/core/runc/runtime"
	runvv1 "github.com/walteh/runv/proto/v1"
)

var _ runvv1.SocketAllocatorServiceServer = (*Server)(nil)

func (s *Server) AllocateConsole(ctx context.Context, req *runvv1.AllocateConsoleRequest) (*runvv1.AllocateConsoleResponse, error) {
	referenceId := runtime.NewConsoleReferenceId()
	cs, err := s.runtime.NewTempConsoleSocket(ctx)
	if err != nil {
		return nil, err
	}
	s.state.StoreOpenConsole(referenceId, cs)
	res := &runvv1.AllocateConsoleResponse{}
	res.SetConsoleReferenceId(referenceId)
	return res, nil
}

func (s *Server) AllocateIO(ctx context.Context, req *runvv1.AllocateIORequest) (*runvv1.AllocateIOResponse, error) {
	ioref := runtime.NewIoReferenceId()
	pio, err := s.runtime.NewPipeIO(ctx, 0, 0)
	if err != nil {
		return nil, err
	}
	s.state.StoreOpenIO(ioref, pio)
	res := &runvv1.AllocateIOResponse{}
	res.SetIoReferenceId(ioref)
	return res, nil
}

// AllocateSocket implements runvv1.SocketAllocatorServiceServer.
func (s *Server) AllocateSocket(ctx context.Context, req *runvv1.AllocateSocketRequest) (*runvv1.AllocateSocketResponse, error) {
	as, err := s.socketAllocator.AllocateSocket(ctx)
	if err != nil {
		return nil, err
	}

	referenceId := runtime.NewSocketReferenceId(as)
	s.state.StoreOpenSocket(referenceId, as)

	res := &runvv1.AllocateSocketResponse{}
	res.SetSocketReferenceId(referenceId)
	return res, nil
}

// AllocateSockets implements runvv1.SocketAllocatorServiceServer.
func (s *Server) AllocateSockets(ctx context.Context, req *runvv1.AllocateSocketsRequest) (*runvv1.AllocateSocketsResponse, error) {
	socksToClean := make([]runtime.AllocatedSocket, 0, req.GetCount())
	defer func() {
		if len(socksToClean) == 0 {
			return
		}
		for _, sock := range socksToClean {
			sock.Close()
			s.state.DeleteOpenSocket(runtime.NewSocketReferenceId(sock))
		}
	}()

	for i := 0; i < int(req.GetCount()); i++ {
		as, err := s.socketAllocator.AllocateSocket(ctx)
		if err != nil {
			return nil, err
		}
		socksToClean = append(socksToClean, as)
	}

	res := &runvv1.AllocateSocketsResponse{}
	refs := make([]string, 0, req.GetCount())
	for _, sock := range socksToClean {
		referenceId := runtime.NewSocketReferenceId(sock)
		s.state.StoreOpenSocket(referenceId, sock)
		refs = append(refs, referenceId)
	}
	res.SetSocketReferenceIds(refs)

	socksToClean = nil

	return res, nil
}

// BindConsoleToSocket implements runvv1.SocketAllocatorServiceServer.
func (s *Server) BindConsoleToSocket(ctx context.Context, req *runvv1.BindConsoleToSocketRequest) (*runvv1.BindConsoleToSocketResponse, error) {
	cs, ok := s.state.GetOpenConsole(req.GetConsoleReferenceId())
	if !ok {
		return nil, fmt.Errorf("console not found")
	}
	as, ok := s.state.GetOpenSocket(req.GetSocketReferenceId())
	if !ok {
		return nil, fmt.Errorf("socket not found")
	}

	err := s.socketAllocator.BindConsoleToSocket(ctx, cs, as)
	if err != nil {
		return nil, err
	}

	return &runvv1.BindConsoleToSocketResponse{}, nil
}

// BindIOToSockets implements runvv1.SocketAllocatorServiceServer.
func (s *Server) BindIOToSockets(ctx context.Context, req *runvv1.BindIOToSocketsRequest) (*runvv1.BindIOToSocketsResponse, error) {
	io, ok := s.state.GetOpenIO(req.GetIoReferenceId())
	if !ok {
		return nil, fmt.Errorf("io not found")
	}

	iosocks := [3]runtime.AllocatedSocket{}

	if req.GetStdinSocketReferenceId() != "" {
		sock, ok := s.state.GetOpenSocket(req.GetStdinSocketReferenceId())
		if !ok {
			return nil, fmt.Errorf("stdin socket not found")
		}
		iosocks[0] = sock
	}
	if req.GetStdoutSocketReferenceId() != "" {
		sock, ok := s.state.GetOpenSocket(req.GetStdoutSocketReferenceId())
		if !ok {
			return nil, fmt.Errorf("stdout socket not found")
		}
		iosocks[1] = sock
	}
	if req.GetStderrSocketReferenceId() != "" {
		sock, ok := s.state.GetOpenSocket(req.GetStderrSocketReferenceId())
		if !ok {
			return nil, fmt.Errorf("stderr socket not found")
		}
		iosocks[2] = sock
	}

	err := s.socketAllocator.BindIOToSockets(ctx, io, iosocks)
	if err != nil {
		return nil, err
	}

	return &runvv1.BindIOToSocketsResponse{}, nil
}

// CloseConsole implements runvv1.SocketAllocatorServiceServer.
func (s *Server) CloseConsole(ctx context.Context, req *runvv1.CloseConsoleRequest) (*runvv1.CloseConsoleResponse, error) {
	val, ok := s.state.GetOpenConsole(req.GetConsoleReferenceId())
	if !ok {
		return nil, fmt.Errorf("console not found")
	}
	val.Close()
	s.state.DeleteOpenConsole(req.GetConsoleReferenceId())
	return &runvv1.CloseConsoleResponse{}, nil
}

// CloseIO implements runvv1.SocketAllocatorServiceServer.
func (s *Server) CloseIO(ctx context.Context, req *runvv1.CloseIORequest) (*runvv1.CloseIOResponse, error) {
	val, ok := s.state.GetOpenIO(req.GetIoReferenceId())
	if !ok {
		return nil, fmt.Errorf("io not found")
	}
	val.Close()
	s.state.DeleteOpenIO(req.GetIoReferenceId())
	return &runvv1.CloseIOResponse{}, nil
}

// CloseSocket implements runvv1.SocketAllocatorServiceServer.
func (s *Server) CloseSocket(ctx context.Context, req *runvv1.CloseSocketRequest) (*runvv1.CloseSocketResponse, error) {
	val, ok := s.state.GetOpenSocket(req.GetSocketReferenceId())
	if !ok {
		return nil, fmt.Errorf("socket not found")
	}
	val.Close()
	s.state.DeleteOpenSocket(req.GetSocketReferenceId())
	return &runvv1.CloseSocketResponse{}, nil
}

// CloseSockets implements runvv1.SocketAllocatorServiceServer.
func (s *Server) CloseSockets(ctx context.Context, req *runvv1.CloseSocketsRequest) (*runvv1.CloseSocketsResponse, error) {
	for _, ref := range req.GetSocketReferenceIds() {
		val, ok := s.state.GetOpenSocket(ref)
		if !ok {
			return nil, fmt.Errorf("socket not found")
		}
		val.Close()
	}

	for _, ref := range req.GetSocketReferenceIds() {
		s.state.DeleteOpenSocket(ref)
	}
	return &runvv1.CloseSocketsResponse{}, nil
}
