package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/patyukin/mbs-log/internal/config"
	"github.com/patyukin/mbs-log/internal/db"
	"github.com/patyukin/mbs-log/internal/server"
	"github.com/patyukin/mbs-log/internal/usecase"
	"github.com/patyukin/mbs-pkg/pkg/dbconn"
	"github.com/patyukin/mbs-pkg/pkg/grpc_server"
	"github.com/patyukin/mbs-pkg/pkg/kafka"
	"github.com/patyukin/mbs-pkg/pkg/migrator"
	"github.com/patyukin/mbs-pkg/pkg/minio"
	"github.com/patyukin/mbs-pkg/pkg/mux_server"
	desc "github.com/patyukin/mbs-pkg/pkg/proto/logger_v1"
	"github.com/patyukin/mbs-pkg/pkg/tracing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/reflection"
)

const ServiceName = "LoggerService"

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Msgf("failed to load config, error: %v", err)
	}

	_, closer, err := tracing.InitJaeger(fmt.Sprintf("localhost:6831"), ServiceName)
	if err != nil {
		log.Fatal().Msgf("failed to initialize tracer: %v", err)
	}

	defer closer()

	log.Info().Msg("Jaeger connected")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCServer.Port))
	if err != nil {
		log.Fatal().Msgf("failed to listen: %v", err)
	}

	dbConn, err := dbconn.NewClickhouse(ctx, cfg.ClickhouseDsn)
	if err != nil {
		log.Fatal().Msgf("failed to connect to db: %v", err)
	}

	if err = migrator.UpMigrationsClickHouse(ctx, dbConn); err != nil {
		log.Fatal().Msgf("failed to up migrations: %v", err)
	}

	kfk, err := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.ConsumerGroup, cfg.Kafka.Topics)
	if err != nil {
		log.Fatal().Msgf("failed to create kafka consumer, err: %v", err)
	}

	mn, err := minio.New(
		ctx,
		cfg.Minio.Endpoint,
		cfg.Minio.AccessKey,
		cfg.Minio.SecretKey,
		cfg.Minio.Bucket,
		false,
	)
	if err != nil {
		log.Fatal().Msgf("failed to create minio: %v", err)
	}

	defer kfk.Close()

	registry := db.New(dbConn)
	uc := usecase.New(registry, kfk, mn)
	srv := server.New(uc)

	// grpc server
	s := grpc_server.NewGRPCServer()

	reflection.Register(s)
	desc.RegisterLoggerServiceServer(s, srv)
	grpcPrometheus.Register(s)

	// mux server
	muxServer := mux_server.New()

	errCh := make(chan error)

	// run log consumer
	go func() {
		if err = kfk.ProcessMessages(ctx, uc.LogProcess); err != nil {
			log.Error().Msgf("failed to process messages: %v", err)
			errCh <- err
		}
	}()

	// GRPC server
	go func() {
		log.Info().Msgf("GRPC started on :%d", cfg.GRPCServer.Port)
		if err = s.Serve(lis); err != nil {
			log.Error().Msgf("failed to serve: %v", err)
			errCh <- err
		}
	}()

	// metrics + pprof server
	go func() {
		if err = muxServer.Run(cfg.HttpServer.Port); err != nil {
			log.Error().Msgf("Failed to serve Prometheus metrics: %v", err)
			errCh <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err = <-errCh:
		log.Error().Msgf("Failed to run, err: %v", err)
	case res := <-sigChan:
		if res == syscall.SIGINT || res == syscall.SIGTERM {
			log.Info().Msg("Signal received")
		} else if res == syscall.SIGHUP {
			log.Info().Msg("Signal received")
		}
	}

	log.Info().Msg("Shutting Down")

	// stop grpc server
	s.GracefulStop()

	// stop pprof server
	if err = muxServer.Shutdown(ctx); err != nil {
		log.Error().Msgf("failed to shutdown pprof server: %s", err.Error())
	}

	if err = dbConn.Close(); err != nil {
		log.Error().Msgf("failed db connection close: %s", err.Error())
	}
}
