package grpcserver

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver/mapper"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault"
)

// mapVaultErr преобразует ошибки vault-usecase в gRPC status-коды.
func mapVaultErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, vault.ErrEmptyUserID),
		errors.Is(err, vault.ErrEmptyVaultKey),
		errors.Is(err, vault.ErrEmptyEncName):
		return status.Error(codes.InvalidArgument, "invalid vault request")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

// mapSecretErr преобразует ошибки secret-usecase в gRPC status-коды.
func mapSecretErr(err error) error {
	if err == nil {
		return nil
	}

	// Конфликт версий → codes.Aborted + деталь с актуальной серверной версией секрета.
	if conflict, ok := errors.AsType[*secret.ErrConflict](err); ok {
		return secretConflictStatus(conflict)
	}

	switch {
	case errors.Is(err, secret.ErrVaultNotFound):
		return status.Error(codes.NotFound, "vault not found")
	case errors.Is(err, secret.ErrSecretNotFound):
		return status.Error(codes.NotFound, "secret not found")
	case errors.Is(err, secret.ErrEmptyUserID),
		errors.Is(err, secret.ErrEmptyVaultID),
		errors.Is(err, secret.ErrEmptySecretID),
		errors.Is(err, secret.ErrEmptyEncRow),
		errors.Is(err, secret.ErrEmptyEncIndex):
		return status.Error(codes.InvalidArgument, "invalid secret request")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

// secretConflictStatus строит gRPC-статус Aborted с деталью SecretConflict (серверная версия).
func secretConflictStatus(conflict *secret.ErrConflict) error {
	st := status.New(codes.Aborted, "secret version conflict")
	withDetails, derr := st.WithDetails(mapper.SecretConflictDetail(conflict.Current))
	if derr != nil {
		// Не удалось приложить деталь — отдаём хотя бы код Aborted.
		return st.Err()
	}
	return withDetails.Err()
}

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
