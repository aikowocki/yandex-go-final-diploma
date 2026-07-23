package keyring_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/keyring"
)

// fakeTokenStore — простой in-memory contracts.TokenStore для тестов AutoRefreshStore.
type fakeTokenStore struct {
	tokens  contracts.Tokens
	loadErr error
	saved   []contracts.Tokens
}

func (f *fakeTokenStore) Save(t contracts.Tokens) error {
	f.saved = append(f.saved, t)
	f.tokens = t
	return nil
}

func (f *fakeTokenStore) Load() (contracts.Tokens, error) {
	if f.loadErr != nil {
		return contracts.Tokens{}, f.loadErr
	}
	return f.tokens, nil
}

func (f *fakeTokenStore) Clear() error {
	f.tokens = contracts.Tokens{}
	return nil
}

// fakeRefresher — мок Refresher (subset ServerClient) для тестов.
type fakeRefresher struct {
	result contracts.LoginResult
	err    error
	calls  int
}

func (f *fakeRefresher) RefreshToken(ctx context.Context, refreshToken string) (contracts.LoginResult, error) {
	f.calls++
	return f.result, f.err
}

// makeJWT строит JWT-подобный токен с заданным exp (только для теста parseJWTExp,
// без проверки подписи.
func makeJWT(t *testing.T, exp time.Time) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload, err := json.Marshal(map[string]int64{"exp": exp.Unix()})
	require.NoError(t, err)
	body := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + body + ".sig"
}

func TestAutoRefreshStore_Save_DelegatesToInner(t *testing.T) {
	inner := &fakeTokenStore{}
	s := keyring.NewAutoRefreshStore(inner, &fakeRefresher{})

	tokens := contracts.Tokens{AccessToken: "a", RefreshToken: "r"}
	require.NoError(t, s.Save(tokens))
	assert.Equal(t, tokens, inner.tokens)
}

func TestAutoRefreshStore_Clear_DelegatesToInner(t *testing.T) {
	inner := &fakeTokenStore{tokens: contracts.Tokens{AccessToken: "a"}}
	s := keyring.NewAutoRefreshStore(inner, &fakeRefresher{})

	require.NoError(t, s.Clear())
	assert.Equal(t, contracts.Tokens{}, inner.tokens)
}

func TestAutoRefreshStore_Load_LoadErrorPropagates(t *testing.T) {
	inner := &fakeTokenStore{loadErr: assert.AnError}
	s := keyring.NewAutoRefreshStore(inner, &fakeRefresher{})

	_, err := s.Load()
	require.ErrorIs(t, err, assert.AnError)
}

func TestAutoRefreshStore_Load_UnparsableTokenReturnsAsIs(t *testing.T) {
	inner := &fakeTokenStore{tokens: contracts.Tokens{AccessToken: "not-a-jwt", RefreshToken: "r"}}
	refresher := &fakeRefresher{}
	s := keyring.NewAutoRefreshStore(inner, refresher)

	got, err := s.Load()
	require.NoError(t, err)
	assert.Equal(t, "not-a-jwt", got.AccessToken)
	assert.Equal(t, 0, refresher.calls, "refresh не должен вызываться для нераспознанного токена")
}

func TestAutoRefreshStore_Load_FreshTokenSkipsRefresh(t *testing.T) {
	freshToken := makeJWT(t, time.Now().Add(time.Hour))
	inner := &fakeTokenStore{tokens: contracts.Tokens{AccessToken: freshToken, RefreshToken: "r"}}
	refresher := &fakeRefresher{}
	s := keyring.NewAutoRefreshStore(inner, refresher)

	got, err := s.Load()
	require.NoError(t, err)
	assert.Equal(t, freshToken, got.AccessToken)
	assert.Equal(t, 0, refresher.calls)
}

