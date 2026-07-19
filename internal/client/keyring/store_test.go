package keyring_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gokeyring "github.com/zalando/go-keyring"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/keyring"
)

func sampleTokens() contracts.Tokens {
	return contracts.Tokens{AccessToken: "access-1", RefreshToken: "refresh-1"}
}

func TestStore_KeyringRoundTrip(t *testing.T) {
	gokeyring.MockInit()

	dir := t.TempDir()
	s := keyring.New(dir, true)

	require.NoError(t, s.Save(sampleTokens()))

	// При работающем keyring файловый fallback не используется.
	_, statErr := os.Stat(filepath.Join(dir, "token.json"))
	assert.ErrorIs(t, statErr, os.ErrNotExist, "token file must not be created when keyring works")

	got, err := s.Load()
	require.NoError(t, err)
	assert.Equal(t, sampleTokens(), got)

	require.NoError(t, s.Clear())

	_, err = s.Load()
	require.ErrorIs(t, err, keyring.ErrNoToken)
}

func TestStore_NoToken(t *testing.T) {
	gokeyring.MockInit()

	s := keyring.New(t.TempDir(), true)

	_, err := s.Load()
	require.ErrorIs(t, err, keyring.ErrNoToken)
}

func TestStore_FileFallback(t *testing.T) {
	// Заставляем OS keyring падать — Save/Load должны использовать файловый fallback.
	gokeyring.MockInitWithError(errors.New("keyring unavailable"))

	dir := t.TempDir()
	s := keyring.New(dir, true)

	require.NoError(t, s.Save(sampleTokens()))

	info, err := os.Stat(filepath.Join(dir, "token.json"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "token file must be 0600")

	got, err := s.Load()
	require.NoError(t, err)
	assert.Equal(t, sampleTokens(), got)
}

// TestStore_NoPersist_UsesMemoryOnly — persist=false (--no-persist) должен работать полностью
// в памяти процесса и НЕ трогать системный OS keyring вообще, даже если он исправен.
func TestStore_NoPersist_UsesMemoryOnly(t *testing.T) {
	gokeyring.MockInit() // keyring исправен — но не должен использоваться при persist=false

	dir := t.TempDir()
	s := keyring.New(dir, false)

	require.NoError(t, s.Save(sampleTokens()))

	got, err := s.Load()
	require.NoError(t, err)
	assert.Equal(t, sampleTokens(), got)

	// Ни файл, ни системный keyring не должны быть тронуты.
	_, statErr := os.Stat(filepath.Join(dir, "token.json"))
	assert.ErrorIs(t, statErr, os.ErrNotExist, "no-persist must not write a token file")

	independentStore := keyring.New(dir, true) // persist=true — читает из keyring/файла
	_, err = independentStore.Load()
	assert.ErrorIs(t, err, keyring.ErrNoToken, "no-persist tokens must not leak into the OS keyring")
}

// TestStore_NoPersist_EvenWhenKeyringFails — no-persist не деградирует в файловый fallback
// при недоступном keyring: он просто не использует ни то, ни другое, независимо от их
// состояния — только память.
func TestStore_NoPersist_EvenWhenKeyringFails(t *testing.T) {
	gokeyring.MockInitWithError(errors.New("keyring unavailable"))

	s := keyring.New(t.TempDir(), false)

	require.NoError(t, s.Save(sampleTokens()), "no-persist Save must succeed purely in-memory regardless of OS keyring state")

	got, err := s.Load()
	require.NoError(t, err)
	assert.Equal(t, sampleTokens(), got)
}

// TestStore_NoPersist_Clear сбрасывает только in-memory состояние.
func TestStore_NoPersist_Clear(t *testing.T) {
	gokeyring.MockInit()

	s := keyring.New(t.TempDir(), false)
	require.NoError(t, s.Save(sampleTokens()))
	require.NoError(t, s.Clear())

	_, err := s.Load()
	require.ErrorIs(t, err, keyring.ErrNoToken)
}
