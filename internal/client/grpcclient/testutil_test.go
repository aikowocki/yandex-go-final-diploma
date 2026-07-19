package grpcclient_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	authusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	authmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
	blobusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/blob"
	secretusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
	secretmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret/mocks"
	vaultusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault"
	vaultmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault/mocks"

	contractsmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/jwt"
)

// passthroughTx — тестовый TxManager, выполняющий fn немедленно (без реальной транзакции).
type passthroughTx struct{}

func (passthroughTx) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

// testServerBundle связывает мок-репозитории серверных usecase с реальным *grpcclient.Client,
// подключённым к серверу, поднятому на локальном TCP-порту (127.0.0.1:0) — позволяет
// тестировать RPC-слой grpcclient.Client end-to-end (маппинг proto<->contracts, коды ошибок,
// извлечение ConflictError, blob-стриминг) без внешней сети/Docker/testcontainers.
type testServerBundle struct {
	Client   *grpcclient.Client
	Users    *authmocks.MockRepository
	Recovery *authmocks.MockRecoveryRepository
	Vaults   *vaultmocks.MockRepository
	Secret   *secretmocks.MockRepository
	Owner    *secretmocks.MockVaultOwnership
	Storage  *contractsmocks.MockBlobStorage
	Tokens   *jwt.TokenIssuer
}

func newTestGRPCClient(t *testing.T) *testServerBundle {
	t.Helper()
	return newTestGRPCClientWithBlob(t, nil)
}

// newTestGRPCClientWithBlobStorage поднимает тот же тестовый сервер, но с реальным
// blob.UseCase над мок-хранилищем — позволяет тестировать полный путь Upload/DownloadBlob
// (стриминг, mapBlobErr) без MinIO/testcontainers.
func newTestGRPCClientWithBlobStorage(t *testing.T) *testServerBundle {
	t.Helper()
	storage := contractsmocks.NewMockBlobStorage(t)
	return newTestGRPCClientWithBlob(t, storage)
}

func newTestGRPCClientWithBlob(t *testing.T, storage *contractsmocks.MockBlobStorage) *testServerBundle {
	t.Helper()

	users := authmocks.NewMockRepository(t)
	recovery := authmocks.NewMockRecoveryRepository(t)
	vaults := vaultmocks.NewMockRepository(t)
	secrets := secretmocks.NewMockRepository(t)
	owner := secretmocks.NewMockVaultOwnership(t)

	tokens := jwt.New([]byte("test-secret"), 15*time.Minute, 7*24*time.Hour)

	authUC := authusecase.New(users, recovery, tokens, passthroughTx{})
	vaultUC := vaultusecase.New(vaults)
	secretUC := secretusecase.New(secrets, owner, passthroughTx{})
	var blobUC *blobusecase.UseCase
	if storage != nil {
		blobUC = blobusecase.New(storage, secrets)
	}

	srv := grpcserver.New(authUC, vaultUC, secretUC, blobUC, tokens)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	require.NoError(t, lis.Close())

	go func() { _ = srv.Run(addr) }()
	t.Cleanup(srv.Stop)

	require.Eventually(t, func() bool {
		conn, dialErr := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if dialErr != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 5*time.Second, 50*time.Millisecond, "тестовый сервер должен начать слушать адрес")

	client, err := grpcclient.New(addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	return &testServerBundle{Client: client, Users: users, Recovery: recovery, Vaults: vaults, Secret: secrets, Owner: owner, Storage: storage, Tokens: tokens}
}
