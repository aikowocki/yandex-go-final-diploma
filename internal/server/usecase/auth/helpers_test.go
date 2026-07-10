package auth_test

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
)

// passthroughTx — тестовая реализация auth.TxManager: выполняет fn с тем же ctx,
type passthroughTx struct{}

func (passthroughTx) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func newUseCase(users *mocks.MockRepository, tokens *mocks.MockTokenIssuer) *auth.UseCase {
	return auth.New(users, tokens, passthroughTx{})
}
