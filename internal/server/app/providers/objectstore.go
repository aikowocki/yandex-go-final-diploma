package providers

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/objectstore"
)

func NewObjectStore(ctx context.Context, cfg *config.ServerConfig) (*objectstore.Store, error) {
	if cfg.MinioEndpoint == "" {
		return nil, nil
	}
	return objectstore.New(ctx, objectstore.Config{
		Endpoint:  cfg.MinioEndpoint,
		AccessKey: cfg.MinioAccess,
		SecretKey: cfg.MinioSecret,
		Bucket:    cfg.MinioBucket,
		UseSSL:    cfg.MinioUseSSL,
	})
}
