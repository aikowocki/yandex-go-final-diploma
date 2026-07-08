package app

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
)

// Container хранит все зависимости клиента.
type Container struct {
	Config    *config.ClientConfig
	GRPC      *grpcclient.Client
	Auth      *auth.UseCase
	Localizer *i18n.Localizer
}

// Close закрывает все закрываемые зависимости контейнера.
func (c *Container) Close() error {
	return c.GRPC.Close()
}
