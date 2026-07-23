package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSetupModel_Defaults(t *testing.T) {
	m := newSetupModel("localhost:9090", "/tmp/data")
	assert.Equal(t, "localhost:9090", m.inputs[setupFieldServer].Value())
	assert.Equal(t, "/tmp/data", m.inputs[setupFieldDataDir].Value())
	assert.False(t, m.noPersist)
	assert.Equal(t, 0, m.focus)
}

func TestSetupModel_EscQuits(t *testing.T) {
	m := newSetupModel("addr", "/tmp")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	sm := updated.(setupModel)
	assert.True(t, sm.done)
	assert.NotNil(t, cmd)
}

func TestSetupModel_CtrlCQuits(t *testing.T) {
	m := newSetupModel("addr", "/tmp")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	sm := updated.(setupModel)
	assert.True(t, sm.done)
	assert.NotNil(t, cmd)
}

func TestSetupModel_TabNavigatesFocus(t *testing.T) {
	m := newSetupModel("addr", "/tmp")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	sm := updated.(setupModel)
	assert.Equal(t, setupFieldDataDir, sm.focus)

	updated, _ = sm.Update(tea.KeyMsg{Type: tea.KeyTab})
	sm = updated.(setupModel)
	assert.Equal(t, setupFieldPersist, sm.focus)

	// Оборачивается на начало.
	updated, _ = sm.Update(tea.KeyMsg{Type: tea.KeyTab})
	sm = updated.(setupModel)
	assert.Equal(t, setupFieldServer, sm.focus)
}

func TestSetupModel_ShiftTabGoesBackward(t *testing.T) {
	m := newSetupModel("addr", "/tmp")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	sm := updated.(setupModel)
	assert.Equal(t, setupFieldPersist, sm.focus)
}

func TestSetupModel_SpaceTogglesPersistOnlyOnPersistField(t *testing.T) {
	m := newSetupModel("addr", "/tmp")
	m.focus = setupFieldServer

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	sm := updated.(setupModel)
	assert.False(t, sm.noPersist, "space должен вводить пробел в текстовое поле, не переключать persist")

	m2 := newSetupModel("addr", "/tmp")
	m2.focus = setupFieldPersist
	updated2, _ := m2.Update(tea.KeyMsg{Type: tea.KeySpace})
	sm2 := updated2.(setupModel)
	assert.True(t, sm2.noPersist)
}

func TestSetupModel_EnterOnNonPersistFieldMovesFocus(t *testing.T) {
	m := newSetupModel("addr", "/tmp")
	m.focus = setupFieldServer

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	sm := updated.(setupModel)
	assert.Equal(t, setupFieldDataDir, sm.focus)
	assert.Nil(t, cmd)
}

func TestSetupModel_EnterOnPersistFieldSaves(t *testing.T) {
	// config.Save пишет в os.UserConfigDir()/gophkeeper/config.json, а не в dir — подменяем
	// HOME/XDG_CONFIG_HOME, чтобы тест не затирал настоящий конфиг пользователя.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	dir := t.TempDir()
	m := newSetupModel("localhost:9090", dir)
	m.focus = setupFieldPersist

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	sm := updated.(setupModel)
	assert.True(t, sm.done)
	assert.NoError(t, sm.saveErr)
	require.NotNil(t, cmd)

	configDir, err := os.UserConfigDir()
	require.NoError(t, err)
	savedPath := filepath.Join(configDir, "gophkeeper", "config.json")
	_, err = os.Stat(savedPath)
	require.NoError(t, err, "config.json should be written under the fake HOME, not the real user config dir")
}

func TestSetupModel_TypingUpdatesFocusedField(t *testing.T) {
	m := newSetupModel("", "/tmp")
	m.focus = setupFieldServer
	m.inputs[setupFieldServer].Focus()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	sm := updated.(setupModel)
	assert.Equal(t, "x", sm.inputs[setupFieldServer].Value())
}

func TestSetupModel_Save_WritesConfig(t *testing.T) {
	// config.Save всегда пишет в os.UserConfigDir()/gophkeeper/config.json (реальный конфиг
	// пользователя), а не в cfg.DataDir — поэтому подменяем HOME/XDG_CONFIG_HOME на временный
	// каталог, чтобы тест не затирал настоящий config.json на машине разработчика/CI.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	dir := t.TempDir()
	m := newSetupModel("myserver:1234", dir)
	m.noPersist = true

	err := m.save()
	require.NoError(t, err)

	configDir, cfgErr := os.UserConfigDir()
	require.NoError(t, cfgErr)
	savedPath := filepath.Join(configDir, "gophkeeper", "config.json")
	_, statErr := os.Stat(savedPath)
	require.NoError(t, statErr, "config.json should be written under the fake HOME, not the real user config dir")
}

func TestSetupModel_View_RendersFields(t *testing.T) {
	m := newSetupModel("localhost:9090", "/tmp/data")
	out := m.View()
	assert.Contains(t, out, "Server address")
	assert.Contains(t, out, "Data directory")
	assert.Contains(t, out, "Store data locally")
}

func TestSetupModel_View_ShowsPersistCheckedByDefault(t *testing.T) {
	m := newSetupModel("addr", "/tmp")
	out := m.View()
	assert.Contains(t, out, "[x]")
}

func TestSetupModel_View_ShowsPersistUncheckedWhenToggled(t *testing.T) {
	m := newSetupModel("addr", "/tmp")
	m.noPersist = true
	out := m.View()
	assert.Contains(t, out, "[ ]")
}

func TestSetupModel_View_ShowsSaveError(t *testing.T) {
	m := newSetupModel("addr", "/tmp")
	m.saveErr = assert.AnError
	out := m.View()
	assert.Contains(t, out, assert.AnError.Error())
}

func TestSetupModel_Init(t *testing.T) {
	m := newSetupModel("addr", "/tmp")
	cmd := m.Init()
	assert.NotNil(t, cmd)
}
