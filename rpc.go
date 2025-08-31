package rpc

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"google.golang.org/grpc"
)

func AuditInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		println("AuditInterceptor: Request received")
		spew.Dump(ctx)
		return handler(ctx, req)
	}
}

func NewRPCServer() *grpc.Server {
	println("NewRPCServer: Initializing gRPC server")

	return grpc.NewServer(
		grpc.UnaryInterceptor(AuditInterceptor()),
	)
}
