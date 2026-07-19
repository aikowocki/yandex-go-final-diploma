package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
)

func TestEnsureUnlocked_AlreadyUnlocked_NoOp(t *testing.T) {
	env := newCLITestEnv(t, mocks.NewMockServerClient(t))
	env.Session.SetMasterKey(make([]byte, 32))

	require.NoError(t, ensureUnlocked(context.Background(), env.Auth, env.Localizer))
}

func TestEnsureUnlocked_RefreshFailsWithNonNetworkError(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, mock.Anything).Return(contracts.LoginResult{}, assert.AnError)

	env := newCLITestEnv(t, server)
	err := ensureUnlocked(context.Background(), env.Auth, env.Localizer)
	require.Error(t, err)
}

func TestEnsureUnlocked_RefreshSuccess_NoEncryptionConfigured(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, mock.Anything).Return(contracts.LoginResult{
		Tokens: contracts.Tokens{AccessToken: "a", RefreshToken: "r"},
	}, nil)

	env := newCLITestEnv(t, server)
	err := ensureUnlocked(context.Background(), env.Auth, env.Localizer)
	require.ErrorIs(t, err, authuc.ErrEncryptionNotSetup)
}
