package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth/mocks"
)

func TestRefreshToken_Success(t *testing.T) {
	t.Parallel()

	users := mocks.NewMockRepository(t)
	users.EXPECT().GetByID(mock.Anything, "user-1").
		Return(domain.User{ID: "user-1", EncKDFSalt: []byte("salt"), EncKDFParams: []byte(`{"version":1}`)}, nil)

	tokens := mocks.NewMockTokenIssuer(t)
	tokens.EXPECT().Verify("old-refresh").Return("user-1", nil)
	tokens.EXPECT().Refresh("old-refresh").Return("new-access", "new-refresh", nil)

	res, err := newUseCase(users, tokens).RefreshToken(context.Background(), auth.RefreshParams{
		RefreshToken: "old-refresh",
	})

	require.NoError(t, err)
	assert.Equal(t, "new-access", res.AccessToken)
	assert.Equal(t, "new-refresh", res.RefreshToken)
	assert.Equal(t, []byte("salt"), res.EncKDFSalt)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	t.Parallel()

	tokens := mocks.NewMockTokenIssuer(t)
	tokens.EXPECT().Verify("bad-token").Return("", assert.AnError)

	_, err := newUseCase(mocks.NewMockRepository(t), tokens).RefreshToken(context.Background(), auth.RefreshParams{
		RefreshToken: "bad-token",
	})

	assert.ErrorIs(t, err, auth.ErrInvalidRefreshToken)
}

func TestRefreshToken_UserNotFound(t *testing.T) {
	t.Parallel()

	users := mocks.NewMockRepository(t)
	users.EXPECT().GetByID(mock.Anything, "user-1").Return(domain.User{}, auth.ErrUserNotFound)

	tokens := mocks.NewMockTokenIssuer(t)
	tokens.EXPECT().Verify("old-refresh").Return("user-1", nil)

	_, err := newUseCase(users, tokens).RefreshToken(context.Background(), auth.RefreshParams{
		RefreshToken: "old-refresh",
	})

	assert.ErrorIs(t, err, auth.ErrInvalidRefreshToken)
}

func TestRefreshToken_RefreshIssueError(t *testing.T) {
	t.Parallel()

	users := mocks.NewMockRepository(t)
	users.EXPECT().GetByID(mock.Anything, "user-1").Return(domain.User{ID: "user-1"}, nil)

	tokens := mocks.NewMockTokenIssuer(t)
	tokens.EXPECT().Verify("old-refresh").Return("user-1", nil)
	tokens.EXPECT().Refresh("old-refresh").Return("", "", assert.AnError)

	_, err := newUseCase(users, tokens).RefreshToken(context.Background(), auth.RefreshParams{
		RefreshToken: "old-refresh",
	})

	assert.ErrorIs(t, err, auth.ErrInvalidRefreshToken)
}
