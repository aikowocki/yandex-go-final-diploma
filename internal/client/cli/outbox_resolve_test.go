package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// secretAADCLI воспроизводит AAD-контекст secret.secretAAD (не экспортирован из продакшн-кода
// пакета secret для внешних пакетов — export_test.go виден только внутри самого пакета secret).
func secretAADCLI(vaultID, secretID string, version int64, tier string) []byte {
	return []byte(fmt.Sprintf("gophkeeper:secret:v1|vault=%s|secret=%s|ver=%d|tier=%s", vaultID, secretID, version, tier))
}

// enqueueConflictUpdateCLI кладёт в outbox update-запись со статусом conflict — как это делает
// ReplayOutbox после гонки версий.
func enqueueConflictUpdateCLI(t *testing.T, env *cliTestEnv, secretID, vaultID string, baseVersion int64, encRow []byte) int64 {
	t.Helper()
	body, err := json.Marshal(contracts.OutboxSecretUpdate{
		SecretID: secretID, VaultID: vaultID, BaseVersion: baseVersion, Type: 1, EncRow: encRow,
	})
	require.NoError(t, err)
	id, err := env.Local.EnqueueOutbox(context.Background(), contracts.OutboxEntry{
		Op: contracts.OutboxOpUpdate, Entity: "secret", EntityID: secretID, BaseVersion: baseVersion,
		Payload: body, Status: contracts.OutboxStatusConflict,
	})
	require.NoError(t, err)
	return id
}

func TestOutboxResolveCmd_Run_ResolvesInFavorOfMine(t *testing.T) {
	vaultKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	c := cryptoimpl.Crypto{}
	mineRow, err := c.EncryptStruct(vaultKey, secretAADCLI("v1", "s1", 6, "row"),
		secretcontent.LoginPasswordRow{V: 1, Title: "mine-title"})
	require.NoError(t, err)

	server := mocks.NewMockServerClient(t)
	env := newCLITestEnv(t, server)
	env.Session.SetMasterKey([]byte("dummy-master-key-not-checked!!!"))
	env.Session.OpenVault("v1", vaultKey)

	entryID := enqueueConflictUpdateCLI(t, env, "s1", "v1", 5, mineRow)

	srvIdx, err := c.EncryptStruct(vaultKey, secretAADCLI("v1", "s1", 7, "index"),
		secretcontent.LoginPasswordIndex{V: 1})
	require.NoError(t, err)
	srvPayload, err := c.EncryptStruct(vaultKey, secretAADCLI("v1", "s1", 7, "payload"),
		secretcontent.LoginPasswordPayload{V: 1})
	require.NoError(t, err)
	srvRow, err := c.EncryptStruct(vaultKey, secretAADCLI("v1", "s1", 7, "row"),
		secretcontent.LoginPasswordRow{V: 1, Title: "ServerTitle"})
	require.NoError(t, err)

	// Первый UpdateSecret (внутри ConflictFromOutbox, показывает карточку) — конфликт.
	server.EXPECT().
		UpdateSecret(mock.Anything, mock.Anything, "s1", int64(5), mineRow, mock.Anything, mock.Anything).
		Return(int64(0), &grpcclient.ConflictError{Server: contracts.ServerSecret{
			ID: "s1", Type: 1, Version: 7, EncRow: srvRow, EncIndex: srvIdx, EncPayload: srvPayload,
		}})
	// Выбор "mine" (choose 'm') повторяет update с baseVersion=7 — на этот раз успех.
	server.EXPECT().
		UpdateSecret(mock.Anything, mock.Anything, "s1", int64(7), mock.Anything, mock.Anything, mock.Anything).
		Return(int64(8), nil)

	scriptLines(t, "m") // conflict_choose prompt

	cmd := &OutboxResolveCmd{ID: entryID}
	require.NoError(t, cmd.Run(env.Auth, env.Secret, env.Localizer))
}

func TestOutboxResolveCmd_Run_UnknownEntry(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	env := newCLITestEnv(t, server)
	env.Session.SetMasterKey([]byte("dummy-master-key-not-checked!!!"))

	cmd := &OutboxResolveCmd{ID: 999}
	require.Error(t, cmd.Run(env.Auth, env.Secret, env.Localizer))
}
