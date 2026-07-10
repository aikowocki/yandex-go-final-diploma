package secret

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

// CreateLoginPassword собирает структуры из plaintext-полей, шифрует каждую VaultKey'ом
// открытой папки и отправляет CreateSecret RPC. Возвращает id созданного секрета.
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

	return u.server.CreateSecret(ctx, token, vaultID, int32(domain.SecretTypeLoginPassword), encRow, encIndex, encPayload)
}
