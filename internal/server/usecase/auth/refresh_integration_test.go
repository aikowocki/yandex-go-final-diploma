package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/jwt"
)

// Интеграционные тесты RefreshToken с РЕАЛЬНЫМ jwt.TokenIssuer
func realIssuer() *jwt.TokenIssuer {
	return jwt.New([]byte("integration-secret"), time.Minute, time.Hour)
}

func TestRefreshToken_Integration_AcceptsRefreshToken(t *testing.T) {
	t.Parallel()

	issuer := realIssuer()
	_, refresh, err := issuer.Issue("user-1")
	require.NoError(t, err)

	users := mocks.NewMockRepository(t)
	users.EXPECT().GetByID(mock.Anything, "user-1").
		Return(domain.User{ID: "user-1", EncKDFSalt: []byte("salt")}, nil)

	uc := auth.New(users, nil, issuer, passthroughTx{})

	res, err := uc.RefreshToken(context.Background(), auth.RefreshParams{RefreshToken: refresh})
	require.NoError(t, err) // с багом (Verify вместо VerifyRefresh) здесь была бы ошибка
	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)

	// Выданный access-токен валиден и принадлежит тому же пользователю.
	userID, err := issuer.Verify(res.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "user-1", userID)
}

func TestRefreshToken_Integration_RejectsAccessTokenAsRefresh(t *testing.T) {
	t.Parallel()

	issuer := realIssuer()
	access, _, err := issuer.Issue("user-1")
	require.NoError(t, err)

	// GetByID не должен вызываться — проверка типа токена падает раньше.
	users := mocks.NewMockRepository(t)

	uc := auth.New(users, nil, issuer, passthroughTx{})

	_, err = uc.RefreshToken(context.Background(), auth.RefreshParams{RefreshToken: access})
	assert.ErrorIs(t, err, auth.ErrInvalidRefreshToken)
}
