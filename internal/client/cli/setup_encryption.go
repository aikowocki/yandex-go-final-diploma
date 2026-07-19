package cli

import (
	"context"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
)

// SetupEncryptionCmd — настройка шифрования (мастер-пароль + recovery codes) для текущего аккаунта.
type SetupEncryptionCmd struct{}

// Run настраивает шифрование для текущего аккаунта и печатает recovery codes.
func (c *SetupEncryptionCmd) Run(uc *authuc.UseCase, l *clienti18n.Localizer) error {
	return runSetupEncryption(context.Background(), uc, l)
}
