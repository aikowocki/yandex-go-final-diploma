package providers_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app/providers"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
)

func TestNewConfig_LoadsDefaults(t *testing.T) {
	// LoadClientConfig парсит os.Args; в тестовом бинарнике там флаги go test
	// (-test.run и т.п.), которые kong не распознаёт — подменяем на пустой набор.
	origArgs := os.Args
	os.Args = []string{origArgs[0]}
	t.Cleanup(func() { os.Args = origArgs })

	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := providers.NewConfig()
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.ServerAddr)
}

func TestNewGRPCClient_LazyConnectDoesNotError(t *testing.T) {
	// grpc.NewClient не дозванивается сразу (lazy-connect), поэтому создание клиента
	// с несуществующим адресом всё равно должно успешно вернуть объект без ошибки.
	client, err := providers.NewGRPCClient(&config.ClientConfig{ServerAddr: "127.0.0.1:9"})
	require.NoError(t, err)
	require.NotNil(t, client)
	t.Cleanup(func() { _ = client.Close() })
}
