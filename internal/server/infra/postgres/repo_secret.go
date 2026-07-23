package postgres

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres/gen"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
	"github.com/jackc/pgx/v5/pgtype"
)

// SecretRepo реализует secret.Repository поверх sqlc-запросов.
type SecretRepo struct {
	baseRepo
}

var _ secret.Repository = (*SecretRepo)(nil)

// NewSecretRepo создаёт SecretRepo поверх переданного пула соединений.
func NewSecretRepo(db *DB) *SecretRepo {
	return &SecretRepo{baseRepo{db: db}}
}

// Create создаёт новый секрет в указанной папке.
func (r *SecretRepo) Create(ctx context.Context, s domain.Secret) (domain.Secret, error) {
	vaultID, err := parseUUID(s.VaultID)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("create secret: invalid vault id: %w", err)
	}
	secretID, err := parseUUID(s.ID)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("create secret: invalid secret id: %w", err)
	}

	row, err := r.q(ctx).CreateSecret(ctx, gen.CreateSecretParams{
		ID:         secretID,
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

// GetForUpdate читает полную строку секрета под блокировкой (FOR UPDATE) внутри транзакции.
func (r *SecretRepo) GetForUpdate(ctx context.Context, secretID, userID string) (domain.Secret, error) {
	sid, err := parseUUIDOr(secretID, secret.ErrSecretNotFound)
	if err != nil {
		return domain.Secret{}, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("get secret for update: invalid user id: %w", err)
	}

	row, err := r.q(ctx).GetSecretForUpdate(ctx, gen.GetSecretForUpdateParams{ID: sid, UserID: uid})
	if err != nil {
		return domain.Secret{}, wrapNotFound(err, secret.ErrSecretNotFound, "get secret for update")
	}

	sec := domain.Secret{
		ID:         uuidToString(row.ID),
		VaultID:    uuidToString(row.VaultID),
		Type:       domain.SecretType(row.Type),
		EncRow:     row.EncRow,
		EncIndex:   row.EncIndex,
		EncPayload: row.EncPayload,
		Version:    row.Version,
		Deleted:    row.Deleted,
	}
	if row.BlobRef.Valid {
		sec.BlobRef = &row.BlobRef.String
	}
	if row.BlobSize.Valid {
		sec.BlobSize = &row.BlobSize.Int64
	}
	return sec, nil
}

// UpdateFields применяет новые шифротексты и инкрементирует версию.
func (r *SecretRepo) UpdateFields(ctx context.Context, secretID string, encRow, encIndex, encPayload []byte) (int64, error) {
	sid, err := parseUUIDOr(secretID, secret.ErrSecretNotFound)
	if err != nil {
		return 0, err
	}

	version, err := r.q(ctx).UpdateSecretFields(ctx, gen.UpdateSecretFieldsParams{
		ID:         sid,
		EncRow:     encRow,
		EncIndex:   encIndex,
		EncPayload: encPayload,
	})
	if err != nil {
		return 0, fmt.Errorf("update secret fields: %w", err)
	}
	return version, nil
}

// SoftDelete помечает секрет удалённым и инкрементирует версию.
func (r *SecretRepo) SoftDelete(ctx context.Context, secretID string) (int64, error) {
	sid, err := parseUUIDOr(secretID, secret.ErrSecretNotFound)
	if err != nil {
		return 0, err
	}

	version, err := r.q(ctx).SoftDeleteSecret(ctx, sid)
	if err != nil {
		return 0, fmt.Errorf("soft delete secret: %w", err)
	}
	return version, nil
}

// AttachBlob прописывает blob_ref/blob_size и инкрементирует версию.
func (r *SecretRepo) AttachBlob(ctx context.Context, secretID, blobRef string, blobSize int64) (int64, error) {
	sid, err := parseUUIDOr(secretID, secret.ErrSecretNotFound)
	if err != nil {
		return 0, err
	}

	version, err := r.q(ctx).AttachBlob(ctx, gen.AttachBlobParams{
		ID:       sid,
		BlobRef:  pgtype.Text{String: blobRef, Valid: true},
		BlobSize: pgtype.Int8{Int64: blobSize, Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("attach blob: %w", err)
	}
	return version, nil
}

// BumpVaultVersion инкрементирует версию папки (сигнал sync).
func (r *SecretRepo) BumpVaultVersion(ctx context.Context, vaultID string) error {
	vid, err := parseUUID(vaultID)
	if err != nil {
		return fmt.Errorf("bump vault version: invalid vault id: %w", err)
	}
	if err := r.q(ctx).BumpVaultVersion(ctx, vid); err != nil {
		return fmt.Errorf("bump vault version: %w", err)
	}
	return nil
}

// ListRow возвращает все секреты папки в виде "строк" (шифрованный EncRow) для синхронизации.
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

// ListIndex возвращает индексные записи (EncIndex) секретов папки для клиентского поиска.
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

// GetPayload возвращает зашифрованный payload секрета, принадлежащего пользователю.
func (r *SecretRepo) GetPayload(ctx context.Context, secretID, userID string) (domain.Secret, error) {
	// Невалидный id → секрет не найден (не раскрываем детали).
	sid, err := parseUUIDOr(secretID, secret.ErrSecretNotFound)
	if err != nil {
		return domain.Secret{}, err
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("get secret payload: invalid user id: %w", err)
	}

	row, err := r.q(ctx).GetSecretPayload(ctx, gen.GetSecretPayloadParams{ID: sid, UserID: uid})
	if err != nil {
		return domain.Secret{}, wrapNotFound(err, secret.ErrSecretNotFound, "get secret payload")
	}

	return domain.Secret{
		ID:         uuidToString(row.ID),
		Type:       domain.SecretType(row.Type),
		Version:    row.Version,
		EncPayload: row.EncPayload,
	}, nil
}
