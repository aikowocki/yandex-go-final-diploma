package auth

import (
	"context"
	"fmt"
)

// RecoveryCodeEntry — одна запись recovery code (code_id + зашифрованный MasterKey).
type RecoveryCodeEntry struct {
	CodeID       string
	EncMasterKey []byte
}

// StoreRecoveryCodes сохраняет набор recovery codes для пользователя (заменяет предыдущие).
func (u *UseCase) StoreRecoveryCodes(ctx context.Context, userID string, codes []RecoveryCodeEntry) error {
	if userID == "" {
		return ErrEmptyUserID
	}
	// Удаляем старые коды (перегенерация).
	if err := u.recovery.DeleteAll(ctx, userID); err != nil {
		return fmt.Errorf("delete old recovery codes: %w", err)
	}
	for _, c := range codes {
		if err := u.recovery.StoreCode(ctx, userID, c.CodeID, c.EncMasterKey); err != nil {
			return fmt.Errorf("store recovery code: %w", err)
		}
	}
	return nil
}

// GetRecoveryBlob возвращает зашифрованный MasterKey для указанного code_id (если не использован).
func (u *UseCase) GetRecoveryBlob(ctx context.Context, userID, codeID string) ([]byte, error) {
	if userID == "" {
		return nil, ErrEmptyUserID
	}
	return u.recovery.GetBlob(ctx, userID, codeID)
}

// MarkRecoveryCodeUsed помечает код как использованный (одноразовый).
func (u *UseCase) MarkRecoveryCodeUsed(ctx context.Context, userID, codeID string) error {
	if userID == "" {
		return ErrEmptyUserID
	}
	return u.recovery.MarkUsed(ctx, userID, codeID)
}
