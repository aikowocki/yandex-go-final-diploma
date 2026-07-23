package cli

import (
	"context"
	"errors"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
)

// ensureUnlocked гарантирует, что в сессии есть MasterKey. В one-shot CLI каждый вызов —
// новый процесс с пустой сессией, поэтому: обновляем токены (Refresh, заодно получаем
// enc_kdf_salt/params), запрашиваем EncryptionPassphrase и выводим MasterKey.
//
// Оффлайн-путь: если сервер недоступен, параметры KDF берутся из локального кеша (их туда
// кладёт предыдущий Login/Refresh) — тогда разблокировка и офлайн-мутации возможны без сети.
func ensureUnlocked(ctx context.Context, auth *authuc.UseCase, l *clienti18n.Localizer) error {
	if auth.MasterKeySet() {
		return nil
	}

	if err := auth.Refresh(ctx); err != nil {
		if !errors.Is(err, grpcclient.ErrUnavailable) {
			return err
		}
		// Сеть недоступна — пробуем офлайн-разблокировку по KDF-параметрам из кеша.
		if cacheErr := auth.LoadCachedEncryption(ctx); cacheErr != nil {
			return err // возвращаем исходную сетевую ошибку
		}
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
