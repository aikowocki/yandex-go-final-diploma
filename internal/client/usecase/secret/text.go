package secret

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

func (in CreateTextInput) toRow() secretcontent.TextRow {
	return secretcontent.TextRow{V: secretcontent.TextSchemaV1, Title: in.Title, Tags: in.Tags}
}

func (in CreateTextInput) toIndex() secretcontent.TextIndex {
	return secretcontent.TextIndex{V: secretcontent.TextSchemaV1, Note: in.Note, CustomFields: in.CustomFields}
}

func (in CreateTextInput) toPayload() secretcontent.TextPayload {
	return secretcontent.TextPayload{V: secretcontent.TextSchemaV1, Body: in.Body, OTPCodes: in.OTPCodes}
}

// CreateText создаёт секрет типа text (произвольная текстовая заметка).
func (u *UseCase) CreateText(ctx context.Context, vaultID string, input CreateTextInput) (string, error) {
	if input.Title == "" {
		return "", ErrEmptyTitle
	}
	return createTyped(ctx, u, vaultID, int32(domain.SecretTypeText), input.toRow(), input.toIndex(), input.toPayload())
}

// UpdateText обновляет секрет типа text с оптимистичной блокировкой.
func (u *UseCase) UpdateText(ctx context.Context, vaultID, secretID string, baseVersion int64, input CreateTextInput) (*GenericConflict, error) {
	if input.Title == "" {
		return nil, ErrEmptyTitle
	}
	if secretID == "" {
		return nil, ErrEmptySecretID
	}
	return updateTyped(ctx, u, vaultID, secretID, baseVersion, int32(domain.SecretTypeText), input.toRow(), input.toIndex(), input.toPayload())
}

// ListTextRows возвращает Tier 2a-строки секретов типа text из локального кеша.
func (u *UseCase) ListTextRows(ctx context.Context, vaultID string) ([]TypedRow[secretcontent.TextRow], error) {
	return listRowsTyped[secretcontent.TextRow](ctx, u, vaultID, int32(domain.SecretTypeText))
}

// GetTextDetail собирает полную карточку секрета типа text.
func (u *UseCase) GetTextDetail(ctx context.Context, vaultID, secretID string) (TypedDetail[secretcontent.TextRow, secretcontent.TextIndex, secretcontent.TextPayload], error) {
	return getDetailTyped[secretcontent.TextRow, secretcontent.TextIndex, secretcontent.TextPayload](ctx, u, vaultID, secretID)
}
