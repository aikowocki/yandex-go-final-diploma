package grpcserver_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver"
	authmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
	secretusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
	secretmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret/mocks"
	vaultmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault/mocks"
)

func newSecretTestServer(t *testing.T, secrets *secretmocks.MockRepository, secretVaultOwner *secretmocks.MockVaultOwnership) *grpcserver.Server {
	t.Helper()
	return newTestServer(t, vaultmocks.NewMockRepository(t), secrets, secretVaultOwner, authmocks.NewMockRepository(t), authmocks.NewMockRecoveryRepository(t), authmocks.NewMockTokenIssuer(t))
}

func TestSecretService_CreateSecret_Success(t *testing.T) {
	secretVaultOwner := secretmocks.NewMockVaultOwnership(t)
	secretVaultOwner.EXPECT().IsOwner(mock.Anything, "vault-1", "user-1").Return(true, nil)

	secrets := secretmocks.NewMockRepository(t)
	secrets.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, s domain.Secret) (domain.Secret, error) {
		return s, nil
	})
	secrets.EXPECT().BumpVaultVersion(mock.Anything, "vault-1").Return(nil)

	srv := newSecretTestServer(t, secrets, secretVaultOwner)
	resp, err := srv.CreateSecret(ctxWithUser("user-1"), &pb.CreateSecretRequest{
		VaultId: "vault-1", SecretId: "secret-1", Type: pb.SecretType_SECRET_TYPE_TEXT,
		EncRow: []byte("row"), EncIndex: []byte("idx"), EncPayload: []byte("payload"),
	})
	require.NoError(t, err)
	assert.Equal(t, "secret-1", resp.GetSecretId())
}

func TestSecretService_CreateSecret_NoUser(t *testing.T) {
	srv := newSecretTestServer(t, secretmocks.NewMockRepository(t), secretmocks.NewMockVaultOwnership(t))
	_, err := srv.CreateSecret(context.Background(), &pb.CreateSecretRequest{})
	requireStatusCode(t, err, codes.Unauthenticated)
}

func TestSecretService_CreateSecret_VaultNotFound(t *testing.T) {
	secretVaultOwner := secretmocks.NewMockVaultOwnership(t)
	secretVaultOwner.EXPECT().IsOwner(mock.Anything, "vault-1", "user-1").Return(false, nil)

	srv := newSecretTestServer(t, secretmocks.NewMockRepository(t), secretVaultOwner)
	_, err := srv.CreateSecret(ctxWithUser("user-1"), &pb.CreateSecretRequest{
		VaultId: "vault-1", SecretId: "secret-1", EncRow: []byte("r"), EncIndex: []byte("i"),
	})
	requireStatusCode(t, err, codes.NotFound)
}

func TestSecretService_UpdateSecret_Conflict(t *testing.T) {
	secrets := secretmocks.NewMockRepository(t)
	secrets.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").Return(domain.Secret{
		ID: "secret-1", VaultID: "vault-1", Version: 5, EncRow: []byte("server-row"),
	}, nil)

	srv := newSecretTestServer(t, secrets, secretmocks.NewMockVaultOwnership(t))
	_, err := srv.UpdateSecret(ctxWithUser("user-1"), &pb.UpdateSecretRequest{
		SecretId: "secret-1", BaseVersion: 3, EncRow: []byte("r"), EncIndex: []byte("i"),
	})
	requireStatusCode(t, err, codes.Aborted)
}

func TestSecretService_UpdateSecret_Success(t *testing.T) {
	secrets := secretmocks.NewMockRepository(t)
	secrets.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").Return(domain.Secret{
		ID: "secret-1", VaultID: "vault-1", Version: 3,
	}, nil)
	secrets.EXPECT().UpdateFields(mock.Anything, "secret-1", mock.Anything, mock.Anything, mock.Anything).Return(int64(4), nil)
	secrets.EXPECT().BumpVaultVersion(mock.Anything, "vault-1").Return(nil)

	srv := newSecretTestServer(t, secrets, secretmocks.NewMockVaultOwnership(t))
	resp, err := srv.UpdateSecret(ctxWithUser("user-1"), &pb.UpdateSecretRequest{
		SecretId: "secret-1", BaseVersion: 3, EncRow: []byte("r"), EncIndex: []byte("i"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(4), resp.GetVersion())
}

func TestSecretService_DeleteSecret_Success(t *testing.T) {
	secrets := secretmocks.NewMockRepository(t)
	secrets.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").Return(domain.Secret{
		ID: "secret-1", VaultID: "vault-1", Version: 2,
	}, nil)
	secrets.EXPECT().SoftDelete(mock.Anything, "secret-1").Return(int64(3), nil)
	secrets.EXPECT().BumpVaultVersion(mock.Anything, "vault-1").Return(nil)

	srv := newSecretTestServer(t, secrets, secretmocks.NewMockVaultOwnership(t))
	_, err := srv.DeleteSecret(ctxWithUser("user-1"), &pb.DeleteSecretRequest{SecretId: "secret-1", BaseVersion: 2})
	require.NoError(t, err)
}

