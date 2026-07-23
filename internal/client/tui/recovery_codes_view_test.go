package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
)

func newRecoveryCodesTestModel(t *testing.T) recoveryCodesModel {
	t.Helper()
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	return newRecoveryCodesModel(c, []string{"AAAA-BBBB", "CCCC-DDDD"})
}

func TestRecoveryCodesModel_EnterGoesToDashboard(t *testing.T) {
	m := newRecoveryCodesTestModel(t)
	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenDashboard, switchMsg.screen)
}

func TestRecoveryCodesModel_EscGoesToDashboard(t *testing.T) {
	m := newRecoveryCodesTestModel(t)
	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenDashboard, switchMsg.screen)
}

func TestRecoveryCodesModel_CopyShowsToast(t *testing.T) {
	m := newRecoveryCodesTestModel(t)
	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(toastMsg)
	assert.True(t, ok)
}

func TestRecoveryCodesModel_ViewRendersCodes(t *testing.T) {
	m := newRecoveryCodesTestModel(t)
	out := m.view(80, 24)
	assert.Contains(t, out, "AAAA-BBBB")
	assert.Contains(t, out, "CCCC-DDDD")
}
