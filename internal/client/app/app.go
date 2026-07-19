package app

import (
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app/providers"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/keyring"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	syncuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/sync"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
	"github.com/aikowocki/yandex-go-final-diploma/internal/logger"
)

// New собирает все зависимости из уже разобранного конфига и возвращает готовый Container.
func New(cfg *config.ClientConfig) (*Container, error) {
	logger.SetupWithDir(cfg.LogLevel, cfg.DataDir)

	grpcClient, err := grpcclient.New(cfg.ServerAddr)
	if err != nil {
		return nil, fmt.Errorf("grpc client: %w", err)
	}

	local, err := localstore.Open(cfg.DataDir, !cfg.NoPersist)
	if err != nil {
		_ = grpcClient.Close()
		return nil, fmt.Errorf("local storage: %w", err)
	}

	tokenStore := keyring.New(cfg.DataDir, !cfg.NoPersist)
	// Оборачиваем TokenStore проактивным auto-refresh: при Load() проверяет exp JWT
	// и обновляет пару через RefreshToken RPC если access token скоро протухнет.
	autoRefreshTokens := keyring.NewAutoRefreshStore(tokenStore, grpcClient)
	crypto := cryptoimpl.Crypto{}
	sess := session.New() // общий крипто-материал сессии (MasterKey + открытые VaultKey)

	authUseCase := authuc.New(grpcClient, crypto, crypto, autoRefreshTokens, sess, local)
	vaultUseCase := vaultuc.New(grpcClient, crypto, autoRefreshTokens, sess, local)
	secretUseCase := secretuc.New(grpcClient, crypto, autoRefreshTokens, sess, local, cfg.DataDir)
	syncUseCase := syncuc.New(grpcClient, local, autoRefreshTokens)
	syncUseCase.SetBlobUploader(secretUseCase)
	syncUseCase.SetIndexLoader(secretUseCase)
	syncUseCase.SetSyncDelay(cfg.SyncDelayMs)

	localizer := providers.NewLocalizer(cfg)

	return &Container{
		Config:    cfg,
		GRPC:      grpcClient,
		Local:     local,
		Session:   sess,
		Auth:      authUseCase,
		Vault:     vaultUseCase,
		Secret:    secretUseCase,
		Sync:      syncUseCase,
		Localizer: localizer,
	}, nil
}
