package providers

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

// NewGRPCClient создаёт gRPC-клиент, подключённый к серверу из конфига.
func NewGRPCClient(cfg *config.ClientConfig) (*grpcclient.Client, error) {
	return grpcclient.New(cfg.ServerAddr)
}
