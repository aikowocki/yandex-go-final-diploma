package secret

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
)

// createVersion — версия только что созданного секрета (для AAD и локального кеша).
const createVersion int64 = 1

// CreateLoginPassword создаёт секрет типа login/password.
func (u *UseCase) CreateLoginPassword(ctx context.Context, vaultID string, input CreateLoginPasswordInput) (string, error) {
	if input.Title == "" {
		return "", ErrEmptyTitle
	}
	return createTyped(ctx, u, vaultID, int32(domain.SecretTypeLoginPassword), input.toRow(), input.toIndex(), input.toPayload())
}

// createOffline сохраняет секрет локально (dirty) и ставит операцию create в outbox.
// id стабилен (client-generated), поэтому reconcile по временному id не требуется.
func (u *UseCase) createOffline(ctx context.Context, secretID, vaultID string, secretType int32, encRow, encIndex, encPayload []byte) (string, error) {
	if err := u.cacheCreated(ctx, secretID, vaultID, secretType, encRow, encIndex, encPayload, true); err != nil {
		return "", err
	}

	body, err := json.Marshal(contracts.OutboxSecretCreate{
		SecretID:   secretID,
		VaultID:    vaultID,
		Type:       secretType,
		EncRow:     encRow,
		EncIndex:   encIndex,
		EncPayload: encPayload,
	})
	if err != nil {
		return "", fmt.Errorf("encode outbox payload: %w", err)
	}
	if _, err := u.local.EnqueueOutbox(ctx, contracts.OutboxEntry{
		Op:       contracts.OutboxOpCreate,
		Entity:   "secret",
		EntityID: secretID,
		Payload:  body,
	}); err != nil {
		return "", err
	}
	return secretID, nil
}

// cacheCreated сохраняет только что созданный секрет в локальный кеш.
func (u *UseCase) cacheCreated(ctx context.Context, id, vaultID string, secretType int32, encRow, encIndex, encPayload []byte, dirty bool) error {
	return u.local.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID:            id,
		VaultID:       vaultID,
		Type:          secretType,
		EncRow:        encRow,
		EncIndex:      encIndex,
		EncPayload:    encPayload,
		Version:       createVersion,
		IndexLoaded:   len(encIndex) > 0,
		PayloadLoaded: len(encPayload) > 0,
		Dirty:         dirty,
	})
}
