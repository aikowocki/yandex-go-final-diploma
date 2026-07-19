package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres/gen"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
)

// RecoveryRepo реализует хранение recovery codes.
type RecoveryRepo struct {
	db *DB
}

// NewRecoveryRepo создаёт RecoveryRepo поверх переданного пула соединений.
func NewRecoveryRepo(db *DB) *RecoveryRepo {
	return &RecoveryRepo{db: db}
}

func (r *RecoveryRepo) q(ctx context.Context) *gen.Queries {
	return gen.New(r.db.querier(ctx))
}

// StoreCode сохраняет зашифрованный master key под recovery-кодом пользователя.
func (r *RecoveryRepo) StoreCode(ctx context.Context, userID, codeID string, encMasterKey []byte) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return auth.ErrUserNotFound
	}
	return r.q(ctx).StoreRecoveryCode(ctx, gen.StoreRecoveryCodeParams{
		UserID:       uid,
		CodeID:       codeID,
		EncMasterKey: encMasterKey,
	})
}

// GetBlob возвращает зашифрованный master key по recovery-коду, если он ещё не использован.
func (r *RecoveryRepo) GetBlob(ctx context.Context, userID, codeID string) ([]byte, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, auth.ErrUserNotFound
	}
	blob, err := r.q(ctx).GetRecoveryBlob(ctx, gen.GetRecoveryBlobParams{
		UserID: uid,
		CodeID: codeID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("recovery code not found or already used")
	}
	if err != nil {
		return nil, fmt.Errorf("get recovery blob: %w", err)
	}
	return blob, nil
}

// MarkUsed помечает recovery-код использованным, чтобы его нельзя было применить повторно.
func (r *RecoveryRepo) MarkUsed(ctx context.Context, userID, codeID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return auth.ErrUserNotFound
	}
	return r.q(ctx).MarkRecoveryCodeUsed(ctx, gen.MarkRecoveryCodeUsedParams{
		UserID: uid,
		CodeID: codeID,
	})
}

// DeleteAll удаляет все recovery-коды пользователя.
func (r *RecoveryRepo) DeleteAll(ctx context.Context, userID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return auth.ErrUserNotFound
	}
	return r.q(ctx).DeleteUserRecoveryCodes(ctx, uid)
}
