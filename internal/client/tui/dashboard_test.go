package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/localstore"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

func newDashboardTestModel(t *testing.T, server contracts.ServerClient) dashboardModel {
	t.Helper()
	c := newTestContainer(t, server)
	return newDashboardModel(context.Background(), c)
}

// newDashboardTestModelStore — вариант newDashboardTestModel с явно переданными local/session,
// для тестов, которым нужно заранее наполнить localstore (outbox-конфликты и т.п.).
func newDashboardTestModelStore(t *testing.T, server contracts.ServerClient, sess *session.Session, local *localstore.Store) dashboardModel {
	t.Helper()
	c := newTestContainerWith(t, server, local, sess)
	return newDashboardModel(context.Background(), c)
}

func newMemStoreTUI(t *testing.T) *localstore.Store {
	t.Helper()
	ls, err := localstore.Open("", false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ls.Close() })
	return ls
}

func openVaultSessionTUI(t *testing.T, vaultID string) (*session.Session, []byte) {
	t.Helper()
	sess := session.New()
	vk, err := crypto.GenerateKey()
	require.NoError(t, err)
	sess.OpenVault(vaultID, vk)
	return sess, vk
}

// secretAADTUI воспроизводит AAD-контекст secret.secretAAD (не экспортирован из продакшн-кода
// пакета secret для внешних пакетов — export_test.go виден только внутри самого пакета secret).
func secretAADTUI(vaultID, secretID string, version int64, tier string) []byte {
	return []byte(fmt.Sprintf("gophkeeper:secret:v1|vault=%s|secret=%s|ver=%d|tier=%s", vaultID, secretID, version, tier))
}

func mustEncryptRowTUI(t *testing.T, vaultKey []byte, vaultID, secretID string, version int64, title string) []byte {
	t.Helper()
	c := cryptoimpl.Crypto{}
	blob, err := c.EncryptStruct(vaultKey, secretAADTUI(vaultID, secretID, version, "row"),
		secretcontent.LoginPasswordRow{V: 1, Title: title})
	require.NoError(t, err)
	return blob
}

// conflictErrorTUI собирает *grpcclient.ConflictError с серверной версией секрета, зашифрованной
// под тем же vaultKey (только Row-тир — достаточно для проверки навигации на screenConflict).
func conflictErrorTUI(vaultKey []byte, vaultID, secretID string, version int64) *grpcclient.ConflictError {
	c := cryptoimpl.Crypto{}
	encRow, err := c.EncryptStruct(vaultKey, secretAADTUI(vaultID, secretID, version, "row"),
		secretcontent.LoginPasswordRow{V: 1, Title: "ServerTitle"})
	if err != nil {
		panic(err)
	}
	return &grpcclient.ConflictError{Server: contracts.ServerSecret{ID: secretID, Type: 1, Version: version, EncRow: encRow}}
}

func enqueueConflictUpdateTUI(t *testing.T, local *localstore.Store, secretID, vaultID string, baseVersion int64, encRow []byte) {
	t.Helper()
	body, err := json.Marshal(contracts.OutboxSecretUpdate{
		SecretID: secretID, VaultID: vaultID, BaseVersion: baseVersion, Type: 1, EncRow: encRow,
	})
	require.NoError(t, err)
	_, err = local.EnqueueOutbox(context.Background(), contracts.OutboxEntry{
		Op: contracts.OutboxOpUpdate, Entity: "secret", EntityID: secretID, BaseVersion: baseVersion,
		Payload: body, Status: contracts.OutboxStatusConflict,
	})
	require.NoError(t, err)
}

func TestDashboardModel_CurrentVault_Empty(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	_, ok := m.currentVault()
	assert.False(t, ok)
	assert.Equal(t, "", m.currentVaultID())
	assert.Equal(t, "", m.currentVaultName())
}

func TestDashboardModel_CurrentVault_WithVaults(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.vaults = []vaultuc.DecryptedVault{{ID: "v1", Name: "Personal"}, {ID: "v2", Name: "Work"}}
	m.vaultCursor = 1

	v, ok := m.currentVault()
	require.True(t, ok)
	assert.Equal(t, "v2", v.ID)
	assert.Equal(t, "Work", m.currentVaultName())
}

func TestDashboardModel_SelectVaultByID(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.vaults = []vaultuc.DecryptedVault{{ID: "v1"}, {ID: "v2"}, {ID: "v3"}}

	m = m.selectVaultByID("v3")
	assert.Equal(t, 2, m.vaultCursor)

	m = m.selectVaultByID("") // no-op
	assert.Equal(t, 2, m.vaultCursor)

	m = m.selectVaultByID("unknown") // no-op, not found
	assert.Equal(t, 2, m.vaultCursor)
}

