package grpcclient

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidCredentials = errors.New("invalid login or credential")
	ErrLoginTaken         = errors.New("login already taken")
	ErrNotFound           = errors.New("not found")
	ErrUnavailable        = errors.New("server unavailable")
	ErrInternal           = errors.New("internal server error")
	ErrInvalidArgument    = errors.New("invalid argument")
)

// mapErr конвертирует gRPC-ошибку в клиентскую sentinel-ошибку.
func mapErr(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	switch st.Code() {
	case codes.OK:
		return nil
	case codes.AlreadyExists:
		return ErrLoginTaken
	case codes.Unauthenticated:
		return ErrInvalidCredentials
	case codes.NotFound:
		return ErrNotFound
	case codes.Unavailable, codes.DeadlineExceeded:
		return ErrUnavailable
	case codes.InvalidArgument:
		return ErrInvalidArgument
	default:
		return ErrInternal
	}
}
