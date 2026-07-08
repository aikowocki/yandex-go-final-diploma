package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
)

func TestRegister_Success(t *testing.T) {
	t.Parallel()

	users := mocks.NewMockUserRepository(t)
	users.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, u domain.User) (domain.User, error) {
			assert.Equal(t, "alice", u.Login)
			assert.NotEmpty(t, u.AuthHash) // PHC-строка
			return domain.User{ID: "user-1", Login: u.Login, AuthHash: u.AuthHash}, nil
		})

	tokens := mocks.NewMockTokenIssuer(t)
	tokens.EXPECT().Issue("user-1").Return("access-tok", "refresh-tok", nil)

	res, err := newUseCase(users, tokens).Register(context.Background(), auth.RegisterParams{
		Login:           "alice",
		LoginCredential: []byte("s3cr3t-password"),
	})

	require.NoError(t, err)
	assert.Equal(t, "access-tok", res.AccessToken)
	assert.Equal(t, "refresh-tok", res.RefreshToken)
}

func TestRegister_Errors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("connection refused")
	signErr := errors.New("signing failed")

	tests := []struct {
		name    string
		setup   func(users *mocks.MockUserRepository, tokens *mocks.MockTokenIssuer)
		wantErr error
	}{
		{
			name: "login taken",
			setup: func(users *mocks.MockUserRepository, _ *mocks.MockTokenIssuer) {
				// Issue не настраиваем — при ошибке Create он вызываться не должен.
				users.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.User{}, auth.ErrLoginTaken)
			},
			wantErr: auth.ErrLoginTaken,
		},
		{
			name: "repository error",
			setup: func(users *mocks.MockUserRepository, _ *mocks.MockTokenIssuer) {
				users.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.User{}, repoErr)
			},
			wantErr: repoErr,
		},
		{
			name: "token issue error",
			setup: func(users *mocks.MockUserRepository, tokens *mocks.MockTokenIssuer) {
				users.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.User{ID: "user-1"}, nil)
				tokens.EXPECT().Issue("user-1").Return("", "", signErr)
			},
			wantErr: signErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			users := mocks.NewMockUserRepository(t)
			tokens := mocks.NewMockTokenIssuer(t)
			tt.setup(users, tokens)

			_, err := newUseCase(users, tokens).Register(context.Background(), auth.RegisterParams{
				Login:           "alice",
				LoginCredential: []byte("s3cr3t-password"),
			})

			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestRegister_DifferentLoginCredentialsProduceDifferentHashes проверяет, что
// AuthHash реально зависит от LoginCredential.
func TestRegister_DifferentLoginCredentialsProduceDifferentHashes(t *testing.T) {
	t.Parallel()

	var capturedHashes []string
	users := mocks.NewMockUserRepository(t)
	users.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, u domain.User) (domain.User, error) {
			capturedHashes = append(capturedHashes, u.AuthHash)
			return domain.User{ID: "user-1"}, nil
		})

	tokens := mocks.NewMockTokenIssuer(t)
	tokens.EXPECT().Issue(mock.Anything).Return("a", "r", nil)

	uc := newUseCase(users, tokens)
	_, err := uc.Register(context.Background(), auth.RegisterParams{Login: "a", LoginCredential: []byte("password-one")})
	require.NoError(t, err)
	_, err = uc.Register(context.Background(), auth.RegisterParams{Login: "b", LoginCredential: []byte("password-two")})
	require.NoError(t, err)

	require.Len(t, capturedHashes, 2)
	assert.NotEqual(t, capturedHashes[0], capturedHashes[1])
}
