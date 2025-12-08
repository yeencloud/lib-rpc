package rpc

import (
	"context"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/yeencloud/lib-rpc/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type RPCClient struct {
	url string

	Connection *grpc.ClientConn
}

func (c *RPCClient) Connect() error {
	if c.Connection != nil && c.Connection.GetState() == connectivity.Ready {
		return nil
	}

	conn, err := grpc.NewClient(c.url, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			val := ctx.Value("correlationid")
			corrID, ok := val.(string)
			if !ok {
				corrID = ""
			}

			md := metadata.Pairs(
				domain.CorrelationIDMetadataKey, corrID,
				domain.RequestIDMetadataKey, uuid.New().String(),
			)
			ctx = metadata.NewOutgoingContext(ctx, md)

			ctx, cancel2 := context.WithTimeout(ctx, time.Second)
			defer cancel2()

			err := invoker(ctx, method, req, reply, cc, opts...)

			if err != nil {
				st, ok := status.FromError(err)
				if ok && st.Code() == codes.InvalidArgument {
					return domain.BadRequestError{}
				}
				return err
			}

			return nil
		}))
	if err != nil {
		log.Errorf("grpc.NewClient failed: %v", err)
		return domain.ServiceUnreachableError{}
	}

	c.Connection = conn

	return nil
}

func NewRPCClient(url string) *RPCClient {
	return &RPCClient{
		url: url,
	}
}
