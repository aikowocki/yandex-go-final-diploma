package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

// CreateLoginPassword шифрует поля VaultKey'ом открытой папки и создаёт секрет. Если сервер
// недоступен (оффлайн), операция кладётся в outbox, а секрет сохраняется в локальном кеше
// с временным id и флагом dirty; пользователю возвращается успех (оптимистично).
func (u *UseCase) CreateLoginPassword(ctx context.Context, vaultID string, input CreateLoginPasswordInput) (string, error) {
	if input.Title == "" {
		return "", ErrEmptyTitle
	}

	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return "", err
	}

	row := secretcontent.LoginPasswordRow{
		V:        secretcontent.LoginPasswordSchemaV1,
		Title:    input.Title,
		Tags:     input.Tags,
		URI:      input.URI,
		Username: input.Username,
	}
	index := secretcontent.LoginPasswordIndex{
		V:            secretcontent.LoginPasswordSchemaV1,
		Note:         input.Note,
		CustomFields: input.CustomFields,
	}
	payload := secretcontent.LoginPasswordPayload{
		V:        secretcontent.LoginPasswordSchemaV1,
		Password: input.Password,
	}

	encRow, err := u.cipher.EncryptStruct(vaultKey, row)
	if err != nil {
		return "", fmt.Errorf("encrypt row: %w", err)
	}
	encIndex, err := u.cipher.EncryptStruct(vaultKey, index)
	if err != nil {
		return "", fmt.Errorf("encrypt index: %w", err)
	}
	encPayload, err := u.cipher.EncryptStruct(vaultKey, payload)
	if err != nil {
		return "", fmt.Errorf("encrypt payload: %w", err)
	}

	secretType := int32(domain.SecretTypeLoginPassword)

	id, err := u.server.CreateSecret(ctx, token, vaultID, secretType, encRow, encIndex, encPayload)
	if err != nil {
		if errors.Is(err, grpcclient.ErrUnavailable) {
			return u.createOffline(ctx, vaultID, secretType, encRow, encIndex, encPayload)
		}
		return "", err
	}

	// Онлайн-успех: кладём секрет в локальный кеш (payload уже известен → payload_loaded=1).
	if err := u.cacheCreated(ctx, id, vaultID, secretType, encRow, encIndex, encPayload, false); err != nil {
		return "", err
	}
	return id, nil
}

// createOffline сохраняет секрет локально с временным id и ставит операцию в outbox.
func (u *UseCase) createOffline(ctx context.Context, vaultID string, secretType int32, encRow, encIndex, encPayload []byte) (string, error) {
	tempID := uuid.NewString()

	if err := u.cacheCreated(ctx, tempID, vaultID, secretType, encRow, encIndex, encPayload, true); err != nil {
		return "", err
	}

	body, err := json.Marshal(contracts.OutboxSecretCreate{
		VaultID:    vaultID,
		TempID:     tempID,
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
		EntityID: tempID,
		Payload:  body,
	}); err != nil {
		return "", err
	}
	return tempID, nil
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
		Version:       1,
		IndexLoaded:   len(encIndex) > 0,
		PayloadLoaded: len(encPayload) > 0,
		Dirty:         dirty,
	})
}
