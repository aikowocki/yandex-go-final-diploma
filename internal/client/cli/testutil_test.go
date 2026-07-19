package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	clienti18n "github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	syncuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/sync"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// cliTestEnv собирает реальные usecase (auth/vault/secret/sync) поверх мок ServerClient и
// in-memory localstore.
type cliTestEnv struct {
	Auth      *authuc.UseCase
	Vault     *vaultuc.UseCase
	Secret    *secretuc.UseCase
	Sync      *syncuc.UseCase
	Localizer *clienti18n.Localizer
	Server    *mocks.MockServerClient
	Session   *session.Session
	Local     *localstore.Store
}

func newCLITestEnv(t *testing.T, server *mocks.MockServerClient) *cliTestEnv {
	t.Helper()

	local, err := localstore.Open("", false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = local.Close() })

	sess := session.New()
	crypt := cryptoimpl.Crypto{}
	tokens := mocks.NewMockTokenStore(t)
	tokens.EXPECT().Save(mock.Anything).Return(nil).Maybe()
	tokens.EXPECT().Load().Return(contracts.Tokens{}, nil).Maybe()
	tokens.EXPECT().Clear().Return(nil).Maybe()

	auth := authuc.New(server, crypt, crypt, tokens, sess, local)
	vault := vaultuc.New(server, crypt, tokens, sess, local)
	secret := secretuc.New(server, crypt, tokens, sess, local, t.TempDir())
	sync := syncuc.New(server, local, tokens)

	bundle := clienti18n.NewBundle()
	localizer := clienti18n.NewLocalizer(bundle, "en")

	return &cliTestEnv{
		Auth: auth, Vault: vault, Secret: secret, Sync: sync,
		Localizer: localizer, Server: server, Session: sess, Local: local,
	}
}

// scriptLines подменяет readLineFn последовательностью заранее заданных ответов (в порядке
// вызовов promptLine). Пустая строка возвращается для всех вызовов сверх заданного списка
// (что удобно для опциональных полей типа tags/note, которые обычно оставляют пустыми).
func scriptLines(t *testing.T, lines ...string) {
	t.Helper()
	orig := readLineFn
	t.Cleanup(func() { readLineFn = orig })

	i := 0
	readLineFn = func(string) (string, error) {
		if i >= len(lines) {
			return "", nil
		}
		s := lines[i]
		i++
		return s, nil
	}
}

// scriptSecretsCLI подменяет readSecretFn последовательностью заданных значений (без ошибок).
func scriptSecretsCLI(t *testing.T, values ...string) {
	t.Helper()
	orig := readSecretFn
	t.Cleanup(func() { readSecretFn = orig })

	i := 0
	readSecretFn = func(string) ([]byte, error) {
		if i >= len(values) {
			return nil, nil
		}
		v := values[i]
		i++
		return []byte(v), nil
	}
}

// --- crypto helpers (тот же паттерн, что в internal/client/usecase/auth и internal/client/tui) ---

func testParamsCLI() crypto.Params {
	return crypto.Params{
		Version:     crypto.ParamsVersionV1,
		Memory:      8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		KeyLen:      32,
	}
}

func testParamsJSONCLI(t *testing.T) []byte {
	t.Helper()
	data, err := json.Marshal(testParamsCLI())
	require.NoError(t, err)
	return data
}

func mustSaltCLI(t *testing.T) []byte {
	t.Helper()
	salt, err := crypto.GenerateSalt()
	require.NoError(t, err)
	return salt
}

func mustWrappedKeyCLI(t *testing.T, passphrase string, salt []byte) []byte {
	t.Helper()
	c := cryptoimpl.Crypto{}
	seed, err := c.DeriveMasterSeed([]byte(passphrase), salt, testParamsCLI())
	require.NoError(t, err)
	kek, err := c.DeriveMasterKey(seed)
	require.NoError(t, err)
	masterKey := bytes.Repeat([]byte{42}, 32)
	wrapped, err := c.WrapVaultKey(masterKey, kek)
	require.NoError(t, err)
	return wrapped
}

func assertAnErrorCLI() error {
	return errors.New("boom")
}

func loginPasswordInputCLI(title, username, password string) secretuc.CreateLoginPasswordInput {
	return secretuc.CreateLoginPasswordInput{Title: title, Username: username, Password: password}
}
