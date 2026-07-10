package vault

import (
	"context"
	"fmt"
)

// List возвращает расшифрованные ваулты пользователя. Для каждого разворачивает VaultKey
// MasterKey'ом, расшифровывает имя и открывает ваулт в сессии (VaultKey понадобится для секретов).
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
		result = append(result, DecryptedVault{ID: it.ID, Name: name, Version: it.Version})
	}
	return result, nil
}
