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
	secretusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
	secretmocks "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret/mocks"
)

func TestBlobService_AttachBlob_Success(t *testing.T) {
	secretVaultOwner := secretmocks.NewMockVaultOwnership(t)
	secrets := secretmocks.NewMockRepository(t)
	secrets.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").Return(domain.Secret{
		ID: "secret-1", VaultID: "vault-1", Version: 1, Type: domain.SecretTypeBinary,
	}, nil)
	secrets.EXPECT().AttachBlob(mock.Anything, "secret-1", "blob-ref", int64(2048)).Return(int64(1), nil)
	secrets.EXPECT().BumpVaultVersion(mock.Anything, "vault-1").Return(nil)

	srv := newSecretTestServer(t, secrets, secretVaultOwner)
	resp, err := srv.AttachBlob(ctxWithUser("user-1"), &pb.AttachBlobRequest{
		SecretId: "secret-1", BaseVersion: 1, BlobRef: "blob-ref", BlobSize: 2048,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.GetVersion())
}

func TestBlobService_AttachBlob_NoUser(t *testing.T) {
	srv := newSecretTestServer(t, secretmocks.NewMockRepository(t), secretmocks.NewMockVaultOwnership(t))
	_, err := srv.AttachBlob(context.Background(), &pb.AttachBlobRequest{})
	requireStatusCode(t, err, codes.Unauthenticated)
}

func TestBlobService_AttachBlob_SecretNotFound(t *testing.T) {
	secrets := secretmocks.NewMockRepository(t)
	secrets.EXPECT().GetForUpdate(mock.Anything, "secret-x", "user-1").
		Return(domain.Secret{}, secretusecase.ErrSecretNotFound)

	srv := newSecretTestServer(t, secrets, secretmocks.NewMockVaultOwnership(t))
	_, err := srv.AttachBlob(ctxWithUser("user-1"), &pb.AttachBlobRequest{SecretId: "secret-x", BlobRef: "ref"})
	requireStatusCode(t, err, codes.NotFound)
}
