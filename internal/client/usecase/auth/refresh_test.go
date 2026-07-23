package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
)

func TestRefresh_Success(t *testing.T) {
	oldTokens := contracts.Tokens{AccessToken: "old-a", RefreshToken: "old-r"}
	res := contracts.LoginResult{
		Tokens:       contracts.Tokens{AccessToken: "new-a", RefreshToken: "new-r", UserID: "u1"},
		EncKDFSalt:   []byte("salt"),
		EncKDFParams: encParamsJSON(t),
		EncMasterKey: []byte("wrapped"),
	}

	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, "old-r").Return(res, nil)

	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Load().Return(oldTokens, nil)
	store.EXPECT().Save(res.Tokens).Return(nil)

	uc := newUseCase(server, store)
	require.NoError(t, uc.Refresh(context.Background()))
	assert.True(t, uc.EncryptionConfigured())
}

func TestRefresh_LoadTokensError(t *testing.T) {
	server := mocks.NewMockServerClient(t) // не должен вызываться
	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Load().Return(contracts.Tokens{}, assert.AnError)

	uc := newUseCase(server, store)
	require.ErrorIs(t, uc.Refresh(context.Background()), assert.AnError)
}

func TestRefresh_ServerError(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, "old-r").Return(contracts.LoginResult{}, assert.AnError)

	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Load().Return(contracts.Tokens{RefreshToken: "old-r"}, nil)

	uc := newUseCase(server, store)
	require.ErrorIs(t, uc.Refresh(context.Background()), assert.AnError)
}

func TestRefresh_SaveTokensError(t *testing.T) {
	res := contracts.LoginResult{Tokens: contracts.Tokens{AccessToken: "new-a", RefreshToken: "new-r"}}

	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, "old-r").Return(res, nil)

	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Load().Return(contracts.Tokens{RefreshToken: "old-r"}, nil)
	store.EXPECT().Save(res.Tokens).Return(assert.AnError)

	uc := newUseCase(server, store)
	require.Error(t, uc.Refresh(context.Background()))
}
