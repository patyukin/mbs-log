package server

import (
	"context"
	"fmt"
	authpb "github.com/patyukin/mbs-pkg/pkg/proto/logger_v1"
)

type UseCase interface {
	GetLogReport(ctx context.Context, in *authpb.LogReportRequest) (string, error)
}

type Server struct {
	authpb.UnimplementedLoggerServiceServer
	uc UseCase
}

func New(uc UseCase) *Server {
	return &Server{
		uc: uc,
	}
}

func (s *Server) GetLogReport(ctx context.Context, in *authpb.LogReportRequest) (*authpb.LogReportResponse, error) {
	fileUrl, err := s.uc.GetLogReport(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed uc.GetLogReport: %w", err)
	}

	return &authpb.LogReportResponse{Message: fileUrl}, nil
}
