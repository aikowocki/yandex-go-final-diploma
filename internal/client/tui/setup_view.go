package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/config"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
)

const (
	setupFieldServer = iota
	setupFieldDataDir
	setupFieldPersist
	setupRowCount
)

type setupModel struct {
	inputs         []textinput.Model // 0: server, 1: data dir
	defaultServer  string
	defaultDataDir string
	noPersist      bool
	focus          int
	done           bool
	saveErr        error
}

func newSetupModel(defaultServer, defaultDataDir string) setupModel {
	server := textinput.New()
	server.Placeholder = "localhost:9090"
	server.SetValue(defaultServer)
	server.CharLimit = 128
	server.Width = 50
	server.Focus()

	dir := textinput.New()
	dir.Placeholder = defaultDataDir
	dir.SetValue(defaultDataDir)
	dir.CharLimit = 256
	dir.Width = 50

	return setupModel{
		inputs:         []textinput.Model{server, dir},
		defaultServer:  defaultServer,
		defaultDataDir: defaultDataDir,
	}
}

func (m setupModel) Init() tea.Cmd { return textinput.Blink }

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			// Отказ от настройки — выходим с дефолтами (config.json не пишем; используются
			// значения по умолчанию kong при следующем запуске).
			m.done = true
			return m, tea.Quit
		case tea.KeyTab, tea.KeyDown:
			m = m.focusNext()
			return m, nil
		case tea.KeyShiftTab, tea.KeyUp:
			m = m.focusPrev()
			return m, nil
		case tea.KeySpace:
			if m.focus == setupFieldPersist {
				m.noPersist = !m.noPersist
				return m, nil
			}
		case tea.KeyEnter:
			if m.focus == setupFieldPersist {
				m.done = true
				m.saveErr = m.save()
				return m, tea.Quit
			}
			m = m.focusNext()
			return m, nil
		}
	}

	if m.focus < len(m.inputs) {
		var cmd tea.Cmd
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m setupModel) focusNext() setupModel {
	m = m.blurAll()
	m.focus = (m.focus + 1) % setupRowCount
	return m.focusCurrent()
}

func (m setupModel) focusPrev() setupModel {
	m = m.blurAll()
	m.focus = (m.focus - 1 + setupRowCount) % setupRowCount
	return m.focusCurrent()
}

func (m setupModel) blurAll() setupModel {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	return m
}

func (m setupModel) focusCurrent() setupModel {
	if m.focus < len(m.inputs) {
		m.inputs[m.focus].Focus()
	}
	return m
}

// save пишет config.json с выбранными значениями. Пустые поля (например, если пользователь
// случайно стёр значение перед enter) откатываются на дефолты, а не сохраняются как есть —
// иначе клиент затем пытается подключаться к пустому server_addr и получает "server unavailable".
func (m setupModel) save() error {
	serverAddr := strings.TrimSpace(m.inputs[setupFieldServer].Value())
	if serverAddr == "" {
		serverAddr = m.defaultServer
	}
	dataDir := strings.TrimSpace(m.inputs[setupFieldDataDir].Value())
	if dataDir == "" {
		dataDir = m.defaultDataDir
	}

	cfg := &config.ClientConfig{
		ServerAddr:      serverAddr,
		DataDir:         dataDir,
		NoPersist:       m.noPersist,
		LogLevel:        "info",
		Lang:            "en",
		AutolockMinutes: 5,
		TOTPRevealMode:  "focused",
	}
	return config.Save(cfg)
}

func (m setupModel) View() string {
	var b []byte
	out := func(s string) { b = append(b, s...) }

	out(styles.Title.Render("🚀 GophKeeper — First-run setup"))
	out("\n\n")

	labelServer := "  Server address:"
	labelDir := "  Data directory:"
	persist := "  [x] Store data locally"
	if m.noPersist {
		persist = "  [ ] Store data locally"
	}
	if m.focus == setupFieldServer {
		labelServer = styles.InputLabel.Render("▸ Server address:")
	}
	if m.focus == setupFieldDataDir {
		labelDir = styles.InputLabel.Render("▸ Data directory:")
	}
	if m.focus == setupFieldPersist {
		persist = styles.InputLabel.Render("▸ " + strings.TrimLeft(persist, " "))
	}

	out(labelServer)
	out("\n  ")
	out(m.inputs[setupFieldServer].View())
	out("\n\n")
	out(labelDir)
	out("\n  ")
	out(m.inputs[setupFieldDataDir].View())
	out("\n\n")
	out(persist)
	out("\n\n")

	if m.saveErr != nil {
		out(styles.ErrorText.Render("✗ " + m.saveErr.Error()))
		out("\n\n")
	}

	out(styles.HelpText.Render("tab/↑↓: navigate • space: toggle persist • enter: save & continue • esc: skip (defaults)"))
	return string(b)
}

// RunOnboarding запускает wizard первого запуска и возвращает ошибку сохранения (если была).
// Вызывается из main.go, когда config.json ещё не существует.
func RunOnboarding(defaultServer, defaultDataDir string) error {
	m := newSetupModel(defaultServer, defaultDataDir)
	final, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	if sm, ok := final.(setupModel); ok {
		return sm.saveErr
	}
	return nil
}
