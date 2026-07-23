package secret

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

func (in CreateTOTPInput) toRow() secretcontent.TOTPRow {
	return secretcontent.TOTPRow{V: secretcontent.TOTPSchemaV1, Title: in.Title, Tags: in.Tags, Issuer: in.Issuer}
}

func (in CreateTOTPInput) toIndex() secretcontent.TOTPIndex {
	return secretcontent.TOTPIndex{V: secretcontent.TOTPSchemaV1, Account: in.Account, Note: in.Note, CustomFields: in.CustomFields}
}

func (in CreateTOTPInput) toPayload() secretcontent.TOTPPayload {
	return secretcontent.TOTPPayload{
		V: secretcontent.TOTPSchemaV1, Secret: in.Secret, Algo: in.Algo,
		Digits: in.Digits, Period: in.Period, OTPCodes: in.OTPCodes,
	}
}

// CreateTOTP создаёт секрет типа totp (authenticator-запись, E2E — сервер секрет не видит).
func (u *UseCase) CreateTOTP(ctx context.Context, vaultID string, input CreateTOTPInput) (string, error) {
	if input.Title == "" {
		return "", ErrEmptyTitle
	}
	if input.Secret == "" {
		return "", ErrEmptyTOTPSecret
	}
	return createTyped(ctx, u, vaultID, int32(domain.SecretTypeTOTP), input.toRow(), input.toIndex(), input.toPayload())
}

// UpdateTOTP обновляет секрет типа totp с оптимистичной блокировкой.
func (u *UseCase) UpdateTOTP(ctx context.Context, vaultID, secretID string, baseVersion int64, input CreateTOTPInput) (*GenericConflict, error) {
	if input.Title == "" {
		return nil, ErrEmptyTitle
	}
	if input.Secret == "" {
		return nil, ErrEmptyTOTPSecret
	}
	if secretID == "" {
		return nil, ErrEmptySecretID
	}
	return updateTyped(ctx, u, vaultID, secretID, baseVersion, int32(domain.SecretTypeTOTP), input.toRow(), input.toIndex(), input.toPayload())
}

// ListTOTPRows возвращает Tier 2a-строки секретов типа totp из локального кеша.
func (u *UseCase) ListTOTPRows(ctx context.Context, vaultID string) ([]TypedRow[secretcontent.TOTPRow], error) {
	return listRowsTyped[secretcontent.TOTPRow](ctx, u, vaultID, int32(domain.SecretTypeTOTP))
}

// GetTOTPDetail собирает полную карточку секрета типа totp.
func (u *UseCase) GetTOTPDetail(ctx context.Context, vaultID, secretID string) (TypedDetail[secretcontent.TOTPRow, secretcontent.TOTPIndex, secretcontent.TOTPPayload], error) {
	return getDetailTyped[secretcontent.TOTPRow, secretcontent.TOTPIndex, secretcontent.TOTPPayload](ctx, u, vaultID, secretID)
}
