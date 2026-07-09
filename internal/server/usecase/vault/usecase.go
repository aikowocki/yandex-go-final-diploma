package vault

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
)

type VaultRepository interface {
	Create(ctx context.Context, v domain.Vault) (domain.Vault, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Vault, error)
}

type UseCase struct {
	vaults VaultRepository
}

func New(vaults VaultRepository) *UseCase {
	return &UseCase{vaults: vaults}
}
