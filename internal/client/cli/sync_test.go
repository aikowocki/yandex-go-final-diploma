package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

func TestSyncCmd_Run_RequiresUnlock(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, mock.Anything).Return(contracts.LoginResult{}, assertAnErrorCLI())

	env := newCLITestEnv(t, server)
	cmd := &SyncCmd{}
	require.Error(t, cmd.Run(env.Auth, env.Sync, env.Localizer))
}

func TestRunSync_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CheckFreshness(mock.Anything, mock.Anything).Return(nil, nil)

	env := newCLITestEnv(t, server)
	require.NoError(t, runSync(context.Background(), env.Sync, env.Localizer))
}

func TestRunSync_OfflineIsNotError(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CheckFreshness(mock.Anything, mock.Anything).Return(nil, grpcclient.ErrUnavailable)

	env := newCLITestEnv(t, server)
	require.NoError(t, runSync(context.Background(), env.Sync, env.Localizer))
}

func TestRunSync_OtherErrorPropagates(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CheckFreshness(mock.Anything, mock.Anything).Return(nil, assertAnErrorCLI())

	env := newCLITestEnv(t, server)
	require.Error(t, runSync(context.Background(), env.Sync, env.Localizer))
}
