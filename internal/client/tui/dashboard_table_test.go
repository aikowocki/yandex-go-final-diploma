package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/cryptoimpl"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func newDashboardTableTestModel(t *testing.T, server contracts.ServerClient) (dashboardTableModel, []byte) {
	t.Helper()
	c := newTestContainer(t, server)
	vaultKey, err := cryptoimpl.Crypto{}.GenerateVaultKey()
	require.NoError(t, err)
	c.Session.OpenVault("v1", vaultKey)
	m := newDashboardTableModel(context.Background(), c)
	return m, vaultKey
}

// upsertLoginSecret создаёт login/password секрет через реальный usecase (CreateLoginPassword),
// который сам корректно шифрует все тиры под правильным AAD — избегаем дублирования приватной
// схемы AAD пакета secret в тестах другого пакета.
func upsertLoginSecret(t *testing.T, m dashboardTableModel, title, username, uri, password string) string {
	t.Helper()
	id, err := m.container.Secret.CreateLoginPassword(context.Background(), "v1", secret.CreateLoginPasswordInput{
		Title: title, Username: username, URI: uri, Password: password,
	})
	require.NoError(t, err)
	return id
}

func TestDashboardTableModel_SetVaultAndTypeEmptyClearsRows(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.rows = []secret.SummaryRow{{ID: "s1"}}
	m, cmd := m.setVaultAndType("", 0)
	assert.Nil(t, cmd)
	assert.Empty(t, m.rows)
	assert.False(t, m.loading)
}

func TestDashboardTableModel_SetVaultAndTypeTriggersReload(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m, cmd := m.setVaultAndType("v1", 0)
	require.NotNil(t, cmd)
	assert.True(t, m.loading)
}

func TestDashboardTableModel_Reload_LoadsRows(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, "v1", int32(1), mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m, _ := newDashboardTableTestModel(t, server)
	m.vaultID = "v1"
	upsertLoginSecret(t, m, "GitHub", "alice", "github.com", "hunter2")

	cmd := m.reload()
	require.NotNil(t, cmd)
	msg := cmd()
	loaded, ok := msg.(rowsLoadedMsg)
	require.True(t, ok)
	require.Len(t, loaded.rows, 1)
	assert.Equal(t, "GitHub", loaded.rows[0].Title)
}

func TestDashboardTableModel_RowsLoadedMsg_AppliesFilter(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	rows := []secret.SummaryRow{{ID: "s1", Title: "GitHub"}, {ID: "s2", Title: "AWS"}}

	m, cmd := m.update(rowsLoadedMsg{rows: rows})
	assert.False(t, m.loading)
	assert.Len(t, m.rows, 2)
	assert.NotNil(t, cmd) // scheduleReveal
}

func TestDashboardTableModel_ApplyLocalFilter_ByTitle(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.allRows = []secret.SummaryRow{
		{ID: "s1", Title: "GitHub"},
		{ID: "s2", Title: "AWS Console"},
	}
	m.searchTerm = "git"
	m = m.applyLocalFilter()
	require.Len(t, m.rows, 1)
	assert.Equal(t, "s1", m.rows[0].ID)
}

func TestDashboardTableModel_ApplyLocalFilter_ByTag(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.allRows = []secret.SummaryRow{
		{ID: "s1", Title: "GitHub", Tags: []string{"work"}},
		{ID: "s2", Title: "AWS", Tags: []string{"personal"}},
	}
	m.searchTerm = "personal"
	m = m.applyLocalFilter()
	require.Len(t, m.rows, 1)
	assert.Equal(t, "s2", m.rows[0].ID)
}

func TestDashboardTableModel_ClearSearch(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.allRows = []secret.SummaryRow{{ID: "s1", Title: "GitHub"}}
	m.searchTerm = "nonexistent"
	m = m.applyLocalFilter()
	require.Empty(t, m.rows)

	m, _ = m.clearSearch()
	assert.Empty(t, m.searchTerm)
	assert.Len(t, m.rows, 1)
}

func TestDashboardTableModel_CursorNavigation(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.rows = []secret.SummaryRow{{ID: "s1"}, {ID: "s2"}, {ID: "s3"}}

	m, cmd := m.update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.cursor)
	assert.NotNil(t, cmd)

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m.cursor)

	// Не выходит за границы.
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m.cursor)

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, m.cursor)
}

func TestDashboardTableModel_RowsErrMsg(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.loading = true

	m, _ = m.update(rowsErrMsg{err: assert.AnError})
	assert.False(t, m.loading)
	assert.ErrorIs(t, m.err, assert.AnError)
}

func TestDashboardTableModel_PayloadRevealedMsg(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.rows = []secret.SummaryRow{{ID: "s1"}}
	m.cursor = 0

	m, _ = m.update(payloadRevealedMsg{secretID: "s1", password: "hunter2"})
	assert.Equal(t, "s1", m.revealedID)
	assert.Equal(t, "hunter2", m.revealedSecret)
}

func TestDashboardTableModel_CardRevealedMsg(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.rows = []secret.SummaryRow{{ID: "s1"}}
	m.cursor = 0

	m, _ = m.update(cardRevealedMsg{secretID: "s1", cvv: "123", pin: "4567", pan: "4111111111111111"})
	assert.True(t, m.cardReveal)
	assert.Equal(t, "123", m.cardCVV)
	assert.Equal(t, "4111111111111111", m.cardPAN)
}

