package server

import (
	"context"
	"log/slog"

	"github.com/walteh/runv/core/runc/runtime"
	"github.com/walteh/runv/core/runc/state"
	runvv1 "github.com/walteh/runv/proto/v1"
)

var _ runvv1.RuncServiceServer = (*Server)(nil)

type Server struct {
	runtime         runtime.Runtime
	runtimeExtras   runtime.RuntimeExtras
	socketAllocator runtime.SocketAllocator

	state *state.State
}

// LogFilePath implements runvv1.RuncServiceServer.
func (s *Server) LogFilePath(context.Context, *runvv1.RuncLogFilePathRequest) (*runvv1.RuncLogFilePathResponse, error) {
	resp := &runvv1.RuncLogFilePathResponse{}
	resp.SetPath(s.runtime.LogFilePath() + "hi")
	slog.Info("LogFilePath", "path", resp.GetPath())
	return resp, nil
}
