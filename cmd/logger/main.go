package main

import (
	"context"
	"fmt"
	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/patyukin/mbs-log/internal/config"
	"github.com/patyukin/mbs-log/internal/db"
	"github.com/patyukin/mbs-log/internal/server/grpc"
	"github.com/patyukin/mbs-log/internal/server/mux"
	"github.com/patyukin/mbs-log/internal/usecase"
	"github.com/patyukin/mbs-pkg/pkg/dbconn"
	"github.com/patyukin/mbs-pkg/pkg/kafka"
	"github.com/patyukin/mbs-pkg/pkg/migrator"
	desc "github.com/patyukin/mbs-pkg/pkg/proto/logger_v1"
	"github.com/patyukin/mbs-pkg/pkg/rabbitmq"
	"github.com/patyukin/mbs-pkg/pkg/s3client"
	"github.com/patyukin/mbs-pkg/pkg/tracing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/reflection"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	log.Info().Msg("Opentracing connected")
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

	rbt, err := rabbitmq.New(cfg.RabbitMQURL, rabbitmq.Exchange)
	if err != nil {
		log.Fatal().Msgf("failed to create rabbit producer: %v", err)
	}

	err = rbt.BindQueueToExchange(
		rabbitmq.Exchange,
		rabbitmq.LoggerNotifyQueue,
		[]string{rabbitmq.LoggerReportRouteKey},
	)
	if err != nil {
		log.Fatal().Msgf("failed to bind LoggerNotifyQueue to exchange with - LoggerReportRouteKey, error: %v", err)
	}

	kfk, err := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.ConsumerGroup, cfg.Kafka.Topics)
	if err != nil {
		log.Fatal().Msgf("failed to create kafka consumer, err: %v", err)
	}

	s3, err := s3client.New(ctx, cfg.S3.Bucket, cfg.S3.AccessKey, cfg.S3.SecretKey)
	if err != nil {
		log.Fatal().Msgf("failed to create s3 client, err: %v", err)
	}

	defer kfk.Close()

	registry := db.New(dbConn)
	uc := usecase.New(registry, kfk, rbt, s3)
	srv := grpc.New(uc, 5)

	// grpc server
	s := grpc.NewGRPCServer()

	reflection.Register(s)
	desc.RegisterLoggerServiceServer(s, srv)
	grpcPrometheus.Register(s)

	log.Printf("server listening at %v", lis.Addr())

	// mux server
	m := mux.New()

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
		log.Info().Msgf("Prometheus metrics exposed on :%d/metrics", cfg.HttpServer.Port)
		if err = m.Run(fmt.Sprintf("0.0.0.0:%d", cfg.HttpServer.Port)); err != nil {
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
	if err = m.Shutdown(ctx); err != nil {
		log.Error().Msgf("failed to shutdown pprof server: %s", err.Error())
	}

	if err = dbConn.Close(); err != nil {
		log.Error().Msgf("failed db connection close: %s", err.Error())
	}
}
