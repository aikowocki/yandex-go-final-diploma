package cli

import (
	"bytes"
	"crypto/sha512"
	"io"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/hkdf"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
)

// mustRecoveryBlobCLI воспроизводит серверную часть флоу recovery codes — шифрует фиктивный MasterKey
// под ключом, выведенным из normalizedCode тем же HKDF-путём, чтобы RecoverCmd мог его
// расшифровать через RecoverWithCode в тесте.
func mustRecoveryBlobCLI(t *testing.T, normalizedCode string) []byte {
	t.Helper()
	hkdfReader := hkdf.New(sha512.New, []byte(normalizedCode), nil, []byte("gophkeeper-recovery-v1"))
	recoveryKey := make([]byte, 32)
	_, err := io.ReadFull(hkdfReader, recoveryKey)
	require.NoError(t, err)

	c := cryptoimpl.Crypto{}
	masterKey := bytes.Repeat([]byte{7}, 32)
	blob, err := c.WrapVaultKey(masterKey, recoveryKey)
	require.NoError(t, err)
	return blob
}

func TestRecoverCmd_Run_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, mock.Anything).Return(contracts.LoginResult{
		Tokens: contracts.Tokens{UserID: "user-1"},
	}, nil)
	server.EXPECT().GetRecoveryBlob(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assertAnErrorCLI()) // расшифровка провалится — этого достаточно для проверки маршрута

	env := newCLITestEnv(t, server)
	cmd := &RecoverCmd{Code: "AAAA-BBBB-CCCC-DDDD"}
	// RecoverWithCode вернёт ошибку (GetRecoveryBlob упал) — команда должна её вернуть, а не
	// провалиться где-то до этого (проверяем, что Refresh/маршрутизация кода отработали).
	require.Error(t, cmd.Run(env.Auth, env.Localizer))
}

func TestRecoverCmd_Run_PromptsForCodeWhenEmpty(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, mock.Anything).Return(contracts.LoginResult{
		Tokens: contracts.Tokens{UserID: "user-1"},
	}, nil)
	server.EXPECT().GetRecoveryBlob(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assertAnErrorCLI())

	scriptLines(t, "AAAA-BBBB-CCCC-DDDD")

	env := newCLITestEnv(t, server)
	cmd := &RecoverCmd{}
	require.Error(t, cmd.Run(env.Auth, env.Localizer))
}

func TestRecoverCmd_Run_RefreshFailsPropagatesError(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, mock.Anything).Return(contracts.LoginResult{}, assertAnErrorCLI())

	env := newCLITestEnv(t, server)
	cmd := &RecoverCmd{Code: "AAAA-BBBB-CCCC-DDDD"}
	require.Error(t, cmd.Run(env.Auth, env.Localizer))
}

// Успешное восстановление MasterKey должно немедленно продолжиться в runSetupEncryption
// (новый мастер-пароль), иначе восстановленный MasterKey потерялся бы вместе с процессом.
func TestRecoverCmd_Run_SuccessContinuesToSetupEncryption(t *testing.T) {
	salt := mustSaltCLI(t)
	recoveryKeyBlob := mustRecoveryBlobCLI(t, "AAAABBBBCCCCDDDD")

	server := mocks.NewMockServerClient(t)
	server.EXPECT().RefreshToken(mock.Anything, mock.Anything).Return(contracts.LoginResult{
		Tokens: contracts.Tokens{UserID: "user-1"},
	}, nil)
	server.EXPECT().GetRecoveryBlob(mock.Anything, mock.Anything, mock.Anything).Return(recoveryKeyBlob, nil)
	server.EXPECT().MarkRecoveryCodeUsed(mock.Anything, mock.Anything, mock.Anything).Return(nil)
	server.EXPECT().SetupEncryption(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	server.EXPECT().StoreRecoveryCodes(mock.Anything, mock.Anything, mock.Anything).Return(nil)
	_ = salt

	scriptSecretsCLI(t, "newmasterpass", "newmasterpass") // runSetupEncryption prompts

	env := newCLITestEnv(t, server)
	cmd := &RecoverCmd{Code: "AAAA-BBBB-CCCC-DDDD"}
	require.NoError(t, cmd.Run(env.Auth, env.Localizer))
}