func TestDashboardTableModel_CopyCurrent_Empty(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	cmd := m.copyCurrent()
	assert.Nil(t, cmd)
}

func TestDashboardTableModel_CopyCurrent_WithRevealed(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.revealedSecret = "hunter2"
	cmd := m.copyCurrent()
	assert.NotNil(t, cmd)
}

func TestDashboardTableModel_RevealPayload_LoginPassword(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, "v1", int32(1), mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m, _ := newDashboardTableTestModel(t, server)
	m.vaultID = "v1"
	id := upsertLoginSecret(t, m, "GitHub", "alice", "github.com", "hunter2")
	m.rows = []secret.SummaryRow{{ID: id, Type: 1}}

	cmd := m.revealPayload(id)
	require.NotNil(t, cmd)
	msg := cmd()
	revealed, ok := msg.(payloadRevealedMsg)
	require.True(t, ok)
	assert.Equal(t, "hunter2", revealed.password)
}

func TestDashboardTableModel_ViewLoading(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.loading = true
	out := m.view(80, 10)
	assert.NotEmpty(t, out)
}

func TestDashboardTableModel_ViewEmpty(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.loading = false
	out := m.view(80, 10)
	assert.NotEmpty(t, out)
}

func TestDashboardTableModel_ViewWithRows(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.loading = false
	m.rows = []secret.SummaryRow{{ID: "s1", Title: "GitHub", Subtitle: "alice", URI: "github.com", Type: 1}}
	out := m.view(80, 10)
	assert.Contains(t, out, "GitHub")
}

func TestScrollWindow(t *testing.T) {
	start, end := scrollWindow(0, 5, 10)
	assert.Equal(t, 0, start)
	assert.Equal(t, 5, end)

	start, end = scrollWindow(50, 100, 10)
	assert.Equal(t, 10, end-start)
}

func TestPadTo(t *testing.T) {
	assert.Equal(t, "ab   ", padTo("ab", 5))
	assert.Equal(t, "abcde ", padTo("abcde", 5))
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "abc", truncate("abc", 5))
	assert.Equal(t, "a…", truncate("abc", 2))
	assert.Equal(t, "", truncate("abc", 0))
}

func TestIsMasked(t *testing.T) {
	assert.True(t, isMasked("••••"))
	assert.True(t, isMasked("•• ••"))
	assert.False(t, isMasked(""))
	assert.False(t, isMasked("abc"))
}

func TestDashboardTableModel_SecretTypeAt(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.rows = []secret.SummaryRow{{ID: "s1", Type: 1}}
	assert.EqualValues(t, 1, m.secretTypeAt(0))
	assert.EqualValues(t, 0, m.secretTypeAt(5))
}

func TestDashboardTableModel_CurrentSecretID(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	assert.Empty(t, m.currentSecretID())
	m.rows = []secret.SummaryRow{{ID: "s1"}}
	assert.Equal(t, "s1", m.currentSecretID())
}

func TestDashboardTableModel_TotpTick_NotTOTPType_NoOp(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.secretType = 0
	_, cmd := m.update(totpTickMsg{})
	assert.Nil(t, cmd)
}

func TestDashboardTableModel_CopiedCellExpiredMsg(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.copiedVis = true
	m, _ = m.update(copiedCellExpiredMsg{})
	assert.False(t, m.copiedVis)
}

func TestDashboardTableModel_SetSearchQuery_EmptyUsesLocalFilter(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.allRows = []secret.SummaryRow{{ID: "s1", Title: "GitHub"}}
	m, cmd := m.setSearchQuery("")
	assert.Nil(t, cmd)
	assert.Len(t, m.rows, 1)
}

// setSearchQuery — чисто локальная фильтрация уже загруженных allRows (без сети/usecase),
// поэтому не возвращает команду и сразу применяет фильтр к m.rows.
func TestDashboardTableModel_SetSearchQuery_FiltersLocalRows(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.allRows = []secret.SummaryRow{
		{ID: "s1", Title: "GitHub"},
		{ID: "s2", Title: "Gmail"},
	}
	m = m.applyLocalFilter() // синхронизируем rows с allRows перед поиском

	m, cmd := m.setSearchQuery("git")
	assert.Nil(t, cmd)
	require.Len(t, m.rows, 1)
	assert.Equal(t, "s1", m.rows[0].ID)
}

// Поиск по полям, обогащённым Tier 2b (Bank/Cardholder) — раньше эти поля были пустыми в
// результатах поиска (SearchSummary не вызывал enrichSummaryRow), теперь фильтрация идёт по
// уже обогащённым allRows, так что поиск по банку/держателю карты тоже работает.
func TestDashboardTableModel_SetSearchQuery_MatchesEnrichedFields(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	m.allRows = []secret.SummaryRow{
		{ID: "s1", Title: "Card1", Bank: "Sber", Cardholder: "IVAN IVANOV"},
	}
	m = m.applyLocalFilter()

	m, cmd := m.setSearchQuery("sber")
	assert.Nil(t, cmd)
	require.Len(t, m.rows, 1)
	assert.Equal(t, "s1", m.rows[0].ID)
}

func TestDashboardTableModel_OpenForm_EmptyRows(t *testing.T) {
	m, _ := newDashboardTableTestModel(t, mocks.NewMockServerClient(t))
	cmd := m.openForm()
	assert.Nil(t, cmd)
}
