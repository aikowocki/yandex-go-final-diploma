package app

import (
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app/providers"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/keyring"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
	"github.com/aikowocki/yandex-go-final-diploma/internal/logger"
)

// New собирает все зависимости из уже разобранного конфига и возвращает готовый Container.
func New(cfg *config.ClientConfig) (*Container, error) {
	logger.Setup(cfg.LogLevel)

	grpcClient, err := grpcclient.New(cfg.ServerAddr)
	if err != nil {
		return nil, fmt.Errorf("grpc client: %w", err)
	}

	tokenStore := keyring.New(cfg.DataDir, !cfg.NoPersist)
	crypto := cryptoimpl.Crypto{}
	sess := session.New() // общий крипто-материал сессии (MasterKey + открытые VaultKey)

	authUseCase := authuc.New(grpcClient, crypto, tokenStore, sess)
	vaultUseCase := vaultuc.New(grpcClient, crypto, tokenStore, sess)
	secretUseCase := secretuc.New(grpcClient, crypto, tokenStore, sess)

	localizer := providers.NewLocalizer(cfg)

	return &Container{
		Config:    cfg,
		GRPC:      grpcClient,
		Session:   sess,
		Auth:      authUseCase,
		Vault:     vaultUseCase,
		Secret:    secretUseCase,
		Localizer: localizer,
	}, nil
}
