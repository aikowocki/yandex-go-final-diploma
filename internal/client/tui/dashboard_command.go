package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
)

// commandSpec — одна команда палитры: каноническое имя (после "/") + ключ локализации описания.
type commandSpec struct {
	name    string
	descKey string
}

// availableCommands — список команд, показываемых в палитре.
var availableCommands = []commandSpec{
	{name: "sync", descKey: "tui_cmd_sync"},
	{name: "new", descKey: "tui_cmd_new"},
	{name: "vault", descKey: "tui_cmd_vault"},
	{name: "lock", descKey: "tui_cmd_lock"},
	{name: "logs", descKey: "tui_cmd_logs"},
	{name: "conflicts", descKey: "tui_cmd_conflicts"},
	{name: "quit", descKey: "tui_cmd_quit"},
}

type commandLineModel struct {
	input            textinput.Model
	suggestionCursor int
}

func newCommandLineModel() commandLineModel {
	ti := textinput.New()
	ti.CharLimit = 128
	ti.Width = 60
	return commandLineModel{input: ti}
}

// activate фокусирует строку с чистым значением — шорткат "/" открывает пустую строку живого
// поиска (как раньше), а не сразу вставляет "/". Командный режим включается, только когда
// пользователь сам печатает "/" первым символом.
func (m commandLineModel) activate() commandLineModel {
	m.input.SetValue("")
	m.input.CursorEnd()
	m.input.Focus()
	m.suggestionCursor = 0
	return m
}

func (m commandLineModel) reset() commandLineModel {
	m.input.SetValue("")
	m.input.Blur()
	m.suggestionCursor = 0
	return m
}

// isCommand сообщает, что строка сейчас в командном режиме (введено "/").
func (m commandLineModel) isCommand() bool {
	return strings.HasPrefix(m.input.Value(), "/")
}

func (m commandLineModel) value() string {
	return m.input.Value()
}

// commandText возвращает введённый текст команды без "/"-префикса (пусто, если режим ещё
// не активирован).
func (m commandLineModel) commandText() string {
	return strings.TrimPrefix(m.input.Value(), "/")
}

// searchQuery возвращает текущий текст без учёта командного режима (пустая строка, если
// сейчас введена команда, а не поиск).
func (m commandLineModel) searchQuery() string {
	if m.isCommand() {
		return ""
	}
	return m.input.Value()
}

// filteredCommands возвращает команды палитры, начинающиеся с уже введённого текста команды.
func (m commandLineModel) filteredCommands() []commandSpec {
	q := strings.ToLower(m.commandText())
	if q == "" {
		return availableCommands
	}
	out := make([]commandSpec, 0, len(availableCommands))
	for _, c := range availableCommands {
		if strings.HasPrefix(c.name, q) {
			out = append(out, c)
		}
	}
	return out
}

// clampSuggestionCursor удерживает suggestionCursor в границах текущего отфильтрованного списка.
func (m commandLineModel) clampSuggestionCursor() commandLineModel {
	n := len(m.filteredCommands())
	if n == 0 {
		m.suggestionCursor = 0
		return m
	}
	if m.suggestionCursor < 0 {
		m.suggestionCursor = n - 1
	}
	if m.suggestionCursor >= n {
		m.suggestionCursor = 0
	}
	return m
}

// moveSuggestion сдвигает выделение в палитре на delta (с оборачиванием).
func (m commandLineModel) moveSuggestion(delta int) commandLineModel {
	filtered := m.filteredCommands()
	if len(filtered) == 0 {
		return m
	}
	n := len(filtered)
	m.suggestionCursor = ((m.suggestionCursor+delta)%n + n) % n
	return m
}

// acceptSuggestion подставляет в строку ввода имя выделенной команды палитры (автодополнение
// по Tab), не запуская её — запуск остаётся за Enter.
func (m commandLineModel) acceptSuggestion() commandLineModel {
	filtered := m.filteredCommands()
	if len(filtered) == 0 {
		return m
	}
	m = m.clampSuggestionCursor()
	m.input.SetValue("/" + filtered[m.suggestionCursor].name)
	m.input.CursorEnd()
	return m
}

// commandToRun возвращает имя команды, которую нужно выполнить по Enter: введённый текст,
// если он совпадает с существующей командой, иначе — имя текущей выделенной подсказки палитры
// (если список после фильтрации не пуст), иначе исходный текст (неизвестная команда — no-op).
func (m commandLineModel) commandToRun() string {
	typed := strings.ToLower(strings.TrimSpace(m.commandText()))
	for _, c := range availableCommands {
		if c.name == typed {
			return typed
		}
	}
	filtered := m.filteredCommands()
	if len(filtered) == 0 {
		return typed
	}
	idx := m.suggestionCursor
	if idx < 0 || idx >= len(filtered) {
		idx = 0
	}
	return filtered[idx].name
}

func (m commandLineModel) update(msg tea.Msg) (commandLineModel, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.isCommand() {
		m = m.clampSuggestionCursor()
	}
	return m, cmd
}

func (m commandLineModel) view(l localizerT, focused bool) string {
	if !focused && m.input.Value() == "" {
		return styles.HelpText.Render(l.T("tui_command_placeholder"))
	}
	if !m.isCommand() {
		return m.input.View()
	}

	var b strings.Builder
	b.WriteString(m.input.View())
	filtered := m.filteredCommands()
	if len(filtered) == 0 {
		b.WriteString("\n")
		b.WriteString(styles.HelpText.Render(l.T("tui_cmd_unknown")))
		return b.String()
	}
	idx := m.suggestionCursor
	if idx < 0 || idx >= len(filtered) {
		idx = 0
	}
	b.WriteString("\n")
	for i, c := range filtered {
		label := "/" + c.name
		if i == idx {
			b.WriteString(styles.InputLabel.Render("▸ " + label))
		} else {
			b.WriteString(styles.HelpText.Render("  " + label))
		}
		if i < len(filtered)-1 {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")
	b.WriteString(styles.HelpText.Render(l.T(filtered[idx].descKey)))
	return b.String()
}