func TestDashboardModel_CurrentTypeTab_OutOfRangeFallsBackToFirst(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.typeCursor = 999
	tab := m.currentTypeTab()
	assert.Equal(t, 0, int(tab.secretType))
}

func TestDashboardModel_VaultsLoadedMsg_FirstLoadShowsSyncScope(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable

	m, _ = m.update(vaultsLoadedMsg{vaults: []vaultuc.DecryptedVault{{ID: "v1", Name: "Personal"}}})
	assert.Equal(t, focusSyncScope, m.focus)
	assert.True(t, m.initialLoadDone)
}

func TestDashboardModel_VaultsErrMsg(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m, _ = m.update(vaultsErrMsg{err: assert.AnError})
	assert.ErrorIs(t, m.vaultsErr, assert.AnError)
}

func TestDashboardModel_VaultCreatedMsg(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusVaultCreate

	m, cmd := m.update(vaultCreatedMsg{id: "v1"})
	assert.Equal(t, focusTable, m.focus)
	assert.NotNil(t, cmd)
}

func TestDashboardModel_SyncDoneMsg(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.syncing = true

	m, cmd := m.update(syncDoneMsg{})
	assert.False(t, m.syncing)
	assert.NotNil(t, cmd)
}

func TestDashboardModel_SyncErrMsg(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.syncing = true

	m, _ = m.update(syncErrMsg{err: assert.AnError})
	assert.False(t, m.syncing)
	assert.ErrorIs(t, m.err, assert.AnError)
}

func TestDashboardModel_SyncScopeConfirmedMsg(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CheckFreshness(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	m := newDashboardTestModel(t, server)
	m.focus = focusSyncScope

	m, cmd := m.update(syncScopeConfirmedMsg{})
	assert.Equal(t, focusTable, m.focus)
	assert.True(t, m.backgroundSyncing)
	require.NotNil(t, cmd)
	// launchBackgroundSync запускает горутину и cmd читает первое сообщение из канала —
	// выполняем его, чтобы дождаться завершения горутины до конца теста (мок не переживёт
	// завершение теста иначе).
	cmd()
}

func TestDashboardModel_BackgroundSyncTickMsg_AlreadySyncing(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.backgroundSyncing = true

	m, cmd := m.update(backgroundSyncTickMsg{})
	assert.True(t, m.backgroundSyncing)
	assert.NotNil(t, cmd)
}

func TestDashboardModel_BackgroundSyncTickMsg_StartsSync(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CheckFreshness(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	m := newDashboardTestModel(t, server)
	m.backgroundSyncing = false

	m, cmd := m.update(backgroundSyncTickMsg{})
	assert.True(t, m.backgroundSyncing)
	require.NotNil(t, cmd)
	cmd() // дождаться завершения запущенной внутри горутины
}

func TestDashboardModel_BackgroundSyncDoneMsg_Success(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.backgroundSyncing = true

	m, cmd := m.update(backgroundSyncDoneMsg{})
	assert.False(t, m.backgroundSyncing)
	assert.False(t, m.lastSyncOK.IsZero())
	assert.NotNil(t, cmd)
}

func TestDashboardModel_BackgroundSyncDoneMsg_Error(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.backgroundSyncing = true

	m, cmd := m.update(backgroundSyncDoneMsg{err: assert.AnError})
	assert.False(t, m.backgroundSyncing)
	assert.ErrorIs(t, m.lastBackgroundSyncErr, assert.AnError)
	assert.NotNil(t, cmd)
}

func TestDashboardModel_SyncProgressWithNext(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	ch := make(chan struct{})
	close(ch)
	_ = ch

	progressCh := make(chan interface{})
	_ = progressCh

	m, cmd := m.update(syncProgressWithNext{label: "syncing rows"})
	assert.Equal(t, "syncing rows", m.syncProgressLabel)
	assert.NotNil(t, cmd)
}

func TestDashboardModel_SettingsSyncToggledMsg(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	_, cmd := m.update(settingsSyncToggledMsg{})
	assert.NotNil(t, cmd)
}

// --- handleKey ---

func TestDashboardModel_HandleKey_TabSwitchesType(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable
	m.vaults = []vaultuc.DecryptedVault{{ID: "v1"}}

	m, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 1, m.typeCursor)
	assert.NotNil(t, cmd)
}

func TestDashboardModel_HandleKey_DigitSelectsVault(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable
	m.vaults = []vaultuc.DecryptedVault{{ID: "v1"}, {ID: "v2"}}

	m, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	assert.Equal(t, 1, m.vaultCursor)
	assert.NotNil(t, cmd)
}

func TestDashboardModel_HandleKey_SlashActivatesSearch(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.Equal(t, focusCommand, m.focus)
	assert.False(t, m.commandLine.isCommand(), "шорткат '/' открывает поиск пустым, а не команду")
}

func TestDashboardModel_HandleKey_TypingSlash_EntersCommandMode(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.commandLine, _ = m.commandLine.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, m.commandLine.isCommand())
}

