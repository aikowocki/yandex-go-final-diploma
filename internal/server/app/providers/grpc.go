package providers

import (
	"time"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/jwt"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

func NewGRPC(cfg *config.ServerConfig, db *postgres.DB) *grpcserver.Server {
	tokens := jwt.New([]byte(cfg.JWTSecret), accessTokenTTL, refreshTokenTTL)
	authUseCase := auth.New(postgres.NewUserRepo(db), tokens, postgres.NewTxManager(db))
	return grpcserver.New(authUseCase, tokens)
}
