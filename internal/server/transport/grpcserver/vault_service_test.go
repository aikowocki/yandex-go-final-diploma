package grpcserver_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	authusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	authmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
	blobusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/blob"
	secretusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
	secretmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret/mocks"
	vaultusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault"
	vaultmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault/mocks"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver"
)

// passthroughTx — тестовый TxManager, выполняющий fn немедленно (без реальной транзакции).
type passthroughTx struct{}

func (passthroughTx) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

// newTestServer собирает *grpcserver.Server с реальными usecase поверх мок-репозиториев —
// проверяет именно grpc-слой (передача userID из контекста, вызов mapper'ов, маппинг ошибок),
// без сети и без БД.
func newTestServer(t *testing.T, vaults *vaultmocks.MockRepository, secrets *secretmocks.MockRepository, secretVaultOwner *secretmocks.MockVaultOwnership, users *authmocks.MockRepository, recovery *authmocks.MockRecoveryRepository, tokens *authmocks.MockTokenIssuer) *grpcserver.Server {
	t.Helper()
	vaultUC := vaultusecase.New(vaults)
	secretUC := secretusecase.New(secrets, secretVaultOwner, passthroughTx{})
	authUC := authusecase.New(users, recovery, tokens, passthroughTx{})
	var blobUC *blobusecase.UseCase // без MinIO — как в проде без объектного хранилища
	return grpcserver.New(authUC, vaultUC, secretUC, blobUC, tokens)
}

func newVaultTestServer(t *testing.T, vaults *vaultmocks.MockRepository) *grpcserver.Server {
	t.Helper()
	return newTestServer(t, vaults, secretmocks.NewMockRepository(t), secretmocks.NewMockVaultOwnership(t), authmocks.NewMockRepository(t), authmocks.NewMockRecoveryRepository(t), authmocks.NewMockTokenIssuer(t))
}

func TestVaultService_CreateVault_Success(t *testing.T) {
	vaults := vaultmocks.NewMockRepository(t)
	vaults.EXPECT().Create(mock.Anything, mock.Anything).Return(
		domain.Vault{ID: "vault-1"}, nil,
	)

	srv := newVaultTestServer(t, vaults)
	resp, err := srv.CreateVault(ctxWithUser("user-1"), &pb.CreateVaultRequest{
		WrappedVaultKey: []byte("wvk"), EncName: []byte("name"),
	})
	require.NoError(t, err)
	assert.Equal(t, "vault-1", resp.GetVaultId())
}

func TestVaultService_CreateVault_NoUser(t *testing.T) {
	srv := newVaultTestServer(t, vaultmocks.NewMockRepository(t))
	_, err := srv.CreateVault(context.Background(), &pb.CreateVaultRequest{})
	requireStatusCode(t, err, codes.Unauthenticated)
}

func TestVaultService_CreateVault_ValidationError(t *testing.T) {
	srv := newVaultTestServer(t, vaultmocks.NewMockRepository(t))
	_, err := srv.CreateVault(ctxWithUser("user-1"), &pb.CreateVaultRequest{})
	requireStatusCode(t, err, codes.InvalidArgument)
}

func TestVaultService_CreateVault_RepoError(t *testing.T) {
	vaults := vaultmocks.NewMockRepository(t)
	vaults.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.Vault{}, assert.AnError)

	srv := newVaultTestServer(t, vaults)
	_, err := srv.CreateVault(ctxWithUser("user-1"), &pb.CreateVaultRequest{
		WrappedVaultKey: []byte("wvk"), EncName: []byte("name"),
	})
	requireStatusCode(t, err, codes.Internal)
}

func TestVaultService_ListVaults_Success(t *testing.T) {
	vaults := vaultmocks.NewMockRepository(t)
	vaults.EXPECT().ListByUser(mock.Anything, "user-1").Return(nil, nil)

	srv := newVaultTestServer(t, vaults)
	resp, err := srv.ListVaults(ctxWithUser("user-1"), &pb.ListVaultsRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.GetVaults())
}

func TestVaultService_ListVaults_NoUser(t *testing.T) {
	srv := newVaultTestServer(t, vaultmocks.NewMockRepository(t))
	_, err := srv.ListVaults(context.Background(), &pb.ListVaultsRequest{})
	requireStatusCode(t, err, codes.Unauthenticated)
}

func TestVaultService_CheckFreshness_Success(t *testing.T) {
	vaults := vaultmocks.NewMockRepository(t)
	vaults.EXPECT().CheckFreshness(mock.Anything, "user-1").Return(nil, nil)

	srv := newVaultTestServer(t, vaults)
	resp, err := srv.CheckFreshness(ctxWithUser("user-1"), &pb.CheckFreshnessRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.GetVaults())
}

func TestVaultService_CheckFreshness_NoUser(t *testing.T) {
	srv := newVaultTestServer(t, vaultmocks.NewMockRepository(t))
	_, err := srv.CheckFreshness(context.Background(), &pb.CheckFreshnessRequest{})
	requireStatusCode(t, err, codes.Unauthenticated)
}

func requireStatusCode(t *testing.T, err error, code codes.Code) {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, code, st.Code())
}
