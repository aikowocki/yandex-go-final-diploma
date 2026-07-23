package providers

import "github.com/aikowocki/yandex-go-final-diploma/internal/server/config"

// NewConfig загружает и возвращает конфигурацию сервера.
func NewConfig() (*config.ServerConfig, error) {
	return config.LoadServerConfig()
}
