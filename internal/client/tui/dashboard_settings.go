package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
)

// autolockPresets — доступные значения таймаута автоблокировки (минуты; 0 = никогда).
var autolockPresets = []int{1, 5, 15, 30, 0}

// settingsVaultRow — строка списка папок в Settings (имя + текущий флаг синхронизации).
type settingsVaultRow struct {
	ID          string
	Name        string
	SyncEnabled bool
}

const (
	settingsRowLang = iota
	settingsRowAutolock
	settingsRowTOTPMode
	settingsFixedRows // Количество фиксированных (не-vault) строк
)

type settingsPopup struct {
	ctx    context.Context
	cfg    *app.Container
	vaults []settingsVaultRow
	cursor int
}

func newSettingsPopup(container *app.Container) settingsPopup {
	return settingsPopup{cfg: container}
}

// SetVaults наполняет список папок для показа/редактирования флага синхронизации.
func (m settingsPopup) SetVaults(ctx context.Context, vaults []settingsVaultRow) settingsPopup {
	m.ctx = ctx
	m.vaults = vaults
	m.cursor = 0
	return m
}

func (m settingsPopup) rowCount() int {
	return settingsFixedRows + len(m.vaults)
}

func (m settingsPopup) update(msg tea.KeyMsg) (settingsPopup, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.cursor < m.rowCount()-1 {
			m.cursor++
		}
		return m, nil
	// Пресет-строки (язык/тема/автолок/TOTP-режим) переключаются ←/→ — тот же UX, что у
	// селектора типа секрета в форме (‹ Значение ›). Space/Enter зарезервированы за
	// vault-строками (чекбокс синхронизации), где направление не имеет смысла.
	case tea.KeyLeft:
		return m.cyclePresetRow(-1)
	case tea.KeyRight:
		return m.cyclePresetRow(1)
	case tea.KeySpace, tea.KeyEnter:
		return m.toggleVaultRow()
	}
	return m, nil
}

// cyclePresetRow переключает пресет-строку (язык/тема/автолок/TOTP-режим) на dir шагов
// (-1 или +1). На vault-строках — no-op (там нет цикла пресетов, только чекбокс).
func (m settingsPopup) cyclePresetRow(dir int) (settingsPopup, tea.Cmd) {
	switch m.cursor {
	case settingsRowLang:
		return m, m.cycleLang(dir)
	case settingsRowAutolock:
		return m, m.cycleAutolock(dir)
	case settingsRowTOTPMode:
		return m, m.toggleTOTPMode()
	}
	return m, nil
}

// toggleVaultRow переключает флаг синхронизации выделенного vault'а (Space/Enter).
func (m settingsPopup) toggleVaultRow() (settingsPopup, tea.Cmd) {
	idx := m.cursor - settingsFixedRows
	if idx < 0 || idx >= len(m.vaults) {
		return m, nil
	}
	row := &m.vaults[idx]
	row.SyncEnabled = !row.SyncEnabled
	id, enabled := row.ID, row.SyncEnabled
	container := m.cfg
	ctx := m.ctx
	return m, func() tea.Msg {
		if err := container.Sync.SetVaultSyncEnabled(ctx, id, enabled); err != nil {
			return vaultsErrMsg{err: err}
		}
		return settingsSyncToggledMsg{}
	}
}

// langOptions — доступные языки интерфейса, в порядке цикла ←/→.
var langOptions = []string{"en", "ru"}

// cycleLang переключает язык по кругу (←/→), применяет немедленно (Localizer.SetLang) и
// сохраняет.
func (m settingsPopup) cycleLang(dir int) tea.Cmd {
	container := m.cfg
	next := cycleOption(langOptions, container.Config.Lang, dir)
	container.Localizer.SetLang(next)
	container.Config.Lang = next
	return persistConfig(container)
}

// cycleAutolock переключает таймаут по пресетам (←/→), применяет к App (autolockChangedMsg)
// и сохраняет.
func (m settingsPopup) cycleAutolock(dir int) tea.Cmd {
	container := m.cfg
	next := cycleIntOption(autolockPresets, container.Config.AutolockMinutes, dir)
	container.Config.AutolockMinutes = next
	timeout := autolockTimeoutFromConfig(next)
	persist := persistConfig(container)
	return tea.Batch(
		func() tea.Msg { return autolockChangedMsg{timeout: timeout} },
		persist,
	)
}

