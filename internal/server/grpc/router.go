package grpc

import (
	"context"
	"github.com/patyukin/mbs-pkg/pkg/proto/error_v1"
	authpb "github.com/patyukin/mbs-pkg/pkg/proto/logger_v1"
	"github.com/rs/zerolog/log"
	"time"
)

type UseCase interface {
	GetLogReport(ctx context.Context, in *authpb.LogReportRequest) error
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
	select {
	case s.semaphore <- struct{}{}:
		defer func() {
			_ = <-s.semaphore
		}()

		response := &authpb.LogReportResponse{Error: nil}

		go func() {
			if err := s.uc.GetLogReport(ctx, in); err != nil {
				log.Error().Msgf("uc.GetLogReport failed: %v", err)
				return
			}

			log.Info().Msgf("uc.GetLogReport success")
		}()

		return response, nil

	case <-time.After(1 * time.Second):
		return &authpb.LogReportResponse{
			Error: &error_v1.ErrorResponse{
				Code:        503,
				Message:     "Service Unavailable",
				Description: "Server is busy, please try again later",
			},
		}, nil
	}
}
