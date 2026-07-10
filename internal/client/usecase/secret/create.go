package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

// createVersion — версия только что созданного секрета (для AAD и локального кеша).
const createVersion int64 = 1

// CreateLoginPassword шифрует поля VaultKey'ом открытой папки и создаёт секрет. id секрета
// генерируется клиентом (нужен для AAD-привязки шифротекста). Если сервер недоступен (оффлайн),
// операция кладётся в outbox, секрет сохраняется в кеше с флагом dirty; пользователю возвращается
// успех (оптимистично).
func (u *UseCase) CreateLoginPassword(ctx context.Context, vaultID string, input CreateLoginPasswordInput) (string, error) {
	if input.Title == "" {
		return "", ErrEmptyTitle
	}

	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return "", err
	}

	secretID := uuid.NewString()
	secretType := int32(domain.SecretTypeLoginPassword)

	encRow, encIndex, encPayload, err := u.encryptLoginPassword(vaultKey, vaultID, secretID, createVersion, input)
	if err != nil {
		return "", err
	}

	if err := u.server.CreateSecret(ctx, token, secretID, vaultID, secretType, encRow, encIndex, encPayload); err != nil {
		if errors.Is(err, grpcclient.ErrUnavailable) {
			return u.createOffline(ctx, secretID, vaultID, secretType, encRow, encIndex, encPayload)
		}
		return "", err
	}

	// Онлайн-успех: кладём секрет в локальный кеш (payload уже известен → payload_loaded=1).
	if err := u.cacheCreated(ctx, secretID, vaultID, secretType, encRow, encIndex, encPayload, false); err != nil {
		return "", err
	}
	return secretID, nil
}

// encryptLoginPassword шифрует три тира секрета типа login/password с AAD-контекстом
// (vault_id|secret_id|version|tier). version — та версия, которую строка получит на сервере.
func (u *UseCase) encryptLoginPassword(vaultKey []byte, vaultID, secretID string, version int64, input CreateLoginPasswordInput) (encRow, encIndex, encPayload []byte, err error) {
	encRow, err = u.cipher.EncryptStruct(vaultKey, secretAAD(vaultID, secretID, version, tierRow), input.toRow())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encrypt row: %w", err)
	}
	encIndex, err = u.cipher.EncryptStruct(vaultKey, secretAAD(vaultID, secretID, version, tierIndex), input.toIndex())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encrypt index: %w", err)
	}
	encPayload, err = u.cipher.EncryptStruct(vaultKey, secretAAD(vaultID, secretID, version, tierPayload), input.toPayload())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encrypt payload: %w", err)
	}
	return encRow, encIndex, encPayload, nil
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
