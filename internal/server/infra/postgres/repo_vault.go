package postgres

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres/gen"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault"
)

// VaultRepo реализует vault.Repository поверх sqlc-запросов.
type VaultRepo struct {
	db *DB
}

var (
	_ vault.Repository      = (*VaultRepo)(nil)
	_ secret.VaultOwnership = (*VaultRepo)(nil)
)

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

// IsOwner проверяет, принадлежит ли папка пользователю.
func (r *VaultRepo) IsOwner(ctx context.Context, vaultID, userID string) (bool, error) {
	vid, err := parseUUID(vaultID)
	if err != nil {
		return false, nil
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return false, fmt.Errorf("is owner: invalid user id: %w", err)
	}

	owns, err := r.q(ctx).VaultBelongsToUser(ctx, gen.VaultBelongsToUserParams{ID: vid, UserID: uid})
	if err != nil {
		return false, fmt.Errorf("is owner: %w", err)
	}
	return owns, nil
}

func (r *VaultRepo) CheckFreshness(ctx context.Context, userID string) ([]vault.Version, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("check freshness: invalid user id: %w", err)
	}

	rows, err := r.q(ctx).CheckVaultFreshness(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("check freshness: %w", err)
	}

	versions := make([]vault.Version, 0, len(rows))
	for _, row := range rows {
		versions = append(versions, vault.Version{
			ID:      uuidToString(row.ID),
			Version: row.Version,
		})
	}
	return versions, nil
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
