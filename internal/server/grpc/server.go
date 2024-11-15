package grpc

import (
	grpcopentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"google.golang.org/grpc"
)

func NewGRPCServer() *grpc.Server {
	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcopentracing.UnaryServerInterceptor(),
		),
	)
}
