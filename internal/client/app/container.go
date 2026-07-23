package app

import (
	"errors"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	syncuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/sync"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

// Container хранит все зависимости клиента.
type Container struct {
	Config    *config.ClientConfig
	GRPC      *grpcclient.Client
	Local     *localstore.Store
	Session   *session.Session
	Auth      *authuc.UseCase
	Vault     *vaultuc.UseCase
	Secret    *secretuc.UseCase
	Sync      *syncuc.UseCase
	Localizer *i18n.Localizer
}

// Close закрывает все закрываемые зависимости контейнера.
func (c *Container) Close() error {
	return errors.Join(c.Local.Close(), c.GRPC.Close())
}
