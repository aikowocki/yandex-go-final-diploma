package providers

import (
	"time"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/objectstore"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/postgres"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/blob"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/jwt"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

// NewGRPC собирает gRPC-сервер со всеми usecase. objectStore может быть nil (сервер без MinIO) —
// тогда blob-usecase создаётся с nil-хранилищем (UploadBlob/DownloadBlob вернут Unimplemented,
// остальные типы секретов не затрагиваются). Принимает конкретный *objectstore.Store (не
// contracts.BlobStorage), чтобы typed-nil не превратился в "non-nil interface с nil внутри".
func NewGRPC(cfg *config.ServerConfig, db *postgres.DB, objectStore *objectstore.Store) *grpcserver.Server {
	tokens := jwt.New([]byte(cfg.JWTSecret), accessTokenTTL, refreshTokenTTL)

	vaultRepo := postgres.NewVaultRepo(db)
	secretRepo := postgres.NewSecretRepo(db)
	authUseCase := auth.New(postgres.NewUserRepo(db), tokens, postgres.NewTxManager(db))
	vaultUseCase := vault.New(vaultRepo)
	secretUseCase := secret.New(secretRepo, vaultRepo, postgres.NewTxManager(db))

	var storage contracts.BlobStorage
	if objectStore != nil {
		storage = objectStore
	}
	blobUseCase := blob.New(storage, secretRepo)

	return grpcserver.New(authUseCase, vaultUseCase, secretUseCase, blobUseCase, tokens)
}
