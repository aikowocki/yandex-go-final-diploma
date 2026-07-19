package tui

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	zone "github.com/lrstanley/bubblezone"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/i18n"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	secretuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	syncuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/sync"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// TestMain инициализирует глобальный bubblezone.Manager (как это делает cmd/client/main.go
// перед запуском tea.Program) — без этого view()-методы, использующие zone.Mark/zone.Scan
// (dashboard_table.go), паникуют с "manager not initialized".
func TestMain(m *testing.M) {
	zone.NewGlobal()
	os.Exit(m.Run())
}

// newTestContainer собирает минимальный app.Container для юнит-тестов моделей TUI: реальные
// crypto/session/localstore (in-memory SQLite) + переданный мок ServerClient вместо сети.
// Это позволяет тестировать Update()-логику экранов без поднятого сервера/докера, в отличие
// от e2e_manual_test.go (build tag e2e), который гоняет полный сценарий против реального grpc.
func newTestContainer(t testing.TB, server contracts.ServerClient) *app.Container {
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
	secretUC := secretuc.New(server, crypt, tokens, sess, local, t.TempDir())
	vaultUC := vaultuc.New(server, crypt, tokens, sess, local)
	syncUC := syncuc.New(server, local, tokens)

	bundle := i18n.NewBundle()
	localizer := i18n.NewLocalizer(bundle, "en")

	return &app.Container{
		Config:    &config.ClientConfig{Lang: "en", AutolockMinutes: 5},
		Local:     local,
		Session:   sess,
		Auth:      auth,
		Secret:    secretUC,
		Vault:     vaultUC,
		Sync:      syncUC,
		Localizer: localizer,
	}
}

// newTestContainerWith — вариант newTestContainer с явно переданными local/session (для
// тестов, которым нужно заранее наполнить localstore или открыть конкретный vault в сессии,
// например сценарии outbox-конфликтов).
func newTestContainerWith(t testing.TB, server contracts.ServerClient, local *localstore.Store, sess *session.Session) *app.Container {
	t.Helper()

	crypt := cryptoimpl.Crypto{}
	tokens := mocks.NewMockTokenStore(t)
	tokens.EXPECT().Save(mock.Anything).Return(nil).Maybe()
	tokens.EXPECT().Load().Return(contracts.Tokens{AccessToken: "tok"}, nil).Maybe()
	tokens.EXPECT().Clear().Return(nil).Maybe()
	auth := authuc.New(server, crypt, crypt, tokens, sess, local)
	secretUC := secretuc.New(server, crypt, tokens, sess, local, t.TempDir())
	vaultUC := vaultuc.New(server, crypt, tokens, sess, local)
	syncUC := syncuc.New(server, local, tokens)

	bundle := i18n.NewBundle()
	localizer := i18n.NewLocalizer(bundle, "en")

	return &app.Container{
		Config:    &config.ClientConfig{Lang: "en", AutolockMinutes: 5},
		Local:     local,
		Session:   sess,
		Auth:      auth,
		Secret:    secretUC,
		Vault:     vaultUC,
		Sync:      syncUC,
		Localizer: localizer,
	}
}

// testParams — облегчённые параметры Argon2id для юнит-тестов (быстрый KDF).
func testParams() crypto.Params {
	return crypto.Params{
		Version:     crypto.ParamsVersionV1,
		Memory:      8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		KeyLen:      32,
	}
}

func testParamsJSON(t *testing.T) []byte {
	t.Helper()
	data, err := json.Marshal(testParams())
	require.NoError(t, err)
	return data
}

func mustSalt(t *testing.T) []byte {
	t.Helper()
	salt, err := crypto.GenerateSalt()
	require.NoError(t, err)
	return salt
}

// mustWrappedKey оборачивает фиксированный «MasterKey» KEK-ом, выведенным из passphrase+salt
// (та же схема, что и настоящий SetupEncryption/Login на другом устройстве).
func mustWrappedKey(t *testing.T, passphrase string, salt []byte) []byte {
	t.Helper()
	c := cryptoimpl.Crypto{}
	seed, err := c.DeriveMasterSeed([]byte(passphrase), salt, testParams())
	require.NoError(t, err)
	kek, err := c.DeriveMasterKey(seed)
	require.NoError(t, err)
	masterKey := bytes.Repeat([]byte{42}, 32)
	wrapped, err := c.WrapVaultKey(masterKey, kek)
	require.NoError(t, err)
	return wrapped
}
