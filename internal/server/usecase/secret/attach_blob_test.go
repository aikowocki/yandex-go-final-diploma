package secret_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret/mocks"
)

func TestAttachBlob_Success(t *testing.T) {
	t.Parallel()

	secrets := mocks.NewMockRepository(t)
	secrets.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").
		Return(domain.Secret{ID: "secret-1", VaultID: "vault-1", Type: domain.SecretTypeBinary, Version: 1}, nil)
	secrets.EXPECT().AttachBlob(mock.Anything, "secret-1", "vault-1/secret-1", int64(2048)).Return(int64(2), nil)
	secrets.EXPECT().BumpVaultVersion(mock.Anything, "vault-1").Return(nil)

	uc := secret.New(secrets, mocks.NewMockVaultOwnership(t), newTx(t))
	version, err := uc.AttachBlob(context.Background(), secret.AttachBlobParams{
		UserID: "user-1", SecretID: "secret-1", BaseVersion: 1,
		BlobRef: "vault-1/secret-1", BlobSize: 2048,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), version)
}

func TestAttachBlob_Conflict(t *testing.T) {
	t.Parallel()

	secrets := mocks.NewMockRepository(t)
	secrets.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").
		Return(domain.Secret{ID: "secret-1", VaultID: "vault-1", Type: domain.SecretTypeBinary, Version: 5}, nil)

	uc := secret.New(secrets, mocks.NewMockVaultOwnership(t), newTx(t))
	_, err := uc.AttachBlob(context.Background(), secret.AttachBlobParams{
		UserID: "user-1", SecretID: "secret-1", BaseVersion: 1, BlobRef: "ref",
	})

	var conflict *secret.ErrConflict
	require.ErrorAs(t, err, &conflict)
	assert.Equal(t, int64(5), conflict.Current.Version)
}

func TestAttachBlob_WrongType(t *testing.T) {
	t.Parallel()

	secrets := mocks.NewMockRepository(t)
	secrets.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").
		Return(domain.Secret{ID: "secret-1", VaultID: "vault-1", Type: domain.SecretTypeLoginPassword, Version: 1}, nil)

	uc := secret.New(secrets, mocks.NewMockVaultOwnership(t), newTx(t))
	_, err := uc.AttachBlob(context.Background(), secret.AttachBlobParams{
		UserID: "user-1", SecretID: "secret-1", BaseVersion: 1, BlobRef: "ref",
	})
	require.ErrorIs(t, err, secret.ErrNotBinarySecret)
}

func TestAttachBlob_DeletedIsNotFound(t *testing.T) {
	t.Parallel()

	secrets := mocks.NewMockRepository(t)
	secrets.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").
		Return(domain.Secret{ID: "secret-1", VaultID: "vault-1", Type: domain.SecretTypeBinary, Version: 1, Deleted: true}, nil)

	uc := secret.New(secrets, mocks.NewMockVaultOwnership(t), newTx(t))
	_, err := uc.AttachBlob(context.Background(), secret.AttachBlobParams{
		UserID: "user-1", SecretID: "secret-1", BaseVersion: 1, BlobRef: "ref",
	})
	require.ErrorIs(t, err, secret.ErrSecretNotFound)
}

func TestAttachBlob_Validation(t *testing.T) {
	t.Parallel()

	uc := secret.New(mocks.NewMockRepository(t), mocks.NewMockVaultOwnership(t), newTx(t))

	_, err := uc.AttachBlob(context.Background(), secret.AttachBlobParams{SecretID: "s", BlobRef: "r"})
	require.ErrorIs(t, err, secret.ErrEmptyUserID)

	_, err = uc.AttachBlob(context.Background(), secret.AttachBlobParams{UserID: "u", BlobRef: "r"})
	require.ErrorIs(t, err, secret.ErrEmptySecretID)

	_, err = uc.AttachBlob(context.Background(), secret.AttachBlobParams{UserID: "u", SecretID: "s"})
	require.ErrorIs(t, err, secret.ErrEmptyBlobRef)
}
