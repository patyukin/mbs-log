package grpc

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
	uc        UseCase
	semaphore chan struct{}
}

func New(uc UseCase, cntWorkers int) *Server {
	return &Server{
		uc:        uc,
		semaphore: make(chan struct{}, cntWorkers),
	}
}

func (s *Server) GetLogReport(ctx context.Context, in *authpb.LogReportRequest) (*authpb.LogReportResponse, error) {
	fileUrl, err := s.uc.GetLogReport(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed uc.GetLogReport: %w", err)
	}

	return &authpb.LogReportResponse{FileUrl: fileUrl}, nil
}
