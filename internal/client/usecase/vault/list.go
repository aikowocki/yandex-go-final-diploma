package vault

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

// List возвращает расшифрованные папки пользователя. Для каждой разворачивает VaultKey
// MasterKey'ом, расшифровывает имя и открывает папку в сессии (VaultKey понадобится для секретов).
func (u *UseCase) List(ctx context.Context) ([]DecryptedVault, error) {
	masterKey, ok := u.sess.MasterKey()
	if !ok {
		return nil, ErrLocked
	}

	token, err := u.accessToken()
	if err != nil {
		return nil, err
	}

	items, err := u.server.ListVaults(ctx, token)
	if err != nil {
		return nil, err
	}

	result := make([]DecryptedVault, 0, len(items))
	for _, it := range items {
		vaultKey, err := u.cipher.UnwrapVaultKey(it.WrappedVaultKey, masterKey)
		if err != nil {
			return nil, fmt.Errorf("unwrap vault key: %w", err)
		}

		var name string
		if err := u.cipher.DecryptStruct(vaultKey, it.EncName, &name); err != nil {
			return nil, fmt.Errorf("decrypt vault name: %w", err)
		}

		u.sess.OpenVault(it.ID, vaultKey)

		// Кешируем метаданные папки локально (шифротексты), чтобы secret-команды и sync
		// могли работать оффлайн. synced_version при этом не трогается (им управляет sync).
		if err := u.local.UpsertVault(ctx, contracts.LocalVault{
			ID:              it.ID,
			WrappedVaultKey: it.WrappedVaultKey,
			EncName:         it.EncName,
			Version:         it.Version,
		}); err != nil {
			return nil, fmt.Errorf("cache vault: %w", err)
		}

		result = append(result, DecryptedVault{ID: it.ID, Name: name, Version: it.Version})
	}
	return result, nil
}

// ListLocal возвращает папки из локального кеша (без сети): разворачивает VaultKey MasterKey'ом,
// расшифровывает имена и открывает в сессии.
func (u *UseCase) ListLocal(ctx context.Context) ([]DecryptedVault, error) {
	masterKey, ok := u.sess.MasterKey()
	if !ok {
		return nil, ErrLocked
	}

	items, err := u.local.ListVaults(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]DecryptedVault, 0, len(items))
	for _, it := range items {
		vaultKey, err := u.cipher.UnwrapVaultKey(it.WrappedVaultKey, masterKey)
		if err != nil {
			return nil, fmt.Errorf("unwrap vault key: %w", err)
		}

		var name string
		if err := u.cipher.DecryptStruct(vaultKey, it.EncName, &name); err != nil {
			return nil, fmt.Errorf("decrypt vault name: %w", err)
		}

		u.sess.OpenVault(it.ID, vaultKey)
		result = append(result, DecryptedVault{ID: it.ID, Name: name, Version: it.Version})
	}
	return result, nil
}
