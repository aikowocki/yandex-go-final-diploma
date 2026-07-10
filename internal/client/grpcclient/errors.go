package grpcclient

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

var (
	ErrInvalidCredentials = errors.New("invalid login or credential")
	ErrLoginTaken         = errors.New("login already taken")
	ErrNotFound           = errors.New("not found")
	ErrUnavailable        = errors.New("server unavailable")
	ErrInternal           = errors.New("internal server error")
	ErrInvalidArgument    = errors.New("invalid argument")
)

// ConflictError — сервер отклонил update/delete из-за оптимистичной блокировки: на сервере
// более новая версия. Несёт полную актуальную серверную версию секрета (все три тира),
// чтобы клиент расшифровал и показал её без отдельного запроса.
type ConflictError struct {
	Server contracts.ServerSecret
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("server version conflict (server version %d)", e.Server.Version)
}

// conflictFromStatus извлекает деталь SecretConflict из gRPC-статуса Aborted. Возвращает
// *ConflictError, если это конфликт версий, иначе nil.
func conflictFromStatus(err error) *ConflictError {
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Aborted {
		return nil
	}
	for _, d := range st.Details() {
		if c, ok := d.(*pb.SecretConflict); ok {
			return &ConflictError{Server: contracts.ServerSecret{
				ID:         c.GetSecretId(),
				Type:       int32(c.GetType()),
				Version:    c.GetVersion(),
				EncRow:     c.GetEncRow(),
				EncIndex:   c.GetEncIndex(),
				EncPayload: c.GetEncPayload(),
			}}
		}
	}
	return nil
}

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
