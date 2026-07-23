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

func TestListIndex_Success(t *testing.T) {
	t.Parallel()

	secrets := mocks.NewMockRepository(t)
	secrets.EXPECT().ListIndex(mock.Anything, "vault-1", "user-1").Return([]domain.Secret{
		{ID: "s1", Version: 1, EncIndex: []byte("i1")},
		{ID: "s2", Version: 2, EncIndex: []byte("i2")},
	}, nil)

	got, err := secret.New(secrets, mocks.NewMockVaultOwnership(t), newTx(t)).ListIndex(context.Background(), "user-1", "vault-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, secret.IndexItem{ID: "s1", Version: 1, EncIndex: []byte("i1")}, got[0])
	assert.Equal(t, secret.IndexItem{ID: "s2", Version: 2, EncIndex: []byte("i2")}, got[1])
}

func TestListIndex_Empty(t *testing.T) {
	t.Parallel()

	secrets := mocks.NewMockRepository(t)
	secrets.EXPECT().ListIndex(mock.Anything, "vault-1", "user-1").Return(nil, nil)

	got, err := secret.New(secrets, mocks.NewMockVaultOwnership(t), newTx(t)).ListIndex(context.Background(), "user-1", "vault-1")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestListIndex_Validation(t *testing.T) {
	t.Parallel()

	uc := secret.New(mocks.NewMockRepository(t), mocks.NewMockVaultOwnership(t), newTx(t))

	_, err := uc.ListIndex(context.Background(), "", "vault-1")
	require.ErrorIs(t, err, secret.ErrEmptyUserID)

	_, err = uc.ListIndex(context.Background(), "user-1", "")
	require.ErrorIs(t, err, secret.ErrEmptyVaultID)
}

func TestListIndex_RepoError(t *testing.T) {
	t.Parallel()

	secrets := mocks.NewMockRepository(t)
	secrets.EXPECT().ListIndex(mock.Anything, "vault-1", "user-1").Return(nil, assert.AnError)

	_, err := secret.New(secrets, mocks.NewMockVaultOwnership(t), newTx(t)).ListIndex(context.Background(), "user-1", "vault-1")
	require.ErrorIs(t, err, assert.AnError)
}