// cycleOption возвращает следующий/предыдущий элемент options относительно cur (по кругу).
// Если cur не найден в options — возвращает первый элемент.
func cycleOption(options []string, cur string, dir int) string {
	if len(options) == 0 {
		return cur
	}
	for i, o := range options {
		if o == cur {
			return options[wrapIndex(i+dir, len(options))]
		}
	}
	return options[0]
}

// cycleIntOption — то же, что cycleOption, но для []int (таймаут автолока).
func cycleIntOption(options []int, cur int, dir int) int {
	if len(options) == 0 {
		return cur
	}
	for i, o := range options {
		if o == cur {
			return options[wrapIndex(i+dir, len(options))]
		}
	}
	return options[0]
}

// wrapIndex оборачивает индекс i в диапазон [0, n) (корректно для отрицательных i).
func wrapIndex(i, n int) int {
	return ((i % n) + n) % n
}

// toggleTOTPMode переключает режим показа TOTP-кодов focused<->all и сохраняет.
func (m settingsPopup) toggleTOTPMode() tea.Cmd {
	container := m.cfg
	if container.Config.TOTPRevealMode == "all" {
		container.Config.TOTPRevealMode = "focused"
	} else {
		container.Config.TOTPRevealMode = "all"
	}
	return persistConfig(container)
}

// persistConfig сохраняет конфиг на диск (best-effort: ошибка идёт тостом, но не роняет UI).
func persistConfig(container *app.Container) tea.Cmd {
	cfg := container.Config
	return func() tea.Msg {
		if err := config.Save(cfg); err != nil {
			return toastMsg{text: fmt.Sprintf("config save failed: %v", err)}
		}
		return settingsSavedMsg{}
	}
}

func autolockLabel(minutes int, l localizerT) string {
	if minutes <= 0 {
		return l.T("tui_settings_autolock_never")
	}
	return (time.Duration(minutes) * time.Minute).String()
}

func totpModeLabel(mode string) string {
	if mode == "all" {
		return "all"
	}
	return "focused"
}

func (m settingsPopup) view(l localizerT) string {
	var b strings.Builder
	b.WriteString(styles.Title.Render(l.T("tui_settings_title")))
	b.WriteString("\n\n")

	// Редактируемые строки (курсор + подсветка активной).
	row := func(idx int, label, value string) {
		cursor := "  "
		line := fmt.Sprintf("%s: %s", label, value)
		if idx == m.cursor {
			cursor = "▸ "
			b.WriteString(styles.InputLabel.Render(cursor + line))
		} else {
			b.WriteString(cursor + line)
		}
		b.WriteString("\n")
	}

	// Значения в "‹ Value ›" — тот же визуальный стиль, что у селектора типа секрета в форме
	// (обе переключаются ←/→, оформление сигнализирует пользователю об этом одинаково).
	row(settingsRowLang, l.T("tui_settings_lang"), "‹ "+m.cfg.Config.Lang+" ›")
	row(settingsRowAutolock, l.T("tui_settings_autolock"), "‹ "+autolockLabel(m.cfg.Config.AutolockMinutes, l)+" ›")
	row(settingsRowTOTPMode, l.T("tui_settings_totp_mode"), "‹ "+totpModeLabel(m.cfg.Config.TOTPRevealMode)+" ›")

	// Read-only (требуют перезапуска).
	b.WriteString("\n")
	b.WriteString(styles.HelpText.Render("  " + l.T("tui_settings_server") + ": " + m.cfg.Config.ServerAddr))
	b.WriteString("\n")
	b.WriteString(styles.HelpText.Render("  " + l.T("tui_settings_datadir") + ": " + m.cfg.Config.DataDir))
	b.WriteString("\n")
	b.WriteString(styles.HelpText.Render("  " + l.T("tui_settings_restart_note")))
	b.WriteString("\n\n")

	b.WriteString(styles.InputLabel.Render(l.T("tui_settings_sync_vaults")))
	b.WriteString("\n")
	if len(m.vaults) == 0 {
		b.WriteString(styles.HelpText.Render(l.T("tui_no_vaults")))
		b.WriteString("\n")
	} else {
		for i, v := range m.vaults {
			idx := settingsFixedRows + i
			box := "[ ]"
			if v.SyncEnabled {
				box = "[x]"
			}
			cursor := "  "
			line := cursor + box + " " + v.Name
			if idx == m.cursor {
				line = "▸ " + box + " " + v.Name
				b.WriteString(styles.InputLabel.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpText.Render(l.T("tui_help_settings")))
	return b.String()
}
