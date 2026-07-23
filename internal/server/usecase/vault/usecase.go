package vault

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
)

// Repository — контракт хранилища папок.
type Repository interface {
	Create(ctx context.Context, v domain.Vault) (domain.Vault, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Vault, error)
	CheckFreshness(ctx context.Context, userID string) ([]Version, error)
}

// UseCase реализует серверные сценарии работы с папками.
type UseCase struct {
	vaults Repository
}

// New создаёт серверный vault-usecase.
func New(vaults Repository) *UseCase {
	return &UseCase{vaults: vaults}
}
