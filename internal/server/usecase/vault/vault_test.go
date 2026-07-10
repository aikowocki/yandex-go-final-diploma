package vault_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault/mocks"
)

func validCreateParams() vault.CreateParams {
	return vault.CreateParams{
		UserID:          "user-1",
		WrappedVaultKey: []byte("wrapped-key"),
		EncName:         []byte("enc-name"),
	}
}

func TestCreateVault_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRepository(t)
	repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, v domain.Vault) (domain.Vault, error) {
			assert.Equal(t, "user-1", v.UserID)
			assert.Equal(t, []byte("wrapped-key"), v.WrappedVaultKey)
			assert.Equal(t, []byte("enc-name"), v.EncName)
			v.ID = "vault-1"
			return v, nil
		})

	id, err := vault.New(repo).CreateVault(context.Background(), validCreateParams())
	require.NoError(t, err)
	assert.Equal(t, "vault-1", id)
}

func TestCreateVault_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*vault.CreateParams)
		wantErr error
	}{
		{"empty user id", func(p *vault.CreateParams) { p.UserID = "" }, vault.ErrEmptyUserID},
		{"empty vault key", func(p *vault.CreateParams) { p.WrappedVaultKey = nil }, vault.ErrEmptyVaultKey},
		{"empty enc name", func(p *vault.CreateParams) { p.EncName = nil }, vault.ErrEmptyEncName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Репозиторий не должен вызываться при провале валидации.
			repo := mocks.NewMockRepository(t)
			params := validCreateParams()
			tt.mutate(&params)

			_, err := vault.New(repo).CreateVault(context.Background(), params)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestCreateVault_RepoError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("insert failed")
	repo := mocks.NewMockRepository(t)
	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.Vault{}, wantErr)

	_, err := vault.New(repo).CreateVault(context.Background(), validCreateParams())
	assert.ErrorIs(t, err, wantErr)
}

func TestListVaults_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRepository(t)
	repo.EXPECT().ListByUser(mock.Anything, "user-1").Return([]domain.Vault{
		{ID: "v1", WrappedVaultKey: []byte("k1"), EncName: []byte("n1"), Version: 1},
		{ID: "v2", WrappedVaultKey: []byte("k2"), EncName: []byte("n2"), Version: 3},
	}, nil)

	got, err := vault.New(repo).ListVaults(context.Background(), "user-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, vault.Tier1{ID: "v1", WrappedVaultKey: []byte("k1"), EncName: []byte("n1"), Version: 1}, got[0])
	assert.Equal(t, vault.Tier1{ID: "v2", WrappedVaultKey: []byte("k2"), EncName: []byte("n2"), Version: 3}, got[1])
}

func TestListVaults_EmptyUserID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRepository(t)
	_, err := vault.New(repo).ListVaults(context.Background(), "")
	require.ErrorIs(t, err, vault.ErrEmptyUserID)
}

func TestListVaults_RepoError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("query failed")
	repo := mocks.NewMockRepository(t)
	repo.EXPECT().ListByUser(mock.Anything, "user-1").Return(nil, wantErr)

	_, err := vault.New(repo).ListVaults(context.Background(), "user-1")
	assert.ErrorIs(t, err, wantErr)
}

func TestCheckFreshness_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRepository(t)
	repo.EXPECT().CheckFreshness(mock.Anything, "user-1").Return([]vault.Version{
		{ID: "v1", Version: 1},
		{ID: "v2", Version: 4},
	}, nil)

	got, err := vault.New(repo).CheckFreshness(context.Background(), "user-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, vault.Version{ID: "v2", Version: 4}, got[1])
}

func TestCheckFreshness_EmptyUserID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRepository(t)
	_, err := vault.New(repo).CheckFreshness(context.Background(), "")
	require.ErrorIs(t, err, vault.ErrEmptyUserID)
}