func TestDashboardModel_HandleKey_S_OpensSettings(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	assert.Equal(t, focusSettings, m.focus)
}

func TestDashboardModel_HandleKey_U_OpensUserMenu(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, focusUser, m.focus)
}

func TestDashboardModel_HandleKey_G_OpensLogs(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	assert.Equal(t, focusLogs, m.focus)
}

func TestDashboardModel_HandleKey_L_SoftLocks(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable
	m.container.Session.SetMasterKey(bytes.Repeat([]byte{42}, 32))

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	require.NotNil(t, cmd)
	assert.False(t, m.container.Session.Unlocked())
}

func TestDashboardModel_HandleKey_N_NoVaultsOpensCreate(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.Equal(t, focusVaultCreate, m.focus)
}

func TestDashboardModel_HandleKey_N_WithVaultOpensForm(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable
	m.vaults = []vaultuc.DecryptedVault{{ID: "v1", Name: "Personal"}}

	m, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenForm, switchMsg.screen)
}

func TestDashboardModel_HandleKey_CtrlN_AlwaysOpensCreate(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable
	m.vaults = []vaultuc.DecryptedVault{{ID: "v1"}}

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlN})
	assert.Equal(t, focusVaultCreate, m.focus)
}

func TestDashboardModel_HandleKey_X_OpensConflictWhenPresent(t *testing.T) {
	sess, vaultKey := openVaultSessionTUI(t, "v1")
	local := newMemStoreTUI(t)
	server := mocks.NewMockServerClient(t)
	enqueueConflictUpdateTUI(t, local, "s1", "v1", 5, mustEncryptRowTUI(t, vaultKey, "v1", "s1", 6, "mine"))
	server.EXPECT().
		UpdateSecret(mock.Anything, mock.Anything, "s1", int64(5), mock.Anything, mock.Anything, mock.Anything).
		Return(int64(0), conflictErrorTUI(vaultKey, "v1", "s1", 7))

	m := newDashboardTestModelStore(t, server, sess, local)
	m.focus = focusTable

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok, "expected switchScreenMsg, got %T", msg)
	assert.Equal(t, screenConflict, switchMsg.screen)
	require.NotNil(t, switchMsg.conflict)
	assert.Equal(t, "s1", switchMsg.conflict.SecretID)
}

func TestDashboardModel_HandleKey_X_NoConflictsShowsToast(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(toastMsg)
	assert.True(t, ok, "expected toastMsg, got %T", msg)
}

func TestDashboardModel_RunCommand_Conflicts(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	_, cmd := m.runCommand("conflicts")
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(toastMsg)
	assert.True(t, ok, "no conflicts present — expect toast")
}

func TestDashboardModel_OutboxConflictCountMsg_UpdatesBadge(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m, _ = m.update(outboxConflictCountMsg{count: 3})
	assert.Equal(t, 3, m.outboxConflictCount)
	assert.Contains(t, m.renderTopBarButtons(), "3")
}

func TestDashboardModel_HandleKey_NoVaults_NoOp(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusTable
	m.vaults = nil

	// 'z' не назначен ни на один шорткат — должен быть тихим no-op при отсутствии vault'ов.
	m2, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	assert.Nil(t, cmd)
	assert.Equal(t, m.vaults, m2.vaults)
}

// --- popup key handlers ---

func TestDashboardModel_HandleSettingsKey_Esc(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusSettings
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, focusTable, m.focus)
}

func TestDashboardModel_HandleUserMenuKey_Esc(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusUser
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, focusTable, m.focus)
}

func TestDashboardModel_HandleUserMenuKey_EnterLogsOut(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusUser
	m.container.Session.SetMasterKey(bytes.Repeat([]byte{42}, 32))

	m, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenLogin, switchMsg.screen)
	assert.False(t, m.container.Session.Unlocked())
}

func TestDashboardModel_HandleSyncScopeKey_Confirm(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusSyncScope
	m.syncScope = newSyncScopePopup([]vaultuc.DecryptedVault{{ID: "v1"}})

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)
}

func TestDashboardModel_HandleVaultCreateKey_Esc(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusVaultCreate
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, focusTable, m.focus)
}

func TestDashboardModel_HandleVaultCreateKey_EmptyNameNoOp(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusVaultCreate

	m, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Equal(t, focusVaultCreate, m.focus)
}

