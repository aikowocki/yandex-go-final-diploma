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
	GetForUpdate(ctx context.Context, secretID, userID string) (domain.Secret, error)
	UpdateFields(ctx context.Context, secretID string, encRow, encIndex, encPayload []byte) (int64, error)
	SoftDelete(ctx context.Context, secretID string) (int64, error)
	BumpVaultVersion(ctx context.Context, vaultID string) error
	AttachBlob(ctx context.Context, secretID, blobRef string, blobSize int64) (int64, error)
}

type VaultOwnership interface {
	IsOwner(ctx context.Context, vaultID, userID string) (bool, error)
}

// TxManager выполняет fn как одну атомарную единицу работы в хранилище.
type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type UseCase struct {
	secrets Repository
	vaults  VaultOwnership
	tx      TxManager
}

func New(secrets Repository, vaults VaultOwnership, tx TxManager) *UseCase {
	return &UseCase{secrets: secrets, vaults: vaults, tx: tx}
}
