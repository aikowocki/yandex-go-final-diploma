package cli

import (
	"context"
	"fmt"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
)

// LoginCmd — логин и разблокировка: сначала аутентификация (JWT), затем отдельный шаг
// Unlock (локальный вывод MasterKey). Два разных промпта — не путать местами.
type LoginCmd struct {
	Login string `arg:"" optional:"" help:"Account login. Prompted if omitted."`
}

// Run выполняет аутентификацию и, при необходимости, разблокировку локальной сессии.
func (c *LoginCmd) Run(uc *authuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()

	login := c.Login
	if login == "" {
		var err error
		login, err = promptLine(l.T("prompt_login"))
		if err != nil {
			return err
		}
	}

	credential, err := promptSecret(l.T("prompt_login_credential"))
	if err != nil {
		return err
	}

	if err := uc.Login(ctx, login, credential); err != nil {
		return err
	}
	fmt.Println(l.T("login_success"))

	// Если шифрование ещё не настроено — предлагаем настроить прямо сейчас.
	if !uc.EncryptionConfigured() {
		fmt.Println(l.T("encryption_not_configured"))
		confirmed, err := promptConfirm(l.T("encryption_confirm"))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println(l.T("encryption_skipped"))
			return nil
		}
		return runSetupEncryption(ctx, uc, l)
	}

	passphrase, err := promptSecret(l.T("prompt_passphrase"))
	if err != nil {
		return err
	}

	if err := uc.Unlock(ctx, passphrase); err != nil {
		return err
	}
	fmt.Println(l.T("unlock_success"))

	return nil
}
