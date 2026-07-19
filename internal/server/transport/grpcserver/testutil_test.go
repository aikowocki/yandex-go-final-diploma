package grpcserver_test

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver/interceptor"
)

// ctxWithUser возвращает context.Context с userID, положенным тем же способом, что и
// interceptor.Auth в реальном запросе — позволяет тестировать grpc-сервисы без сети.
func ctxWithUser(userID string) context.Context {
	return interceptor.ContextWithUserID(context.Background(), userID)
}
