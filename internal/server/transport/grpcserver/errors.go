package grpcserver

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
)

// mapAuthErr преобразует ошибки auth в соответствующие gRPC status-коды.
func mapAuthErr(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, auth.ErrLoginTaken):
		return status.Error(codes.AlreadyExists, "login already taken")
	case errors.Is(err, auth.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "invalid login or password")
	case errors.Is(err, auth.ErrUserNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, auth.ErrInvalidRefreshToken):
		return status.Error(codes.Unauthenticated, "invalid refresh token")
	case errors.Is(err, auth.ErrEmptyLogin):
		return status.Error(codes.InvalidArgument, "login must not be empty")
	case errors.Is(err, auth.ErrEmptyPassword):
		return status.Error(codes.InvalidArgument, "credential must not be empty")
	case errors.Is(err, auth.ErrNotImplemented):
		return status.Error(codes.Unimplemented, "not implemented")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
