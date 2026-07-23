package secret

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

func (in CreateBankCardInput) toRow() secretcontent.BankCardRow {
	last4 := in.PAN
	if len(last4) > 4 {
		last4 = last4[len(last4)-4:]
	}
	return secretcontent.BankCardRow{V: secretcontent.BankCardSchemaV1, Title: in.Title, Tags: in.Tags, Last4: last4}
}

func (in CreateBankCardInput) toIndex() secretcontent.BankCardIndex {
	return secretcontent.BankCardIndex{
		V: secretcontent.BankCardSchemaV1, Bank: in.Bank, Cardholder: in.Cardholder,
		Brand: in.Brand, Expiry: in.Expiry, Note: in.Note, CustomFields: in.CustomFields,
	}
}

func (in CreateBankCardInput) toPayload() secretcontent.BankCardPayload {
	return secretcontent.BankCardPayload{V: secretcontent.BankCardSchemaV1, PAN: in.PAN, CVV: in.CVV, PIN: in.PIN, OTPCodes: in.OTPCodes}
}

// CreateBankCard создаёт новый секрет типа «банковская карта».
func (u *UseCase) CreateBankCard(ctx context.Context, vaultID string, input CreateBankCardInput) (string, error) {
	if input.Title == "" {
		return "", ErrEmptyTitle
	}
	return createTyped(ctx, u, vaultID, int32(domain.SecretTypeBankCard), input.toRow(), input.toIndex(), input.toPayload())
}

// UpdateBankCard обновляет существующий секрет.
func (u *UseCase) UpdateBankCard(ctx context.Context, vaultID, secretID string, baseVersion int64, input CreateBankCardInput) (*GenericConflict, error) {
	if input.Title == "" {
		return nil, ErrEmptyTitle
	}
	if secretID == "" {
		return nil, ErrEmptySecretID
	}
	return updateTyped(ctx, u, vaultID, secretID, baseVersion, int32(domain.SecretTypeBankCard), input.toRow(), input.toIndex(), input.toPayload())
}

// ListBankCardRows возвращает строки (row-тир) всех секретов типа «банковская карта» в папке.
func (u *UseCase) ListBankCardRows(ctx context.Context, vaultID string) ([]TypedRow[secretcontent.BankCardRow], error) {
	return listRowsTyped[secretcontent.BankCardRow](ctx, u, vaultID, int32(domain.SecretTypeBankCard))
}

// GetBankCardDetail возвращает полностью расшифрованный секрет.
func (u *UseCase) GetBankCardDetail(ctx context.Context, vaultID, secretID string) (TypedDetail[secretcontent.BankCardRow, secretcontent.BankCardIndex, secretcontent.BankCardPayload], error) {
	return getDetailTyped[secretcontent.BankCardRow, secretcontent.BankCardIndex, secretcontent.BankCardPayload](ctx, u, vaultID, secretID)
}
