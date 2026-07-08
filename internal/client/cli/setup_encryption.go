package cli

import (
	"context"

	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
)

type SetupEncryptionCmd struct{}

func (c *SetupEncryptionCmd) Run(uc *authuc.UseCase, l *clienti18n.Localizer) error {
	return runSetupEncryption(context.Background(), uc, l)
}
