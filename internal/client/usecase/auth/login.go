package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// Login аутентифицирует пользователя, сохраняет токены и запоминает в сессии параметры
// KDF (enc_kdf_salt/enc_kdf_params) для последующего Unlock. MasterKey здесь не выводится —
// это отдельный шаг (сервер в нём не участвует).
func (u *UseCase) Login(ctx context.Context, login string, loginCredential []byte) error {
	if login == "" {
		return ErrEmptyLogin
	}
	if len(loginCredential) == 0 {
		return ErrEmptyCredential
	}

	res, err := u.server.Login(ctx, login, loginCredential)
	if err != nil {
		return err
	}

	if err := u.reconcileAccount(ctx, res.UserID); err != nil {
		return err
	}

	// Кешируем login для отображения в UI (Lock-экран, User-меню).
	_ = u.local.KVSet(ctx, kvAccountLogin, []byte(login))

	if err := u.tokens.Save(res.Tokens); err != nil {
		return fmt.Errorf("save tokens: %w", err)
	}

	u.encKDFSalt = res.EncKDFSalt
	u.encKDFParams = res.EncKDFParams
	u.encMasterKey = res.EncMasterKey
	if err := u.persistEncryption(ctx); err != nil {
		return err
	}

	// Подтягиваем vault-метаданные с сервера (Tier 1: wrapped_vault_key) и кешируем,
	// чтобы при последующем Unlock было чем проверить корректность MasterKey.
	u.prefetchVaultMeta(ctx, res.AccessToken)
	return nil
}

// prefetchVaultMeta тянет метаданные vault'ов с сервера и кеширует.
func (u *UseCase) prefetchVaultMeta(ctx context.Context, accessToken string) {
	items, err := u.server.ListVaults(ctx, accessToken)
	if err != nil {
		return
	}
	for _, it := range items {
		_ = u.local.UpsertVault(ctx, contracts.LocalVault{
			ID:              it.ID,
			WrappedVaultKey: it.WrappedVaultKey,
			EncName:         it.EncName,
			Version:         it.Version,
		})
	}
}

// Unlock выводит MasterKey локально по параметрам KDF, полученным при Login.
// Сервер в этом шаге не участвует. После вывода MasterKey проверяет его корректность:
// пытается развернуть VaultKey первого закешированного vault'а (если есть) — при неверном
// passphrase Unwrap провалится с ошибкой AEAD, и MasterKey НЕ устанавливается в сессию.
func (u *UseCase) Unlock(ctx context.Context, passphrase []byte) error {
	if len(passphrase) == 0 {
		return ErrEmptyPassphrase
	}
	if len(u.encKDFSalt) == 0 || len(u.encKDFParams) == 0 || len(u.encMasterKey) == 0 {
		return ErrEncryptionNotSetup
	}

	var params crypto.Params
	if err := json.Unmarshal(u.encKDFParams, &params); err != nil {
		return fmt.Errorf("unmarshal kdf params: %w", err)
	}

	// Выводим KEK из passphrase и разворачиваем им MasterKey. Неверный пароль → KEK неверный →
	// UnwrapVaultKey падает с AEAD-ошибкой (сам факт проверки корректности пароля).
	kek, err := u.deriveKEK(passphrase, u.encKDFSalt, params)
	if err != nil {
		return err
	}

	masterKey, err := u.cipher.UnwrapVaultKey(u.encMasterKey, kek)
	if err != nil {
		return fmt.Errorf("incorrect passphrase")
	}

	u.sess.SetMasterKey(masterKey)
	return nil
}
