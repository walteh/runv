package server

import (
	"context"

	runmv1 "github.com/walteh/runm/proto/v1"
)

var _ runmv1.GuestManagementServiceServer = (*Server)(nil)

// GuestReadiness implements runmv1.GuestManagementServiceServer.
func (s *Server) GuestReadiness(context.Context, *runmv1.GuestReadinessRequest) (*runmv1.GuestReadinessResponse, error) {
	return &runmv1.GuestReadinessResponse{}, nil
}

// GuestRunCommand implements runmv1.GuestManagementServiceServer.
func (s *Server) GuestRunCommand(context.Context, *runmv1.GuestRunCommandRequest) (*runmv1.GuestRunCommandResponse, error) {
	return &runmv1.GuestRunCommandResponse{}, nil
}

// GuestTimeSync implements runmv1.GuestManagementServiceServer.
func (s *Server) GuestTimeSync(context.Context, *runmv1.GuestTimeSyncRequest) (*runmv1.GuestTimeSyncResponse, error) {
	return &runmv1.GuestTimeSyncResponse{}, nil
}
