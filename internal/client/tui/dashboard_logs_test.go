package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeLogFile(t *testing.T, dir string, lines []string) {
	t.Helper()
	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client.log"), []byte(content), 0o600))
}

func TestNewLogsPopup_FileNotFound(t *testing.T) {
	m := newLogsPopup(t.TempDir())
	require.Error(t, m.err)
}

func TestNewLogsPopup_ReadsLines(t *testing.T) {
	dir := t.TempDir()
	writeLogFile(t, dir, []string{"line1", "line2", "line3"})

	m := newLogsPopup(dir)
	require.NoError(t, m.err)
	assert.Len(t, m.lines, 3)
}

func TestNewLogsPopup_TruncatesToMaxKeep(t *testing.T) {
	dir := t.TempDir()
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = "line"
	}
	writeLogFile(t, dir, lines)

	m := newLogsPopup(dir)
	require.NoError(t, m.err)
	assert.LessOrEqual(t, len(m.lines), logsMaxLines*3)
}

func TestLogsPopup_UpdateScroll(t *testing.T) {
	dir := t.TempDir()
	lines := make([]string, 60)
	for i := range lines {
		lines[i] = "line"
	}
	writeLogFile(t, dir, lines)
	m := newLogsPopup(dir)

	m = m.update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, m.offset)

	m = m.update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, m.offset)

	// Down не должен уходить в минус.
	m = m.update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, m.offset)
}

func TestLogsPopup_UpdateHomeEnd(t *testing.T) {
	dir := t.TempDir()
	lines := make([]string, 60)
	for i := range lines {
		lines[i] = "line"
	}
	writeLogFile(t, dir, lines)
	m := newLogsPopup(dir)

	m = m.update(tea.KeyMsg{Type: tea.KeyHome})
	assert.Equal(t, len(m.lines)-logsMaxLines, m.offset)

	m = m.update(tea.KeyMsg{Type: tea.KeyEnd})
	assert.Equal(t, 0, m.offset)
}

func TestLogsPopup_UpdateMouseWheel(t *testing.T) {
	dir := t.TempDir()
	lines := make([]string, 60)
	for i := range lines {
		lines[i] = "line"
	}
	writeLogFile(t, dir, lines)
	m := newLogsPopup(dir)

	m = m.updateMouse(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	assert.Equal(t, 3, m.offset)

	m = m.updateMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 0, m.offset)

	// Не уходит в минус.
	m = m.updateMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 0, m.offset)
}

func TestLogsPopup_Clear(t *testing.T) {
	dir := t.TempDir()
	writeLogFile(t, dir, []string{"line1"})
	m := newLogsPopup(dir)
	require.NotEmpty(t, m.lines)

	m = m.clear(dir)
	assert.Empty(t, m.lines)
	assert.Equal(t, 0, m.offset)
	assert.NoError(t, m.err)

	data, err := os.ReadFile(filepath.Join(dir, "client.log"))
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestLogsPopup_View_Error(t *testing.T) {
	m := newLogsPopup(t.TempDir())
	out := m.view()
	assert.NotEmpty(t, out)
}

func TestLogsPopup_View_Empty(t *testing.T) {
	dir := t.TempDir()
	writeLogFile(t, dir, []string{""})
	m := newLogsPopup(dir)
	m.lines = nil
	out := m.view()
	assert.Contains(t, out, "empty")
}

func TestLogsPopup_View_WithLines(t *testing.T) {
	dir := t.TempDir()
	writeLogFile(t, dir, []string{`{"time":"2026-07-17T16:27:38Z","level":"INFO","msg":"hello"}`})
	m := newLogsPopup(dir)
	out := m.view()
	assert.Contains(t, out, "hello")
}

func TestLogsPopup_View_WarnLineHighlighted(t *testing.T) {
	dir := t.TempDir()
	writeLogFile(t, dir, []string{`{"time":"2026-07-17T16:27:38Z","level":"WARN","msg":"careful"}`})
	m := newLogsPopup(dir)
	out := m.view()
	assert.Contains(t, out, "careful")
}

func TestTruncateLog(t *testing.T) {
	assert.Equal(t, "short", truncateLog("short", 10))
	assert.Equal(t, "abcdefghi…", truncateLog("abcdefghijklmnop", 10))
}

func TestFormatLogLine_NonJSON(t *testing.T) {
	got := formatLogLine("plain text line")
	assert.Equal(t, "plain text line", got)
}

func TestFormatLogLine_JSON(t *testing.T) {
	got := formatLogLine(`{"time":"2026-07-17T16:27:38.123Z","level":"INFO","msg":"hello","key":"val"}`)
	assert.Contains(t, got, "16:27:38")
	assert.Contains(t, got, "INFO")
	assert.Contains(t, got, "hello")
	assert.Contains(t, got, "key=val")
}

func TestFormatLogLine_MissingFields(t *testing.T) {
	got := formatLogLine(`{"msg":"hello"}`)
	assert.Contains(t, got, "hello")
}
