package providers

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

func NewGRPCClient(cfg *config.ClientConfig) (*grpcclient.Client, error) {
	return grpcclient.New(cfg.ServerAddr)
}
