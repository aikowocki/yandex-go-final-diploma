package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	authuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

func newUseCase(server contracts.ServerClient, store contracts.TokenStore) *authuc.UseCase {
	return authuc.New(server, cryptoimpl.Crypto{}, cryptoimpl.Crypto{}, store, session.New(), memStore())
}

// memStore открывает in-memory localstore для тестов (kv-кеш KDF-параметров).
func memStore() *localstore.Store {
	ls, err := localstore.Open("", false)
	if err != nil {
		panic(err)
	}
	return ls
}

// testParams — облегчённые параметры Argon2id для юнит-тестов: KDF гоняется по-настоящему, но быстро.
func testParams() crypto.Params {
	return crypto.Params{
		Version:     crypto.ParamsVersionV1,
		Memory:      8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		KeyLen:      32,
	}
}

func encParamsJSON(t *testing.T) []byte {
	t.Helper()
	data, err := json.Marshal(testParams())
	require.NoError(t, err)
	return data
}

// makeEncMasterKey строит enc_master_key: оборачивает известный (фиксированный) MasterKey
// ключом KEK, выведенным из passphrase+salt+params. Возвращает обёртку и сам MasterKey.
// Имитирует то, что делает SetupEncryption на другом устройстве.
func makeEncMasterKey(t *testing.T, passphrase string, salt, paramsJSON []byte) (encMK, masterKey []byte) {
	t.Helper()
	var params crypto.Params
	require.NoError(t, json.Unmarshal(paramsJSON, &params))
	c := cryptoimpl.Crypto{}
	seed, err := c.DeriveMasterSeed([]byte(passphrase), salt, params)
	require.NoError(t, err)
	kek, err := c.DeriveMasterKey(seed)
	require.NoError(t, err)
	masterKey = bytes.Repeat([]byte{42}, 32) // фиксированный «случайный» MasterKey
	encMK, err = c.WrapVaultKey(masterKey, kek)
	require.NoError(t, err)
	return encMK, masterKey
}

func TestRegister_SavesTokens(t *testing.T) {
	tokens := contracts.Tokens{AccessToken: "a", RefreshToken: "r"}

	server := mocks.NewMockServerClient(t)
	server.EXPECT().Register(mock.Anything, "alice", []byte("pw")).Return(tokens, nil)

	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Save(tokens).Return(nil)

	uc := newUseCase(server, store)

	require.NoError(t, uc.Register(context.Background(), "alice", []byte("pw")))
}

