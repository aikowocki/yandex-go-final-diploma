package secret

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
)

type Repository interface {
	Create(ctx context.Context, s domain.Secret) (domain.Secret, error)
	ListRow(ctx context.Context, vaultID, userID string) ([]domain.Secret, error)
	ListIndex(ctx context.Context, vaultID, userID string) ([]domain.Secret, error)
	GetPayload(ctx context.Context, secretID, userID string) (domain.Secret, error)
}

type VaultOwnership interface {
	IsOwner(ctx context.Context, vaultID, userID string) (bool, error)
}

type UseCase struct {
	secrets Repository
	vaults  VaultOwnership
}

func New(secrets Repository, vaults VaultOwnership) *UseCase {
	return &UseCase{secrets: secrets, vaults: vaults}
}
