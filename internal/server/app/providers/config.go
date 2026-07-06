package providers

import "github.com/aikowocki/yandex-go-final-diploma/internal/server/config"

func NewConfig() (*config.ServerConfig, error) {
	return config.LoadServerConfig()
}
