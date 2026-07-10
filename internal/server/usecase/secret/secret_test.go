package secret_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret/mocks"
)

func validCreateParams() secret.CreateParams {
	return secret.CreateParams{
		UserID:     "user-1",
		VaultID:    "vault-1",
		Type:       domain.SecretTypeLoginPassword,
		EncRow:     []byte("enc-row"),
		EncIndex:   []byte("enc-index"),
		EncPayload: []byte("enc-payload"),
	}
}

func TestCreateSecret_Success(t *testing.T) {
	t.Parallel()

	vaults := mocks.NewMockVaultOwnership(t)
	vaults.EXPECT().IsOwner(mock.Anything, "vault-1", "user-1").Return(true, nil)

	secrets := mocks.NewMockRepository(t)
	secrets.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, s domain.Secret) (domain.Secret, error) {
			assert.Equal(t, "vault-1", s.VaultID)
			assert.Equal(t, domain.SecretTypeLoginPassword, s.Type)
			s.ID = "secret-1"
			return s, nil
		})

	id, err := secret.New(secrets, vaults).CreateSecret(context.Background(), validCreateParams())
	require.NoError(t, err)
	assert.Equal(t, "secret-1", id)
}

func TestCreateSecret_NotOwner(t *testing.T) {
	t.Parallel()

	vaults := mocks.NewMockVaultOwnership(t)
	vaults.EXPECT().IsOwner(mock.Anything, "vault-1", "user-1").Return(false, nil)

	// Секрет создаваться не должен — репозиторий не вызывается.
	secrets := mocks.NewMockRepository(t)

	_, err := secret.New(secrets, vaults).CreateSecret(context.Background(), validCreateParams())
	require.ErrorIs(t, err, secret.ErrVaultNotFound)
}

func TestCreateSecret_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*secret.CreateParams)
		wantErr error
	}{
		{"empty user id", func(p *secret.CreateParams) { p.UserID = "" }, secret.ErrEmptyUserID},
		{"empty vault id", func(p *secret.CreateParams) { p.VaultID = "" }, secret.ErrEmptyVaultID},
		{"empty enc row", func(p *secret.CreateParams) { p.EncRow = nil }, secret.ErrEmptyEncRow},
		{"empty enc index", func(p *secret.CreateParams) { p.EncIndex = nil }, secret.ErrEmptyEncIndex},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Ни проверка владения, ни создание не должны вызываться при провале валидации.
			vaults := mocks.NewMockVaultOwnership(t)
			secrets := mocks.NewMockRepository(t)
			params := validCreateParams()
			tt.mutate(&params)

			_, err := secret.New(secrets, vaults).CreateSecret(context.Background(), params)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestCreateSecret_RepoError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("insert failed")
	vaults := mocks.NewMockVaultOwnership(t)
	vaults.EXPECT().IsOwner(mock.Anything, "vault-1", "user-1").Return(true, nil)
	secrets := mocks.NewMockRepository(t)
	secrets.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.Secret{}, wantErr)

	_, err := secret.New(secrets, vaults).CreateSecret(context.Background(), validCreateParams())
	assert.ErrorIs(t, err, wantErr)
}

func TestListRow_Success(t *testing.T) {
	t.Parallel()

	secrets := mocks.NewMockRepository(t)
	secrets.EXPECT().ListRow(mock.Anything, "vault-1", "user-1").Return([]domain.Secret{
		{ID: "s1", Type: domain.SecretTypeLoginPassword, Version: 1, EncRow: []byte("r1")},
	}, nil)

	got, err := secret.New(secrets, mocks.NewMockVaultOwnership(t)).ListRow(context.Background(), "user-1", "vault-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, secret.Row{ID: "s1", Type: domain.SecretTypeLoginPassword, Version: 1, EncRow: []byte("r1")}, got[0])
}

func TestListRow_Validation(t *testing.T) {
	t.Parallel()

	uc := secret.New(mocks.NewMockRepository(t), mocks.NewMockVaultOwnership(t))

	_, err := uc.ListRow(context.Background(), "", "vault-1")
	require.ErrorIs(t, err, secret.ErrEmptyUserID)

	_, err = uc.ListRow(context.Background(), "user-1", "")
	require.ErrorIs(t, err, secret.ErrEmptyVaultID)
}

func TestGetPayload_Success(t *testing.T) {
	t.Parallel()

	secrets := mocks.NewMockRepository(t)
	secrets.EXPECT().GetPayload(mock.Anything, "secret-1", "user-1").Return(domain.Secret{
		ID: "secret-1", Type: domain.SecretTypeLoginPassword, Version: 2, EncPayload: []byte("p"),
	}, nil)

	got, err := secret.New(secrets, mocks.NewMockVaultOwnership(t)).GetPayload(context.Background(), "user-1", "secret-1")
	require.NoError(t, err)
	assert.Equal(t, secret.Payload{ID: "secret-1", Type: domain.SecretTypeLoginPassword, Version: 2, EncPayload: []byte("p")}, got)
}

func TestGetPayload_NotFound(t *testing.T) {
	t.Parallel()

	secrets := mocks.NewMockRepository(t)
	secrets.EXPECT().GetPayload(mock.Anything, "secret-x", "user-1").Return(domain.Secret{}, secret.ErrSecretNotFound)

	_, err := secret.New(secrets, mocks.NewMockVaultOwnership(t)).GetPayload(context.Background(), "user-1", "secret-x")
	require.ErrorIs(t, err, secret.ErrSecretNotFound)
}

func TestGetPayload_Validation(t *testing.T) {
	t.Parallel()

	uc := secret.New(mocks.NewMockRepository(t), mocks.NewMockVaultOwnership(t))

	_, err := uc.GetPayload(context.Background(), "", "secret-1")
	require.ErrorIs(t, err, secret.ErrEmptyUserID)

	_, err = uc.GetPayload(context.Background(), "user-1", "")
	require.ErrorIs(t, err, secret.ErrEmptySecretID)
}