func TestDashboardModel_HandleVaultCreateKey_EnterCreates(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateVault(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("v1", nil)

	m := newDashboardTestModel(t, server)
	m.focus = focusVaultCreate
	m.container.Session.SetMasterKey(bytes.Repeat([]byte{42}, 32))
	m.vaultCreate.input.SetValue("My Vault")

	m, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, focusTable, m.focus)
	require.NotNil(t, cmd)
	msg := cmd()
	if errMsg, ok := msg.(vaultsErrMsg); ok {
		t.Fatalf("unexpected vaultsErrMsg: %v", errMsg.err)
	}
	created, ok := msg.(vaultCreatedMsg)
	require.True(t, ok, "expected vaultCreatedMsg, got %T", msg)
	assert.Equal(t, "v1", created.id)
}

func TestDashboardModel_HandleLogsKey_Esc(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusLogs
	m.logs = newLogsPopup(t.TempDir())
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, focusTable, m.focus)
}

// --- command line ---

func TestDashboardModel_HandleCommandKey_Esc(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusCommand
	m.commandLine = m.commandLine.activate()

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, focusTable, m.focus)
	assert.Empty(t, m.commandLine.value())
}

func TestDashboardModel_RunCommand_Sync(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m, cmd := m.runCommand("sync")
	assert.True(t, m.syncing)
	assert.NotNil(t, cmd)
}

func TestDashboardModel_RunCommand_Quit(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	_, cmd := m.runCommand("quit")
	require.NotNil(t, cmd)
}

func TestDashboardModel_RunCommand_Lock(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	_, cmd := m.runCommand("lock")
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenLock, switchMsg.screen)
}

func TestDashboardModel_RunCommand_Vault(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m, _ = m.runCommand("vault")
	assert.Equal(t, focusVaultCreate, m.focus)
}

func TestDashboardModel_RunCommand_Logs(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m, _ = m.runCommand("logs")
	assert.Equal(t, focusLogs, m.focus)
}

func TestDashboardModel_RunCommand_Unknown_NoOp(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m2, cmd := m.runCommand("bogus")
	assert.Nil(t, cmd)
	assert.Equal(t, m.focus, m2.focus)
}

// --- loadVaults / doCreateVault / doSync ---

func TestDashboardModel_LoadVaults_LocalCacheEmpty(t *testing.T) {
	// ListLocal успешно возвращает пустой список (не ошибку) для пустого кеша — server.List
	// не вызывается в этом случае.
	server := mocks.NewMockServerClient(t)
	m := newDashboardTestModel(t, server)
	m.container.Session.SetMasterKey(bytes.Repeat([]byte{42}, 32))

	cmd := m.loadVaults()
	require.NotNil(t, cmd)
	msg := cmd()
	loaded, ok := msg.(vaultsLoadedMsg)
	require.True(t, ok)
	assert.Empty(t, loaded.vaults)
}

func TestDashboardModel_LoadVaultsFromServer_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, nil)

	m := newDashboardTestModel(t, server)
	m.container.Session.SetMasterKey(bytes.Repeat([]byte{42}, 32))

	cmd := m.loadVaultsFromServer()
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(vaultsLoadedMsg)
	assert.True(t, ok)
}

func TestDashboardModel_DoCreateVault_VaultLocked(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	cmd := m.doCreateVault("My Vault")
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(vaultsErrMsg)
	assert.True(t, ok)
}

func TestDashboardModel_DoSync_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CheckFreshness(mock.Anything, mock.Anything).Return(nil, nil)

	m := newDashboardTestModel(t, server)
	cmd := m.doSync()
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(syncDoneMsg)
	assert.True(t, ok)
}

// --- render / view ---

func TestDashboardModel_View_NoVaults(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	out := m.view(80, 24)
	assert.NotEmpty(t, out)
}

func TestDashboardModel_View_WithVaultsErr(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.vaultsErr = assert.AnError
	out := m.view(80, 24)
	assert.Contains(t, out, assert.AnError.Error())
}

func TestDashboardModel_View_Settings(t *testing.T) {
	m := newDashboardTestModel(t, mocks.NewMockServerClient(t))
	m.focus = focusSettings
	m.settings = newSettingsPopup(m.container)
	out := m.view(80, 24)
	assert.NotEmpty(t, out)
}

func TestIncDecWrap(t *testing.T) {
	assert.Equal(t, 1, incWrap(0, 3))
	assert.Equal(t, 0, incWrap(2, 3))
	assert.Equal(t, 0, incWrap(0, 0))

	assert.Equal(t, 2, decWrap(0, 3))
	assert.Equal(t, 1, decWrap(2, 3))
	assert.Equal(t, 0, decWrap(0, 0))
}

func TestLipglossWidth(t *testing.T) {
	assert.Equal(t, 5, lipglossWidth("hello"))
}

func TestIsAuthError(t *testing.T) {
	assert.False(t, isAuthError(nil))
	assert.False(t, isAuthError(assert.AnError))
	assert.True(t, isAuthError(errors.New("invalid login or credential")))
	assert.True(t, isAuthError(errors.New("token expired")))
}
