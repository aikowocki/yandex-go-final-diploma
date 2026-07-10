package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// SetupEncryption выводит MasterKey из EncryptionPassphrase локально и отправляет на
// сервер только непрозрачные параметры KDF (enc_kdf_salt/enc_kdf_params). Сам MasterKey
// и passphrase сервер не видит. MasterKey остаётся в памяти сессии.
func (u *UseCase) SetupEncryption(ctx context.Context, passphrase []byte) error {
	if len(passphrase) == 0 {
		return ErrEmptyPassphrase
	}

	salt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}

	params := crypto.DefaultParams()

	masterKey, err := u.deriveMasterKey(passphrase, salt, params)
	if err != nil {
		return err
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal kdf params: %w", err)
	}

	tokens, err := u.tokens.Load()
	if err != nil {
		return fmt.Errorf("load tokens: %w", err)
	}

	if err := u.server.SetupEncryption(ctx, tokens.AccessToken, salt, paramsJSON); err != nil {
		return err
	}

	u.sess.SetMasterKey(masterKey)
	u.encKDFSalt = salt
	u.encKDFParams = paramsJSON
	return u.persistEncryption(ctx)
}

// deriveMasterKey — общая цепочка Argon2id → HKDF для SetupEncryption и Unlock.
func (u *UseCase) deriveMasterKey(passphrase, salt []byte, params crypto.Params) ([]byte, error) {
	seed, err := u.crypto.DeriveMasterSeed(domain.EncryptionPassphrase(passphrase), salt, params)
	if err != nil {
		return nil, fmt.Errorf("derive master seed: %w", err)
	}

	masterKey, err := u.crypto.DeriveMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("derive master key: %w", err)
	}

	return masterKey, nil
}
