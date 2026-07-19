package secret

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// UpdateLoginPassword обновляет секрет типа login/password с оптимистичной блокировкой.
func (u *UseCase) UpdateLoginPassword(ctx context.Context, vaultID, secretID string, baseVersion int64, input CreateLoginPasswordInput) (*GenericConflict, error) {
	if input.Title == "" {
		return nil, ErrEmptyTitle
	}
	if secretID == "" {
		return nil, ErrEmptySecretID
	}
	return updateTyped(ctx, u, vaultID, secretID, baseVersion, int32(domain.SecretTypeLoginPassword), input.toRow(), input.toIndex(), input.toPayload())
}

// updateOffline кладёт операцию update в outbox и кеширует секрет как dirty.
func (u *UseCase) updateOffline(ctx context.Context, secretID, vaultID string, secretType int32, baseVersion int64, encRow, encIndex, encPayload []byte) error {
	if err := u.cacheFullSecret(ctx, secretID, vaultID, secretType, encRow, encIndex, encPayload, baseVersion+1, true); err != nil {
		return err
	}

	body, err := json.Marshal(contracts.OutboxSecretUpdate{
		SecretID:    secretID,
		VaultID:     vaultID,
		BaseVersion: baseVersion,
		Type:        secretType,
		EncRow:      encRow,
		EncIndex:    encIndex,
		EncPayload:  encPayload,
	})
	if err != nil {
		return fmt.Errorf("encode outbox update: %w", err)
	}
	_, err = u.local.EnqueueOutbox(ctx, contracts.OutboxEntry{
		Op:          contracts.OutboxOpUpdate,
		Entity:      "secret",
		EntityID:    secretID,
		BaseVersion: baseVersion,
		Payload:     body,
	})
	return err
}

// deleteOffline кладёт операцию delete в outbox и помечает секрет удалённым в кеше (dirty).
func (u *UseCase) deleteOffline(ctx context.Context, secretID, vaultID string, baseVersion int64) error {
	body, err := json.Marshal(contracts.OutboxSecretDelete{
		SecretID:    secretID,
		VaultID:     vaultID,
		BaseVersion: baseVersion,
	})
	if err != nil {
		return fmt.Errorf("encode outbox delete: %w", err)
	}
	if _, err := u.local.EnqueueOutbox(ctx, contracts.OutboxEntry{
		Op:          contracts.OutboxOpDelete,
		Entity:      "secret",
		EntityID:    secretID,
		BaseVersion: baseVersion,
		Payload:     body,
	}); err != nil {
		return err
	}
	// Локально прячем секрет (soft): убираем из кеша, чтобы список не показывал его до синка.
	return u.local.DeleteSecret(ctx, secretID)
}

// cacheFullSecret сохраняет все три тира секрета в локальный кеш с указанной версией.
func (u *UseCase) cacheFullSecret(ctx context.Context, id, vaultID string, secretType int32, encRow, encIndex, encPayload []byte, version int64, dirty bool) error {
	if err := u.local.UpsertSecretRow(ctx, contracts.LocalSecret{
		ID:            id,
		VaultID:       vaultID,
		Type:          secretType,
		EncRow:        encRow,
		EncIndex:      encIndex,
		EncPayload:    encPayload,
		Version:       version,
		IndexLoaded:   len(encIndex) > 0,
		PayloadLoaded: len(encPayload) > 0,
		Dirty:         dirty,
	}); err != nil {
		return err
	}
	// UpsertSecretRow при обновлении существующей строки не трогает enc_index/enc_payload —
	// выставляем их явно, чтобы кеш отражал новую версию всех тиров.
	if len(encIndex) > 0 {
		if err := u.local.SetSecretIndex(ctx, id, encIndex, version); err != nil {
			return err
		}
	}
	if len(encPayload) > 0 {
		if err := u.local.SetSecretPayload(ctx, id, encPayload, version); err != nil {
			return err
		}
	}
	return nil
}

// --- helpers для сборки типизированных структур из ввода ---

func (in CreateLoginPasswordInput) toRow() secretcontent.LoginPasswordRow {
	return secretcontent.LoginPasswordRow{
		V:        secretcontent.LoginPasswordSchemaV1,
		Title:    in.Title,
		Tags:     in.Tags,
		URI:      in.URI,
		Username: in.Username,
	}
}

func (in CreateLoginPasswordInput) toIndex() secretcontent.LoginPasswordIndex {
	return secretcontent.LoginPasswordIndex{
		V:            secretcontent.LoginPasswordSchemaV1,
		Note:         in.Note,
		CustomFields: in.CustomFields,
	}
}

func (in CreateLoginPasswordInput) toPayload() secretcontent.LoginPasswordPayload {
	return secretcontent.LoginPasswordPayload{
		V:        secretcontent.LoginPasswordSchemaV1,
		Password: in.Password,
		OTPCodes: in.OTPCodes,
	}
}
