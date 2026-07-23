package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
)

const logsMaxLines = 30

type logsPopup struct {
	lines  []string
	offset int // scroll offset (0 = bottom)
	err    error
}

func newLogsPopup(dataDir string) logsPopup {
	logPath := filepath.Join(dataDir, "client.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		return logsPopup{err: err}
	}
	all := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	// Берём последние logsMaxLines*2 строк (чтобы можно было скроллить вверх).
	maxKeep := logsMaxLines * 3
	if len(all) > maxKeep {
		all = all[len(all)-maxKeep:]
	}
	return logsPopup{lines: all, offset: 0}
}

func (m logsPopup) update(msg tea.KeyMsg) logsPopup {
	switch msg.Type {
	case tea.KeyUp:
		if m.offset < len(m.lines)-logsMaxLines {
			m.offset++
		}
	case tea.KeyDown:
		if m.offset > 0 {
			m.offset--
		}
	case tea.KeyHome:
		m.offset = max(0, len(m.lines)-logsMaxLines)
	case tea.KeyEnd:
		m.offset = 0
	}
	return m
}

func (m logsPopup) updateMouse(msg tea.MouseMsg) logsPopup {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.offset < len(m.lines)-logsMaxLines {
			m.offset += 3
			if m.offset > len(m.lines)-logsMaxLines {
				m.offset = len(m.lines) - logsMaxLines
			}
		}
	case tea.MouseButtonWheelDown:
		m.offset -= 3
		if m.offset < 0 {
			m.offset = 0
		}
	}
	return m
}

// сlear очищает файл логов и сбрасывает содержимое попапа.
func (m logsPopup) clear(dataDir string) logsPopup {
	logPath := filepath.Join(dataDir, "client.log")
	_ = os.Truncate(logPath, 0)
	m.lines = nil
	m.offset = 0
	m.err = nil
	return m
}

func (m logsPopup) view() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Logs (client.log)"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(styles.ErrorText.Render(m.err.Error()))
		b.WriteString("\n")
		return b.String()
	}

	if len(m.lines) == 0 {
		b.WriteString(styles.HelpText.Render("(empty)"))
		return b.String()
	}

	// Показываем logsMaxLines строк с учётом offset (offset=0 → самые последние).
	end := len(m.lines) - m.offset
	start := end - logsMaxLines
	if start < 0 {
		start = 0
	}
	if end > len(m.lines) {
		end = len(m.lines)
	}

	for _, line := range m.lines[start:end] {
		rendered := formatLogLine(line)
		if strings.Contains(line, `"level":"WARN"`) || strings.Contains(line, `"level":"ERROR"`) {
			b.WriteString(styles.ErrorText.Render(rendered))
		} else {
			b.WriteString(rendered)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpText.Render("↑↓/scroll: navigate • home/end: top/bottom • d: clear logs • esc/q: close"))
	return b.String()
}

func truncateLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

// formatLogLine парсит JSON-строку лога и форматирует в читаемый вид:
// "16:27:38 WARN  msg  key=val key2=val2"
func formatLogLine(raw string) string {
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		// Не JSON — возвращаем как есть (обрезав).
		return truncateLog(raw, 120)
	}

	// Извлекаем время (только HH:MM:SS).
	timeStr := ""
	if t, ok := entry["time"].(string); ok && len(t) >= 19 {
		// "2026-07-15T16:27:38..." → "16:27:38"
		if idx := strings.IndexByte(t, 'T'); idx >= 0 && len(t) > idx+9 {
			timeStr = t[idx+1 : idx+9]
		}
	}

	level := ""
	if l, ok := entry["level"].(string); ok {
		level = fmt.Sprintf("%-5s", l)
	}

	msg := ""
	if m, ok := entry["msg"].(string); ok {
		msg = m
	}

	// Остальные поля (кроме time/level/msg) — как key=val.
	var extras []string
	for k, v := range entry {
		if k == "time" || k == "level" || k == "msg" {
			continue
		}
		extras = append(extras, fmt.Sprintf("%s=%v", k, v))
	}

	result := timeStr + " " + level + " " + msg
	if len(extras) > 0 {
		result += "  " + strings.Join(extras, " ")
	}
	return truncateLog(result, 120)
}
