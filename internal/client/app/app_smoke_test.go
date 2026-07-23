package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clientapp "github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
)

// TestClientApp_SmokeBoot — смок-тест клиентского DI-контейнера: собирает весь
// client/app.Container (grpc-клиент, localstore, keyring->файл fallback, auth/vault/secret/
// sync usecase, i18n) без реального сервера — grpc.NewClient не дозванивается сразу
// (lazy-connect), поэтому сборка контейнера не требует сети. Проверяет, что весь DI-стек
// поднимается без ошибок и все поля Container заполнены рабочими объектами.
func TestClientApp_SmokeBoot(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.ClientConfig{
		ServerAddr: "127.0.0.1:9",
		DataDir:    dataDir,
		NoPersist:  false, // разрешаем keyring->файл fallback, чтобы тест не зависел от OS keychain
		LogLevel:   "error",
		Lang:       "en",
	}

	container, err := clientapp.New(cfg)
	require.NoError(t, err, "клиентский DI-стек (grpc, localstore, keyring, usecase, i18n) должен собраться без ошибок")
	t.Cleanup(func() { _ = container.Close() })

	assert.NotNil(t, container.GRPC)
	assert.NotNil(t, container.Local)
	assert.NotNil(t, container.Session)
	assert.NotNil(t, container.Auth)
	assert.NotNil(t, container.Vault)
	assert.NotNil(t, container.Secret)
	assert.NotNil(t, container.Sync)
	assert.NotNil(t, container.Localizer)
	assert.Equal(t, "en", container.Localizer.Lang())
}

// TestClientApp_SmokeBoot_NoPersist проверяет тот же сценарий сборки в режиме --no-persist
// (in-memory localstore, без файлового fallback для токенов).
func TestClientApp_SmokeBoot_NoPersist(t *testing.T) {
	cfg := &config.ClientConfig{
		ServerAddr: "127.0.0.1:9",
		DataDir:    t.TempDir(),
		NoPersist:  true,
		LogLevel:   "error",
		Lang:       "ru",
	}

	container, err := clientapp.New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Close() })

	assert.Equal(t, "ru", container.Localizer.Lang())
}
