package providers

import "github.com/aikowocki/yandex-go-final-diploma/internal/client/config"

// NewConfig загружает конфигурацию клиента.
func NewConfig() (*config.ClientConfig, error) {
	return config.LoadClientConfig()
}