func TestAutoRefreshStore_Load_ExpiringSoonTriggersRefresh(t *testing.T) {
	expiringToken := makeJWT(t, time.Now().Add(30*time.Second))
	inner := &fakeTokenStore{tokens: contracts.Tokens{AccessToken: expiringToken, RefreshToken: "r"}}
	newAccessToken := makeJWT(t, time.Now().Add(time.Hour))
	refresher := &fakeRefresher{result: contracts.LoginResult{
		Tokens: contracts.Tokens{AccessToken: newAccessToken, RefreshToken: "new-r"},
	}}
	s := keyring.NewAutoRefreshStore(inner, refresher)

	got, err := s.Load()
	require.NoError(t, err)
	assert.Equal(t, newAccessToken, got.AccessToken)
	assert.Equal(t, 1, refresher.calls)
	require.Len(t, inner.saved, 1)
	assert.Equal(t, "new-r", inner.saved[0].RefreshToken)
}

func TestAutoRefreshStore_Load_RefreshErrorFallsBackToOldToken(t *testing.T) {
	expiredToken := makeJWT(t, time.Now().Add(-time.Minute))
	inner := &fakeTokenStore{tokens: contracts.Tokens{AccessToken: expiredToken, RefreshToken: "r"}}
	refresher := &fakeRefresher{err: assert.AnError}
	s := keyring.NewAutoRefreshStore(inner, refresher)

	got, err := s.Load()
	require.NoError(t, err, "ошибка refresh не должна возвращаться наружу (offline fallback)")
	assert.Equal(t, expiredToken, got.AccessToken)
	assert.Empty(t, inner.saved, "новый токен не должен сохраняться при ошибке refresh")
}

// Тесты порога проактивного обновления (refreshThreshold = 2 минуты) на
// синтетическом времени testing/synctest.
//
// exp у JWT хранится в целых Unix-секундах, поэтому сдвиги считаем в целых
// секундах, чтобы округление при кодировании токена не съедало границу.

func TestAutoRefreshStore_Load_ExactlyAtThresholdTriggersRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// time.Until(exp) == refreshThreshold: условие "> threshold" ложно,
		// значит должен сработать refresh (порог не строгий "чуть меньше").
		expiringToken := makeJWT(t, time.Now().Add(keyring.RefreshThreshold))
		inner := &fakeTokenStore{tokens: contracts.Tokens{AccessToken: expiringToken, RefreshToken: "r"}}
		newAccessToken := makeJWT(t, time.Now().Add(time.Hour))
		refresher := &fakeRefresher{result: contracts.LoginResult{
			Tokens: contracts.Tokens{AccessToken: newAccessToken, RefreshToken: "new-r"},
		}}
		s := keyring.NewAutoRefreshStore(inner, refresher)

		got, err := s.Load()
		require.NoError(t, err)
		assert.Equal(t, newAccessToken, got.AccessToken)
		assert.Equal(t, 1, refresher.calls, "на границе порога refresh должен сработать")
	})
}

func TestAutoRefreshStore_Load_JustAboveThresholdSkipsRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// time.Until(exp) == threshold + 1s > threshold: refresh не нужен.
		freshToken := makeJWT(t, time.Now().Add(keyring.RefreshThreshold+time.Second))
		inner := &fakeTokenStore{tokens: contracts.Tokens{AccessToken: freshToken, RefreshToken: "r"}}
		refresher := &fakeRefresher{}
		s := keyring.NewAutoRefreshStore(inner, refresher)

		got, err := s.Load()
		require.NoError(t, err)
		assert.Equal(t, freshToken, got.AccessToken)
		assert.Equal(t, 0, refresher.calls, "чуть выше порога refresh не должен вызываться")
	})
}

func TestAutoRefreshStore_Load_JustBelowThresholdTriggersRefresh(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// time.Until(exp) == threshold - 1s <= threshold: refresh нужен.
		expiringToken := makeJWT(t, time.Now().Add(keyring.RefreshThreshold-time.Second))
		inner := &fakeTokenStore{tokens: contracts.Tokens{AccessToken: expiringToken, RefreshToken: "r"}}
		newAccessToken := makeJWT(t, time.Now().Add(time.Hour))
		refresher := &fakeRefresher{result: contracts.LoginResult{
			Tokens: contracts.Tokens{AccessToken: newAccessToken, RefreshToken: "new-r"},
		}}
		s := keyring.NewAutoRefreshStore(inner, refresher)

		got, err := s.Load()
		require.NoError(t, err)
		assert.Equal(t, newAccessToken, got.AccessToken)
		assert.Equal(t, 1, refresher.calls, "чуть ниже порога refresh должен сработать")
	})
}
