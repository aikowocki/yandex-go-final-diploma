package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
)

// --- vaultCreateModel ---

func TestVaultCreateModel_ValueTrimsSpace(t *testing.T) {
	m := newVaultCreateModel()
	m.input.SetValue("  My Vault  ")
	assert.Equal(t, "My Vault", m.value())
}

func TestVaultCreateModel_Update(t *testing.T) {
	m := newVaultCreateModel()
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Equal(t, "a", m.value())
}

func TestVaultCreateModel_View(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	m := newVaultCreateModel()
	out := m.view(c.Localizer)
	assert.NotEmpty(t, out)
}

// --- commandLineModel ---

func TestCommandLineModel_ActivateAsSearch(t *testing.T) {
	m := newCommandLineModel()
	m = m.activate()
	assert.Equal(t, "", m.value())
	assert.False(t, m.isCommand())
}

// Печатая "/" первым символом после activate(), пользователь переводит строку в командный режим.
func TestCommandLineModel_TypingSlash_EntersCommandMode(t *testing.T) {
	m := newCommandLineModel()
	m = m.activate()
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.Equal(t, "/", m.value())
	assert.True(t, m.isCommand())
}

func TestCommandLineModel_Reset(t *testing.T) {
	m := newCommandLineModel()
	m = m.activate()
	m.input.SetValue("/sync")
	m = m.reset()
	assert.Equal(t, "", m.value())
}

func TestCommandLineModel_SearchQuery_EmptyWhenCommand(t *testing.T) {
	m := newCommandLineModel()
	m.input.SetValue("/sync")
	assert.Equal(t, "", m.searchQuery())
}

func TestCommandLineModel_SearchQuery_ReturnsTextWhenNotCommand(t *testing.T) {
	m := newCommandLineModel()
	m.input.SetValue("github")
	assert.Equal(t, "github", m.searchQuery())
}

func TestCommandLineModel_Update(t *testing.T) {
	m := newCommandLineModel()
	m = m.activate()
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Equal(t, "x", m.value())
}

func TestCommandLineModel_View_PlaceholderWhenUnfocused(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	m := newCommandLineModel()
	out := m.view(c.Localizer, false)
	assert.NotEmpty(t, out)
}

func TestCommandLineModel_View_FocusedShowsInput(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	m := newCommandLineModel()
	m = m.activate()
	m.input.SetValue("/")
	out := m.view(c.Localizer, true)
	assert.Contains(t, out, "/")
}

// --- палитра команд ---

func TestCommandLineModel_FilteredCommands_AllWhenEmpty(t *testing.T) {
	m := newCommandLineModel()
	m.input.SetValue("/")
	assert.Len(t, m.filteredCommands(), len(availableCommands))
}

func TestCommandLineModel_FilteredCommands_FiltersByPrefix(t *testing.T) {
	m := newCommandLineModel()
	m.input.SetValue("/log")
	filtered := m.filteredCommands()
	require.Len(t, filtered, 1)
	assert.Equal(t, "logs", filtered[0].name)
}

func TestCommandLineModel_MoveSuggestion_WrapsAround(t *testing.T) {
	m := newCommandLineModel()
	m.input.SetValue("/")
	m = m.moveSuggestion(-1)
	assert.Equal(t, len(availableCommands)-1, m.suggestionCursor)
	m = m.moveSuggestion(1)
	assert.Equal(t, 0, m.suggestionCursor)
}

func TestCommandLineModel_AcceptSuggestion_FillsInput(t *testing.T) {
	m := newCommandLineModel()
	m.input.SetValue("/log")
	m = m.acceptSuggestion()
	assert.Equal(t, "/logs", m.value())
}

func TestCommandLineModel_CommandToRun_ExactMatch(t *testing.T) {
	m := newCommandLineModel()
	m.input.SetValue("/sync")
	assert.Equal(t, "sync", m.commandToRun())
}

func TestCommandLineModel_CommandToRun_FallsBackToHighlightedSuggestion(t *testing.T) {
	m := newCommandLineModel()
	m.input.SetValue("/log")
	assert.Equal(t, "logs", m.commandToRun())
}

// --- userMenuPopup ---

func TestUserMenuPopup_View_NoLoginShowsDash(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	m := userMenuPopup{}
	out := m.view(context.Background(), c, c.Localizer)
	assert.Contains(t, out, "—")
}
