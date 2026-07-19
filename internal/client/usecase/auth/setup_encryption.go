package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// SetupEncryption настраивает шифрование по схеме «случайный MasterKey под passphrase-обёрткой».
//
// Два режима (определяются по состоянию сессии):
//   - Сессия БЕЗ MasterKey (первичная настройка) — генерируется НОВЫЙ случайный MasterKey.
//   - Сессия С MasterKey (после RecoverWithCode или смена пароля) — переиспользуется
//     СУЩЕСТВУЮЩИЙ MasterKey, только оборачивается новым passphrase-ключом (KEK).
//     Это гарантирует, что все wrapped_vault_key остаются валидными при смене пароля.
//
// На сервер уходят только непрозрачные параметры (salt, params) и enc_master_key
// (MasterKey, обёрнутый KEK). Сам MasterKey и passphrase сервер не видит.
func (u *UseCase) SetupEncryption(ctx context.Context, passphrase []byte) error {
	if len(passphrase) == 0 {
		return ErrEmptyPassphrase
	}

	// Определяем MasterKey: переиспользуем из сессии (recovery/смена пароля) либо генерируем новый.
	masterKey, ok := u.sess.MasterKey()
	if !ok {
		var err error
		masterKey, err = u.cipher.GenerateVaultKey() // случайный 32-байтный ключ
		if err != nil {
			return fmt.Errorf("generate master key: %w", err)
		}
	}

	salt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}
	params := crypto.DefaultParams()

	// Выводим KEK из passphrase (Argon2id + HKDF) — тот же путь, что раньше давал MasterKey,
	// но теперь это key-encryption-key для обёртки случайного MasterKey.
	kek, err := u.deriveKEK(passphrase, salt, params)
	if err != nil {
		return err
	}

	// Оборачиваем MasterKey ключом KEK.
	encMasterKey, err := u.cipher.WrapVaultKey(masterKey, kek)
	if err != nil {
		return fmt.Errorf("wrap master key: %w", err)
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal kdf params: %w", err)
	}

	tokens, err := u.tokens.Load()
	if err != nil {
		return fmt.Errorf("load tokens: %w", err)
	}

	if err := u.server.SetupEncryption(ctx, tokens.AccessToken, salt, paramsJSON, encMasterKey); err != nil {
		return err
	}

	u.sess.SetMasterKey(masterKey)
	u.encKDFSalt = salt
	u.encKDFParams = paramsJSON
	u.encMasterKey = encMasterKey
	return u.persistEncryption(ctx)
}

// deriveKEK выводит key-encryption-key из passphrase (Argon2id → HKDF).
func (u *UseCase) deriveKEK(passphrase, salt []byte, params crypto.Params) ([]byte, error) {
	seed, err := u.crypto.DeriveMasterSeed(domain.EncryptionPassphrase(passphrase), salt, params)
	if err != nil {
		return nil, fmt.Errorf("derive kek seed: %w", err)
	}
	kek, err := u.crypto.DeriveMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("derive kek: %w", err)
	}
	return kek, nil
}
