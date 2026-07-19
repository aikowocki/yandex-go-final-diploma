package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
)

func TestPromptOTPCodes_EmptyLineEnds(t *testing.T) {
	scriptLines(t, "")
	codes, err := promptOTPCodes(testLocalizer())
	require.NoError(t, err)
	assert.Empty(t, codes)
}

func TestPromptOTPCodes_CollectsCodes(t *testing.T) {
	scriptLines(t, "111111", "222222", "")
	codes, err := promptOTPCodes(testLocalizer())
	require.NoError(t, err)
	require.Len(t, codes, 2)
	assert.Equal(t, "111111", codes[0].Code)
	assert.Equal(t, "222222", codes[1].Code)
}

func TestPrintOTPCodes_Empty(t *testing.T) {
	printOTPCodes(testLocalizer(), nil) // не должно паниковать
}

func TestPrintOTPCodes_WithCodes(t *testing.T) {
	printOTPCodes(testLocalizer(), []secretcontent.OTPCode{
		{Code: "111111", Used: false},
		{Code: "222222", Used: true},
	})
}

func TestOTPUseCmd_RequiresUnlock(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, mock.Anything).Return(contracts.LoginResult{}, assertAnErrorCLI())

	env := newCLITestEnv(t, server)
	cmd := &OTPUseCmd{Vault: "Personal", ID: "s1", Index: 1}
	require.Error(t, cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer))
}

func TestOTPUseCmd_VaultNotFound(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	env := newCLITestEnv(t, server)
	env.Session.SetMasterKey(make([]byte, 32))

	cmd := &OTPUseCmd{Vault: "Unknown", ID: "s1", Index: 1}
	err := cmd.Run(env.Auth, env.Vault, env.Secret, env.Localizer)
	require.Error(t, err)
	assert.ErrorIs(t, err, errVaultNotFound)
}
