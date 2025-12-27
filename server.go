package rpc

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/yeencloud/lib-rpc/domain"
	"github.com/yeencloud/lib-rpc/domain/config"
	"github.com/yeencloud/lib-shared/apperr"
	logShared "github.com/yeencloud/lib-shared/log"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcReflection "google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
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

func handleError(err error) error {
	errStatus := status.New(mapErrorTypeToGrpcCode(err), err.Error())

	var detailedError apperr.DetailedError
	if errors.As(err, &detailedError) {
		details := detailedError.Details()
		d := &errdetails.ErrorInfo{
			Reason:   details.Reason,
			Metadata: details.Details,
		}
		errStatus, _ = errStatus.WithDetails(protoadapt.MessageV1Of(d))
	}

	return errStatus.Err()
}

func AuditInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "metadata is not provided")
		}
		_ = md
		m, err := handler(ctx, req)

		if err == nil {
			log.Infoln("RPC call succeeded")
			return m, nil
		}

		err = handleError(err)
		log.Errorf("RPC call failed with error: %v", err)
		return nil, err
	}
}

func RecoverPanic() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (data interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Infoln("Recovered from panic", r)

				err = domain.CallPanicedError{
					RecoverInfo: fmt.Sprint(r),
				}
			}
		}()

		return handler(ctx, req)
	}
}

func NewRPCServer(config *config.Config) *Server {
	log.Infoln("NewRPCServer: Initializing gRPC server")

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			StartTracingRequest(),
			RequireMetadataPresence(),
			RequireValidUUID(domain.RequestIDMetadataKey),
			RequireValidUUID(domain.CorrelationIDMetadataKey),
			AuditInterceptor(),
			RecoverPanic(),
		),
	)

	grpcReflection.Register(grpcServer)

	return &Server{
		config:    config,
		RpcServer: grpcServer,
	}
}

func (s *Server) Start(ctx context.Context) {
	addr := fmt.Sprintf(":%d", s.config.Port)

	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	if err := s.RpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
