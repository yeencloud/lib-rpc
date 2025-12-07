package domain

import "net/http"

// MARK: - Bad Request

type BadRequestError struct {
}

func (s BadRequestError) Error() string {
	return "bad request"
}

func (s BadRequestError) RestCode() int {
	return http.StatusBadRequest
}

// MARK: - Service Unreachable

type ServiceUnreachableError struct {
}

func (s ServiceUnreachableError) Error() string {
	return "service unreachable"
}

func (s ServiceUnreachableError) RestCode() int {
	return http.StatusServiceUnavailable
}
