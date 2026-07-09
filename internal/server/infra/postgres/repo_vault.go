package postgres

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres/gen"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault"
)

// VaultRepo реализует vault.VaultRepository поверх sqlc-запросов.
type VaultRepo struct {
	db *DB
}

var _ vault.VaultRepository = (*VaultRepo)(nil)

func NewVaultRepo(db *DB) *VaultRepo {
	return &VaultRepo{db: db}
}

func (r *VaultRepo) q(ctx context.Context) *gen.Queries {
	return gen.New(r.db.querier(ctx))
}

func (r *VaultRepo) Create(ctx context.Context, v domain.Vault) (domain.Vault, error) {
	userID, err := parseUUID(v.UserID)
	if err != nil {
		return domain.Vault{}, fmt.Errorf("create vault: invalid user id: %w", err)
	}

	row, err := r.q(ctx).CreateVault(ctx, gen.CreateVaultParams{
		UserID:          userID,
		WrappedVaultKey: v.WrappedVaultKey,
		EncName:         v.EncName,
	})
	if err != nil {
		return domain.Vault{}, fmt.Errorf("create vault: %w", err)
	}

	v.ID = uuidToString(row.ID)
	v.Version = row.Version
	v.CreatedAt = row.CreatedAt.Time
	v.UpdatedAt = row.UpdatedAt.Time
	return v, nil
}

func (r *VaultRepo) ListByUser(ctx context.Context, userID string) ([]domain.Vault, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("list vaults: invalid user id: %w", err)
	}

	rows, err := r.q(ctx).ListVaultsByUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("list vaults by user: %w", err)
	}

	vaults := make([]domain.Vault, 0, len(rows))
	for _, row := range rows {
		vaults = append(vaults, domain.Vault{
			ID:              uuidToString(row.ID),
			UserID:          userID,
			WrappedVaultKey: row.WrappedVaultKey,
			EncName:         row.EncName,
			Version:         row.Version,
		})
	}
	return vaults, nil
}
