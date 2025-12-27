package domain

import "github.com/yeencloud/lib-shared/apperr"

// MARK: - RemoteError
type RemoteError struct {
	errtype apperr.ErrorType
}

func (e RemoteError) Error() string {
	return string(e.Type())
}

func (e RemoteError) Type() apperr.ErrorType {
	return e.errtype
}

func NewRemoteError(errtype apperr.ErrorType) error {
	return RemoteError{errtype}
}

// MARK: - CallPanicedError
type CallPanicedError struct {
	RecoverInfo string
}

func (e CallPanicedError) Error() string { return e.RecoverInfo }

func (e CallPanicedError) Type() apperr.ErrorType { return apperr.ErrorTypeInternal }
