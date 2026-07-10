package postgres

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres/gen"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
)

// SecretRepo реализует secret.Repository поверх sqlc-запросов.
type SecretRepo struct {
	db *DB
}

var _ secret.Repository = (*SecretRepo)(nil)

func NewSecretRepo(db *DB) *SecretRepo {
	return &SecretRepo{db: db}
}

func (r *SecretRepo) q(ctx context.Context) *gen.Queries {
	return gen.New(r.db.querier(ctx))
}

func (r *SecretRepo) Create(ctx context.Context, s domain.Secret) (domain.Secret, error) {
	vaultID, err := parseUUID(s.VaultID)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("create secret: invalid vault id: %w", err)
	}

	row, err := r.q(ctx).CreateSecret(ctx, gen.CreateSecretParams{
		VaultID:    vaultID,
		Type:       int16(s.Type),
		EncRow:     s.EncRow,
		EncIndex:   s.EncIndex,
		EncPayload: s.EncPayload,
	})
	if err != nil {
		return domain.Secret{}, fmt.Errorf("create secret: %w", err)
	}

	s.ID = uuidToString(row.ID)
	s.Version = row.Version
	s.CreatedAt = row.CreatedAt.Time
	s.UpdatedAt = row.UpdatedAt.Time
	return s, nil
}

func (r *SecretRepo) ListRow(ctx context.Context, vaultID, userID string) ([]domain.Secret, error) {
	vid, err := parseUUID(vaultID)
	if err != nil {
		return nil, fmt.Errorf("list secret rows: invalid vault id: %w", err)
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("list secret rows: invalid user id: %w", err)
	}

	rows, err := r.q(ctx).ListSecretRows(ctx, gen.ListSecretRowsParams{VaultID: vid, UserID: uid})
	if err != nil {
		return nil, fmt.Errorf("list secret rows: %w", err)
	}

	secrets := make([]domain.Secret, 0, len(rows))
	for _, row := range rows {
		secrets = append(secrets, domain.Secret{
			ID:      uuidToString(row.ID),
			VaultID: vaultID,
			Type:    domain.SecretType(row.Type),
			Version: row.Version,
			EncRow:  row.EncRow,
		})
	}
	return secrets, nil
}

func (r *SecretRepo) ListIndex(ctx context.Context, vaultID, userID string) ([]domain.Secret, error) {
	vid, err := parseUUID(vaultID)
	if err != nil {
		return nil, fmt.Errorf("list secret index: invalid vault id: %w", err)
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("list secret index: invalid user id: %w", err)
	}

	rows, err := r.q(ctx).ListSecretIndex(ctx, gen.ListSecretIndexParams{VaultID: vid, UserID: uid})
	if err != nil {
		return nil, fmt.Errorf("list secret index: %w", err)
	}

	secrets := make([]domain.Secret, 0, len(rows))
	for _, row := range rows {
		secrets = append(secrets, domain.Secret{
			ID:       uuidToString(row.ID),
			VaultID:  vaultID,
			Version:  row.Version,
			EncIndex: row.EncIndex,
		})
	}
	return secrets, nil
}

func (r *SecretRepo) GetPayload(ctx context.Context, secretID, userID string) (domain.Secret, error) {
	sid, err := parseUUID(secretID)
	if err != nil {
		// Невалидный id → секрет не найден (не раскрываем детали).
		return domain.Secret{}, secret.ErrSecretNotFound
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("get secret payload: invalid user id: %w", err)
	}

	row, err := r.q(ctx).GetSecretPayload(ctx, gen.GetSecretPayloadParams{ID: sid, UserID: uid})
	if err != nil {
		if isNoRows(err) {
			return domain.Secret{}, secret.ErrSecretNotFound
		}
		return domain.Secret{}, fmt.Errorf("get secret payload: %w", err)
	}

	return domain.Secret{
		ID:         uuidToString(row.ID),
		Type:       domain.SecretType(row.Type),
		Version:    row.Version,
		EncPayload: row.EncPayload,
	}, nil
}
