package rpc

import (
	"context"
	"fmt"
	"net"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/yeencloud/lib-rpc/domain"
	"github.com/yeencloud/lib-rpc/domain/config"
	logShared "github.com/yeencloud/lib-shared/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcReflection "google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc"
)

type Server struct {
	RpcServer *grpc.Server

	config *config.Config
}

func RequireMetadataPresence() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		_, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "metadata is not provided")
		}

		return handler(ctx, req)
	}
}

func RequireValidUUID(key string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logEntry := logShared.GetLoggerFromContext(ctx)
		md, _ := metadata.FromIncomingContext(ctx)

		data, ok := md[key]
		if !ok || len(data) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "metadata %s is not provided", key)
		}

		uuidValue, err := uuid.Parse(data[0])
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%s is not a valid uuid", key)
		}

		logEntry = logEntry.WithField(key, uuidValue.String())
		logEntry.Infoln("Found " + key)

		ctx = logShared.WithLogger(ctx, logEntry)
		return handler(ctx, req)
	}
}

func StartTracingRequest() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logEntry := log.NewEntry(log.StandardLogger())

		logEntry = logEntry.WithField("call_id", uuid.New()).WithField("method", info.FullMethod)

		ctx = logShared.WithLogger(ctx, logEntry)
		return handler(ctx, req)
	}
}

func AuditInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "metadata is not provided")
		}
		_ = md
		m, err := handler(ctx, req)
		if err != nil {
			log.Errorln("RPC failed with error: %v", err)
		} else {
			log.Infoln("Succeeded")
		}

		return m, err
	}
}

func NewRPCServer(config *config.Config) *Server {
	println("NewRPCServer: Initializing gRPC server")

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			StartTracingRequest(),
			RequireMetadataPresence(),
			RequireValidUUID(domain.RequestIDMetadataKey),
			RequireValidUUID(domain.CorrelationIDMetadataKey),
			AuditInterceptor(),
		),
	)

	grpcReflection.Register(grpcServer)

	return &Server{
		config:    config,
		RpcServer: grpcServer,
	}
}

func (s *Server) Start() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
	if err != nil {
		log.Fatalln(err)
	}

	if err := s.RpcServer.Serve(lis); err != nil {
		log.Fatalln(err)
	}
}
