package rpc

import (
	"github.com/yeencloud/lib-rpc/domain"
	"github.com/yeencloud/lib-shared/apperr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapErrorTypeToGrpcCode(err error) codes.Code {
	errtype := apperr.GetErrorTypeOrNil(err)
	const defaultCode = codes.Internal

	if errtype == nil {
		return defaultCode
	}

	switch *errtype {
	case apperr.ErrorTypeUnavailable:
		return codes.Unavailable
	case apperr.ErrorTypeUnauthorized:
		return codes.PermissionDenied
	case apperr.ErrorTypeConflict:
		return codes.AlreadyExists
	case apperr.ErrorTypeNotImplemented:
		return codes.Unimplemented
	case apperr.ErrorTypeInvalidArgument:
		return codes.InvalidArgument
	default:
		return defaultCode
	}
}

func mapRpcErrorToRemoteError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)

	errType := apperr.ErrorTypeInternal
	if !ok {
		return domain.NewRemoteError(errType)
	}

	switch st.Code() {
	case codes.Unavailable:
		errType = apperr.ErrorTypeUnavailable
	case codes.PermissionDenied:
		errType = apperr.ErrorTypeUnauthorized
	case codes.AlreadyExists:
		errType = apperr.ErrorTypeConflict
	case codes.Unimplemented:
		errType = apperr.ErrorTypeNotImplemented
	case codes.InvalidArgument:
		errType = apperr.ErrorTypeInvalidArgument
	}

	return domain.NewRemoteError(errType)
}
