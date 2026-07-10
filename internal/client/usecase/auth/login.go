package auth

import (
	"context"
	"encoding/json"
	"fmt"

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

	if err := u.tokens.Save(res.Tokens); err != nil {
		return fmt.Errorf("save tokens: %w", err)
	}

	u.encKDFSalt = res.EncKDFSalt
	u.encKDFParams = res.EncKDFParams
	return u.persistEncryption(ctx)
}

// Unlock выводит MasterKey локально по параметрам KDF, полученным при Login.
// Сервер в этом шаге не участвует.
func (u *UseCase) Unlock(_ context.Context, passphrase []byte) error {
	if len(passphrase) == 0 {
		return ErrEmptyPassphrase
	}
	if len(u.encKDFSalt) == 0 || len(u.encKDFParams) == 0 {
		return ErrEncryptionNotSetup
	}

	var params crypto.Params
	if err := json.Unmarshal(u.encKDFParams, &params); err != nil {
		return fmt.Errorf("unmarshal kdf params: %w", err)
	}

	masterKey, err := u.deriveMasterKey(passphrase, u.encKDFSalt, params)
	if err != nil {
		return err
	}

	u.sess.SetMasterKey(masterKey)
	return nil
}