func TestSecretService_DeleteSecret_NoUser(t *testing.T) {
	srv := newSecretTestServer(t, secretmocks.NewMockRepository(t), secretmocks.NewMockVaultOwnership(t))
	_, err := srv.DeleteSecret(context.Background(), &pb.DeleteSecretRequest{})
	requireStatusCode(t, err, codes.Unauthenticated)
}

func TestSecretService_ListRow_Success(t *testing.T) {
	secrets := secretmocks.NewMockRepository(t)
	secrets.EXPECT().ListRow(mock.Anything, "vault-1", "user-1").Return([]domain.Secret{
		{ID: "s1", Type: domain.SecretTypeText, Version: 1, EncRow: []byte("r1")},
	}, nil)

	srv := newSecretTestServer(t, secrets, secretmocks.NewMockVaultOwnership(t))
	resp, err := srv.ListRow(ctxWithUser("user-1"), &pb.ListRowRequest{VaultId: "vault-1"})
	require.NoError(t, err)
	require.Len(t, resp.GetSecrets(), 1)
}

func TestSecretService_ListRow_NoUser(t *testing.T) {
	srv := newSecretTestServer(t, secretmocks.NewMockRepository(t), secretmocks.NewMockVaultOwnership(t))
	_, err := srv.ListRow(context.Background(), &pb.ListRowRequest{})
	requireStatusCode(t, err, codes.Unauthenticated)
}

func TestSecretService_ListIndex_Success(t *testing.T) {
	secrets := secretmocks.NewMockRepository(t)
	secrets.EXPECT().ListIndex(mock.Anything, "vault-1", "user-1").Return(nil, nil)

	srv := newSecretTestServer(t, secrets, secretmocks.NewMockVaultOwnership(t))
	resp, err := srv.ListIndex(ctxWithUser("user-1"), &pb.ListIndexRequest{VaultId: "vault-1"})
	require.NoError(t, err)
	assert.Empty(t, resp.GetSecrets())
}

func TestSecretService_ListIndex_NoUser(t *testing.T) {
	srv := newSecretTestServer(t, secretmocks.NewMockRepository(t), secretmocks.NewMockVaultOwnership(t))
	_, err := srv.ListIndex(context.Background(), &pb.ListIndexRequest{})
	requireStatusCode(t, err, codes.Unauthenticated)
}

func TestSecretService_GetPayload_Success(t *testing.T) {
	secrets := secretmocks.NewMockRepository(t)
	secrets.EXPECT().GetPayload(mock.Anything, "secret-1", "user-1").Return(domain.Secret{
		ID: "secret-1", Type: domain.SecretTypeText, Version: 1, EncPayload: []byte("p"),
	}, nil)

	srv := newSecretTestServer(t, secrets, secretmocks.NewMockVaultOwnership(t))
	resp, err := srv.GetPayload(ctxWithUser("user-1"), &pb.GetPayloadRequest{SecretId: "secret-1"})
	require.NoError(t, err)
	assert.Equal(t, []byte("p"), resp.GetEncPayload())
}

func TestSecretService_GetPayload_NotFound(t *testing.T) {
	secrets := secretmocks.NewMockRepository(t)
	secrets.EXPECT().GetPayload(mock.Anything, "secret-x", "user-1").Return(domain.Secret{}, secretusecase.ErrSecretNotFound)

	srv := newSecretTestServer(t, secrets, secretmocks.NewMockVaultOwnership(t))
	_, err := srv.GetPayload(ctxWithUser("user-1"), &pb.GetPayloadRequest{SecretId: "secret-x"})
	requireStatusCode(t, err, codes.NotFound)
}

func TestSecretService_GetPayload_NoUser(t *testing.T) {
	srv := newSecretTestServer(t, secretmocks.NewMockRepository(t), secretmocks.NewMockVaultOwnership(t))
	_, err := srv.GetPayload(context.Background(), &pb.GetPayloadRequest{})
	requireStatusCode(t, err, codes.Unauthenticated)
}
