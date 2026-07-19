package cli

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
)

func TestLoginCmd_Run_EncryptionConfigured_UnlocksSuccessfully(t *testing.T) {
	salt := mustSaltCLI(t)
	params := testParamsJSONCLI(t)
	wrapped := mustWrappedKeyCLI(t, "master-pass", salt)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().Login(mock.Anything, "alice", []byte("pw")).Return(contracts.LoginResult{
		EncKDFSalt: salt, EncKDFParams: params, EncMasterKey: wrapped,
	}, nil)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	scriptSecretsCLI(t, "pw", "master-pass")

	env := newCLITestEnv(t, server)
	cmd := &LoginCmd{Login: "alice"}
	require.NoError(t, cmd.Run(env.Auth, env.Localizer))
	require.True(t, env.Session.Unlocked())
}

func TestLoginCmd_Run_PromptsForLoginWhenEmpty(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().Login(mock.Anything, "bob", []byte("pw")).Return(contracts.LoginResult{}, nil)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	scriptLines(t, "bob") // логин запрашивается
	scriptSecretsCLI(t, "pw")

	env := newCLITestEnv(t, server)
	cmd := &LoginCmd{}
	// Шифрование не настроено -> encryption_confirm запрашивается через promptConfirm (readLineFn),
	// отвечаем "n" третьей строкой.
	orig := readLineFn
	calls := 0
	answers := []string{"bob", "n"}
	readLineFn = func(string) (string, error) {
		if calls >= len(answers) {
			return "", nil
		}
		a := answers[calls]
		calls++
		return a, nil
	}
	t.Cleanup(func() { readLineFn = orig })

	require.NoError(t, cmd.Run(env.Auth, env.Localizer))
}

func TestLoginCmd_Run_LoginError(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().Login(mock.Anything, "alice", []byte("wrong")).Return(contracts.LoginResult{}, assertAnErrorCLI())

	scriptSecretsCLI(t, "wrong")

	env := newCLITestEnv(t, server)
	cmd := &LoginCmd{Login: "alice"}
	require.Error(t, cmd.Run(env.Auth, env.Localizer))
}
