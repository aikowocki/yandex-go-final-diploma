package cli

import (
	"context"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
)

// ensureUnlocked гарантирует, что в сессии есть MasterKey. В one-shot CLI каждый вызов —
// новый процесс с пустой сессией, поэтому: обновляем токены (Refresh, заодно получаем
// enc_kdf_salt/params), запрашиваем EncryptionPassphrase и выводим MasterKey.
func ensureUnlocked(ctx context.Context, auth *authuc.UseCase, l *clienti18n.Localizer) error {
	if auth.MasterKeySet() {
		return nil
	}

	if err := auth.Refresh(ctx); err != nil {
		return err
	}
	if !auth.EncryptionConfigured() {
		return authuc.ErrEncryptionNotSetup
	}

	passphrase, err := promptSecret(l.T("prompt_passphrase"))
	if err != nil {
		return err
	}
	return auth.Unlock(ctx, passphrase)
}
