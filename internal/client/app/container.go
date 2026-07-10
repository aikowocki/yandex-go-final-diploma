package app

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// Container хранит все зависимости клиента.
type Container struct {
	Config    *config.ClientConfig
	GRPC      *grpcclient.Client
	Session   *session.Session
	Auth      *authuc.UseCase
	Vault     *vaultuc.UseCase
	Secret    *secretuc.UseCase
	Localizer *i18n.Localizer
}

// Close закрывает все закрываемые зависимости контейнера.
func (c *Container) Close() error {
	return c.GRPC.Close()
}
