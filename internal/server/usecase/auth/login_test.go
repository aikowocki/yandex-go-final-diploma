package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alexedwards/argon2id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
)

func mustHash(t *testing.T, credential string) string {
	t.Helper()
	hash, err := argon2id.CreateHash(credential, argon2id.DefaultParams)
	require.NoError(t, err)
	return hash
}

func TestLogin_Success(t *testing.T) {
	t.Parallel()

	hash := mustHash(t, "correct-password")
	users := mocks.NewMockUserRepository(t)
	users.EXPECT().GetByLogin(mock.Anything, "alice").Return(domain.User{
		ID:           "user-1",
		Login:        "alice",
		AuthHash:     hash,
		EncKDFSalt:   []byte("salt"),
		EncKDFParams: []byte(`{"version":1}`),
	}, nil)

	tokens := mocks.NewMockTokenIssuer(t)
	tokens.EXPECT().Issue("user-1").Return("access-tok", "refresh-tok", nil)

	res, err := newUseCase(users, tokens).Login(context.Background(), auth.LoginParams{
		Login:           "alice",
		LoginCredential: []byte("correct-password"),
	})

	require.NoError(t, err)
	assert.Equal(t, "access-tok", res.AccessToken)
	assert.Equal(t, "refresh-tok", res.RefreshToken)
	assert.Equal(t, []byte("salt"), res.EncKDFSalt)
	assert.Equal(t, []byte(`{"version":1}`), res.EncKDFParams)
}

func TestLogin_Errors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("connection refused")
	validHash := mustHash(t, "correct-password")

	tests := []struct {
		name string
		// setup настраивает репозиторий на конкретный сценарий.
		setup func(users *mocks.MockUserRepository)
		// credential — пароль, с которым пытаемся войти.
		credential string
		// wantIs — ошибка, которой результат ДОЛЖЕН соответствовать (errors.Is). Может быть nil.
		wantIs error
		// wantNotIs — ошибка, которой результат НЕ должен соответствовать. Может быть nil.
		wantNotIs error
	}{
		{
			// user enumeration: "не найден" неотличим от "неверный пароль".
			name: "user not found is masked as invalid credentials",
			setup: func(users *mocks.MockUserRepository) {
				users.EXPECT().GetByLogin(mock.Anything, "ghost").Return(domain.User{}, auth.ErrUserNotFound)
			},
			credential: "whatever",
			wantIs:     auth.ErrInvalidCredentials,
			wantNotIs:  auth.ErrUserNotFound,
		},
		{
			name: "wrong password returns invalid credentials",
			setup: func(users *mocks.MockUserRepository) {
				users.EXPECT().GetByLogin(mock.Anything, "ghost").Return(domain.User{ID: "user-1", AuthHash: validHash}, nil)
			},
			credential: "wrong-password",
			wantIs:     auth.ErrInvalidCredentials,
		},
		{
			// Инфраструктурные ошибки не маскируются под "неверные данные".
			name: "repository error is not disguised",
			setup: func(users *mocks.MockUserRepository) {
				users.EXPECT().GetByLogin(mock.Anything, "ghost").Return(domain.User{}, repoErr)
			},
			credential: "whatever",
			wantIs:     repoErr,
			wantNotIs:  auth.ErrInvalidCredentials,
		},
		{
			// Битый хеш в БД — внутренняя ошибка, а не "неверный пароль".
			name: "malformed stored hash is internal error",
			setup: func(users *mocks.MockUserRepository) {
				users.EXPECT().GetByLogin(mock.Anything, "ghost").Return(domain.User{ID: "user-1", AuthHash: "not-a-phc-hash"}, nil)
			},
			credential: "whatever",
			wantNotIs:  auth.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			users := mocks.NewMockUserRepository(t)
			tt.setup(users)

			_, err := newUseCase(users, mocks.NewMockTokenIssuer(t)).Login(context.Background(), auth.LoginParams{
				Login:           "ghost",
				LoginCredential: []byte(tt.credential),
			})

			require.Error(t, err)
			if tt.wantIs != nil {
				assert.ErrorIs(t, err, tt.wantIs)
			}
			if tt.wantNotIs != nil {
				assert.NotErrorIs(t, err, tt.wantNotIs)
			}
		})
	}
}