func TestRegister_Validation(t *testing.T) {
	tests := []struct {
		name  string
		login string
		cred  []byte
		want  error
	}{
		{"empty login", "", []byte("pw"), authuc.ErrEmptyLogin},
		{"empty credential", "alice", nil, authuc.ErrEmptyCredential},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Без EXPECT: любой вызов мока приведёт к падению теста —
			// это и проверяет, что валидация происходит до сети/хранилища.
			server := mocks.NewMockServerClient(t)
			store := mocks.NewMockTokenStore(t)
			uc := newUseCase(server, store)

			err := uc.Register(context.Background(), tt.login, tt.cred)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestLogin_ConfiguredEncryption(t *testing.T) {
	res := contracts.LoginResult{
		Tokens:       contracts.Tokens{AccessToken: "a", RefreshToken: "r"},
		EncKDFSalt:   bytes.Repeat([]byte{1}, 16),
		EncKDFParams: encParamsJSON(t),
		EncMasterKey: bytes.Repeat([]byte{2}, 60),
	}

	server := mocks.NewMockServerClient(t)
	server.EXPECT().Login(mock.Anything, "alice", []byte("pw")).Return(res, nil)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Save(res.Tokens).Return(nil)

	uc := newUseCase(server, store)

	require.NoError(t, uc.Login(context.Background(), "alice", []byte("pw")))
	assert.True(t, uc.EncryptionConfigured())
}

func TestLogin_EncryptionNotConfigured(t *testing.T) {
	res := contracts.LoginResult{Tokens: contracts.Tokens{AccessToken: "a", RefreshToken: "r"}}

	server := mocks.NewMockServerClient(t)
	server.EXPECT().Login(mock.Anything, "alice", []byte("pw")).Return(res, nil)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Save(res.Tokens).Return(nil)

	uc := newUseCase(server, store)

	require.NoError(t, uc.Login(context.Background(), "alice", []byte("pw")))
	assert.False(t, uc.EncryptionConfigured())
}

func TestLogin_Validation(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	store := mocks.NewMockTokenStore(t)
	uc := newUseCase(server, store)

	err := uc.Login(context.Background(), "alice", nil)
	require.ErrorIs(t, err, authuc.ErrEmptyCredential)
}

func TestUnlock_WithoutLogin(t *testing.T) {
	uc := newUseCase(mocks.NewMockServerClient(t), mocks.NewMockTokenStore(t))

	err := uc.Unlock(context.Background(), []byte("pass"))
	require.ErrorIs(t, err, authuc.ErrEncryptionNotSetup)
}

func TestUnlock_EmptyPassphrase(t *testing.T) {
	uc := newUseCase(mocks.NewMockServerClient(t), mocks.NewMockTokenStore(t))

	err := uc.Unlock(context.Background(), nil)
	require.ErrorIs(t, err, authuc.ErrEmptyPassphrase)
}

func TestSetupEncryption_SendsParamsAndDerivesKey(t *testing.T) {
	var gotSalt, gotParams []byte

	var gotEncMK []byte
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		SetupEncryption(mock.Anything, "access-1", mock.Anything, mock.Anything, mock.Anything).
		Run(func(_ context.Context, _ string, salt, params, encMK []byte) {
			gotSalt = salt
			gotParams = params
			gotEncMK = encMK
		}).
		Return(nil)

	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Load().Return(contracts.Tokens{AccessToken: "access-1"}, nil)

	uc := newUseCase(server, store)

	require.NoError(t, uc.SetupEncryption(context.Background(), []byte("passphrase")))
	assert.NotEmpty(t, gotSalt, "salt must be sent to server")
	assert.NotEmpty(t, gotParams, "params must be sent to server")
	assert.NotEmpty(t, gotEncMK, "enc_master_key must be sent to server")
	assert.NotEmpty(t, uc.MasterKeyForTest(), "master key must be in session")
}

func TestSetupEncryption_EmptyPassphrase(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	uc := newUseCase(server, mocks.NewMockTokenStore(t))

	err := uc.SetupEncryption(context.Background(), nil)
	require.ErrorIs(t, err, authuc.ErrEmptyPassphrase)
}

// TestUnlock_Envelope: верная passphrase разворачивает enc_master_key → тот самый MasterKey;
// неверная passphrase → ошибка (KEK не тот, AEAD не открывается). MasterKey НЕ выводится из
// пароля напрямую — он случайный и обёрнут KEK.
func TestUnlock_Envelope(t *testing.T) {
	salt := bytes.Repeat([]byte{7}, 16)
	params := encParamsJSON(t)
	encMK, wantMK := makeEncMasterKey(t, "correct-horse", salt, params)

	res := contracts.LoginResult{
		Tokens:       contracts.Tokens{AccessToken: "a", RefreshToken: "r"},
		EncKDFSalt:   salt,
		EncKDFParams: params,
		EncMasterKey: encMK,
	}

	newUC := func() *authuc.UseCase {
		server := mocks.NewMockServerClient(t)
		server.EXPECT().Login(mock.Anything, "alice", []byte("pw")).Return(res, nil)
		server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
		store := mocks.NewMockTokenStore(t)
		store.EXPECT().Save(res.Tokens).Return(nil)
		uc := newUseCase(server, store)
		require.NoError(t, uc.Login(context.Background(), "alice", []byte("pw")))
		return uc
	}

	// Верный пароль → получаем исходный MasterKey.
	uc := newUC()
	require.NoError(t, uc.Unlock(context.Background(), []byte("correct-horse")))
	assert.Equal(t, wantMK, uc.MasterKeyForTest(), "correct passphrase must recover the master key")

	// Неверный пароль → ошибка, MasterKey не установлен.
	uc2 := newUC()
	err := uc2.Unlock(context.Background(), []byte("wrong"))
	require.Error(t, err)
	assert.False(t, uc2.MasterKeySet(), "wrong passphrase must not set master key")
}

// TestOfflineUnlock_FromCachedKDF: Login кеширует KDF-параметры локально; новый процесс
// (свежая сессия, тот же localstore) может поднять их из кеша и разблокироваться без сети.
func TestOfflineUnlock_FromCachedKDF(t *testing.T) {
	local := memStore()
	salt := bytes.Repeat([]byte{9}, 16)
	params := encParamsJSON(t)
	encMK, wantMK := makeEncMasterKey(t, "correct-horse", salt, params)
	res := contracts.LoginResult{
		Tokens:       contracts.Tokens{AccessToken: "a", RefreshToken: "r"},
		EncKDFSalt:   salt,
		EncKDFParams: params,
		EncMasterKey: encMK,
	}

	// Онлайн-логин — параметры KDF оседают в localstore.
	server := mocks.NewMockServerClient(t)
	server.EXPECT().Login(mock.Anything, "alice", []byte("pw")).Return(res, nil)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Save(res.Tokens).Return(nil)

	online := authuc.New(server, cryptoimpl.Crypto{}, cryptoimpl.Crypto{}, store, session.New(), local)
	require.NoError(t, online.Login(context.Background(), "alice", []byte("pw")))
	require.NoError(t, online.Unlock(context.Background(), []byte("correct-horse")))
	require.Equal(t, wantMK, online.MasterKeyForTest())

	// Новый процесс: сеть недоступна, но кеш KDF+enc_master_key на месте → офлайн-разблокировка.
	offline := authuc.New(mocks.NewMockServerClient(t), cryptoimpl.Crypto{}, cryptoimpl.Crypto{}, mocks.NewMockTokenStore(t), session.New(), local)
	assert.False(t, offline.EncryptionConfigured())
	require.NoError(t, offline.LoadCachedEncryption(context.Background()))
	assert.True(t, offline.EncryptionConfigured())
	require.NoError(t, offline.Unlock(context.Background(), []byte("correct-horse")))

	assert.Equal(t, wantMK, offline.MasterKeyForTest(), "offline unlock must recover the same master key")
}

func TestLoadCachedEncryption_Empty(t *testing.T) {
	uc := newUseCase(mocks.NewMockServerClient(t), mocks.NewMockTokenStore(t))
	err := uc.LoadCachedEncryption(context.Background())
	require.ErrorIs(t, err, authuc.ErrEncryptionNotSetup)
}

// --- error-ветки: ошибки сервера/хранилища должны пробрасываться ---

func TestRegister_ServerError(t *testing.T) {
	boom := errors.New("register rpc failed")

	server := mocks.NewMockServerClient(t)
	server.EXPECT().Register(mock.Anything, "alice", []byte("pw")).Return(contracts.Tokens{}, boom)

	// Save не должен вызываться — токенов нет.
	store := mocks.NewMockTokenStore(t)

	err := newUseCase(server, store).Register(context.Background(), "alice", []byte("pw"))
	assert.ErrorIs(t, err, boom)
}

func TestRegister_SaveError(t *testing.T) {
	boom := errors.New("keyring write failed")
	tokens := contracts.Tokens{AccessToken: "a", RefreshToken: "r"}

	server := mocks.NewMockServerClient(t)
	server.EXPECT().Register(mock.Anything, "alice", []byte("pw")).Return(tokens, nil)

	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Save(tokens).Return(boom)

	err := newUseCase(server, store).Register(context.Background(), "alice", []byte("pw"))
	assert.ErrorIs(t, err, boom)
}

func TestLogin_ServerError(t *testing.T) {
	boom := errors.New("login rpc failed")

	server := mocks.NewMockServerClient(t)
	server.EXPECT().Login(mock.Anything, "alice", []byte("pw")).Return(contracts.LoginResult{}, boom)

	store := mocks.NewMockTokenStore(t)

	err := newUseCase(server, store).Login(context.Background(), "alice", []byte("pw"))
	assert.ErrorIs(t, err, boom)
}

func TestLogin_SaveError(t *testing.T) {
	boom := errors.New("keyring write failed")
	res := contracts.LoginResult{Tokens: contracts.Tokens{AccessToken: "a", RefreshToken: "r"}}

	server := mocks.NewMockServerClient(t)
	server.EXPECT().Login(mock.Anything, "alice", []byte("pw")).Return(res, nil)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Save(res.Tokens).Return(boom)

	err := newUseCase(server, store).Login(context.Background(), "alice", []byte("pw"))
	assert.ErrorIs(t, err, boom)
}

func TestSetupEncryption_LoadError(t *testing.T) {
	boom := errors.New("no tokens")

	// SetupEncryption не должен вызываться, если access-токен не удалось загрузить.
	server := mocks.NewMockServerClient(t)

	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Load().Return(contracts.Tokens{}, boom)

	err := newUseCase(server, store).SetupEncryption(context.Background(), []byte("passphrase"))
	assert.ErrorIs(t, err, boom)
}

func TestSetupEncryption_ServerError(t *testing.T) {
	boom := errors.New("setup rpc failed")

	server := mocks.NewMockServerClient(t)
	server.EXPECT().SetupEncryption(mock.Anything, "access-1", mock.Anything, mock.Anything, mock.Anything).Return(boom)

	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Load().Return(contracts.Tokens{AccessToken: "access-1"}, nil)

	uc := newUseCase(server, store)
	err := uc.SetupEncryption(context.Background(), []byte("passphrase"))
	assert.ErrorIs(t, err, boom)
	assert.False(t, uc.MasterKeySet(), "master key must not be set into session when server call fails")
}

// TestUnlock_MalformedParams: битый JSON в enc_kdf_params (пришёл при Login) → ошибка,
// а не паника и не «успешный» вывод ключа.
func TestUnlock_MalformedParams(t *testing.T) {
	res := contracts.LoginResult{
		Tokens:       contracts.Tokens{AccessToken: "a", RefreshToken: "r"},
		EncKDFSalt:   bytes.Repeat([]byte{1}, 16),
		EncKDFParams: []byte("{not-json"),
		EncMasterKey: bytes.Repeat([]byte{2}, 60), // непустой, чтобы дойти до разбора params
	}

	server := mocks.NewMockServerClient(t)
	server.EXPECT().Login(mock.Anything, "alice", []byte("pw")).Return(res, nil)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	store := mocks.NewMockTokenStore(t)
	store.EXPECT().Save(res.Tokens).Return(nil)

	uc := newUseCase(server, store)
	require.NoError(t, uc.Login(context.Background(), "alice", []byte("pw")))

	err := uc.Unlock(context.Background(), []byte("passphrase"))
	require.Error(t, err)
	assert.NotErrorIs(t, err, authuc.ErrEmptyPassphrase)
	assert.NotErrorIs(t, err, authuc.ErrEncryptionNotSetup)
	assert.False(t, uc.MasterKeySet())
}
