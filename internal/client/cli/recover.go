package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
)

// RecoverCmd восстанавливает MasterKey по recovery-коду (получен при setup-encryption/register),
// когда мастер-пароль забыт.
type RecoverCmd struct {
	Code string `arg:"" optional:"" help:"Recovery code (from 'register'/'setup-encryption' output). Prompted if omitted."`
}

// Run восстанавливает MasterKey по коду восстановления и предлагает задать новый мастер-пароль.
func (c *RecoverCmd) Run(uc *authuc.UseCase, l *clienti18n.Localizer) error {
	ctx := context.Background()

	// Нужен валидный access token, но НЕ MasterKey (пароль забыт — Unlock невозможен).
	// Refresh поднимает токены из локально сохранённого refresh-токена (после предыдущего Login).
	if err := uc.Refresh(ctx); err != nil {
		if errors.Is(err, grpcclient.ErrUnavailable) {
			return fmt.Errorf("%s", l.T("sync_offline"))
		}
		return err
	}

	code := c.Code
	if code == "" {
		var err error
		code, err = promptLine(l.T("prompt_recovery_code"))
		if err != nil {
			return err
		}
	}

	if err := uc.RecoverWithCode(ctx, code); err != nil {
		return fmt.Errorf("%s: %w", l.T("recovery_invalid_code"), err)
	}
	fmt.Println(l.T("recovery_success"))

	// MasterKey восстановлен в сессии этого процесса, но осядет только если немедленно
	// обернем его новым мастер-паролем — иначе он исчезнет вместе с процессом CLI.
	return runSetupEncryption(ctx, uc, l)
}
