package app

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver"
)

// Container хранит все зависимости сервера.
type Container struct {
	Config *config.ServerConfig
	DB     *postgres.DB
	GRPC   *grpcserver.Server
}
