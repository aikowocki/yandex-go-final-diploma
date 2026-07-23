package cli

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
)

func TestRegisterCmd_Run_SkipEncryption(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().Register(mock.Anything, "alice", []byte("pw")).Return(contracts.Tokens{}, nil)

	scriptSecretsCLI(t, "pw", "pw") // credential + repeat
	scriptLines(t, "n")             // encryption_confirm -> нет

	env := newCLITestEnv(t, server)
	cmd := &RegisterCmd{Login: "alice"}
	require.NoError(t, cmd.Run(env.Auth, env.Localizer))
}

func TestRegisterCmd_Run_WithEncryptionSetup(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().Register(mock.Anything, "alice", []byte("pw")).Return(contracts.Tokens{}, nil)
	server.EXPECT().SetupEncryption(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	server.EXPECT().StoreRecoveryCodes(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	scriptSecretsCLI(t, "pw", "pw", "masterpass", "masterpass")
	scriptLines(t, "y")

	env := newCLITestEnv(t, server)
	cmd := &RegisterCmd{Login: "alice"}
	require.NoError(t, cmd.Run(env.Auth, env.Localizer))
}

func TestRegisterCmd_Run_MismatchedCredential(t *testing.T) {
	server := mocks.NewMockServerClient(t) // Register не должен вызываться
	scriptSecretsCLI(t, "pw1", "pw2", "pw1", "pw2", "pw1", "pw2")

	env := newCLITestEnv(t, server)
	cmd := &RegisterCmd{Login: "alice"}
	require.ErrorIs(t, cmd.Run(env.Auth, env.Localizer), errMismatch)
}

func TestRegisterCmd_Run_RegisterError(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().Register(mock.Anything, "alice", []byte("pw")).Return(contracts.Tokens{}, assertAnErrorCLI())

	scriptSecretsCLI(t, "pw", "pw")

	env := newCLITestEnv(t, server)
	cmd := &RegisterCmd{Login: "alice"}
	require.Error(t, cmd.Run(env.Auth, env.Localizer))
}

func TestSetupEncryptionCmd_Run(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().SetupEncryption(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	server.EXPECT().StoreRecoveryCodes(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	scriptSecretsCLI(t, "masterpass", "masterpass")

	env := newCLITestEnv(t, server)
	cmd := &SetupEncryptionCmd{}
	require.NoError(t, cmd.Run(env.Auth, env.Localizer))
}

// Если GenerateRecoveryCodes падает (например StoreRecoveryCodes вернул ошибку), команда всё
// равно завершается успешно — шифрование уже настроено, отсутствие recovery codes не должно
// откатывать уже выполненную настройку.
func TestSetupEncryptionCmd_Run_RecoveryCodesGenFailedIsNonFatal(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().SetupEncryption(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	server.EXPECT().StoreRecoveryCodes(mock.Anything, mock.Anything, mock.Anything).Return(assertAnErrorCLI())

	scriptSecretsCLI(t, "masterpass", "masterpass")

	env := newCLITestEnv(t, server)
	cmd := &SetupEncryptionCmd{}
	require.NoError(t, cmd.Run(env.Auth, env.Localizer))
}
