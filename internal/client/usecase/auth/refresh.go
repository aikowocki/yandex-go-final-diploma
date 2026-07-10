package auth

import (
	"context"
	"fmt"
)

// Refresh обновляет токены по сохранённому refresh-токену и запоминает параметры KDF
// (enc_kdf_salt/params), нужные для последующего Unlock. Используется командами в свежем
// процессе (CLI one-shot), чтобы получить enc-параметры без повторного ввода пароля входа.
func (u *UseCase) Refresh(ctx context.Context) error {
	tokens, err := u.tokens.Load()
	if err != nil {
		return err
	}

	res, err := u.server.RefreshToken(ctx, tokens.RefreshToken)
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
