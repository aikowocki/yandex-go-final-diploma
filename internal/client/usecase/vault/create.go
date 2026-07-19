package vault

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

// Create создаёт папку: генерирует VaultKey, оборачивает его MasterKey'ом, шифрует имя,
// отправляет на сервер только обёртки. Открытый VaultKey кладётся в сессию.
func (u *UseCase) Create(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", ErrEmptyName
	}

	masterKey, ok := u.sess.MasterKey()
	if !ok {
		return "", ErrLocked
	}

	vaultKey, err := u.cipher.GenerateVaultKey()
	if err != nil {
		return "", fmt.Errorf("generate vault key: %w", err)
	}

	wrapped, err := u.cipher.WrapVaultKey(vaultKey, masterKey)
	if err != nil {
		return "", fmt.Errorf("wrap vault key: %w", err)
	}

	encName, err := u.cipher.EncryptStruct(vaultKey, nil, name)
	if err != nil {
		return "", fmt.Errorf("encrypt name: %w", err)
	}

	token, err := u.accessToken()
	if err != nil {
		return "", err
	}

	id, err := u.server.CreateVault(ctx, token, wrapped, encName)
	if err != nil {
		return "", err
	}

	u.sess.OpenVault(id, vaultKey)

	// Кешируем новую папку локально, чтобы последующие secret-команды могли открыть его оффлайн.
	if err := u.local.UpsertVault(ctx, contracts.LocalVault{
		ID:              id,
		WrappedVaultKey: wrapped,
		EncName:         encName,
		Version:         1,
		SyncEnabled:     true, // vault созданный пользователем — синхронизируется по умолчанию
	}); err != nil {
		return "", fmt.Errorf("cache vault: %w", err)
	}
	return id, nil
}
