package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
)

const maxUnlockAttempts = 5

// lockMode — режим ввода на экране разблокировки.
type lockMode int

const (
	lockModePassword lockMode = iota // ввод полного master-пароля
	lockModePIN                      // ввод PIN (доступен, если сессия «тёплая» и PIN установлен)
	lockModeSetPIN                   // предложение задать PIN после успешной разблокировки
	lockModeRecovery                 // ввод recovery code (забыл пароль)
)

type lockModel struct {
	container *app.Container
	input     textinput.Model
	mode      lockMode
	err       error
	attempts  int
	cold      bool // превышен лимит попыток
}

func newLockModel(container *app.Container) lockModel {
	ti := textinput.New()
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()
	ti.CharLimit = 128
	ti.Width = 40

	m := lockModel{container: container, input: ti}
	// Если PIN установлен (сессия тёплая после авто-лока) — по умолчанию просим PIN.
	if container.Session.HasPIN() {
		m.mode = lockModePIN
	}
	return m
}

func (m lockModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m lockModel) update(msg tea.Msg) (lockModel, tea.Cmd) {
	if m.cold {
		return m, nil // холодное состояние — только Ctrl+C (в App)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := m.input.Value()
			if val == "" {
				return m, nil
			}
			m.input.SetValue("")
			switch m.mode {
			case lockModePIN:
				return m, m.tryUnlockPIN(val)
			case lockModeSetPIN:
				return m, m.trySetPIN(val)
			case lockModeRecovery:
				return m, m.tryRecovery(val)
			default:
				return m, m.tryUnlock(val)
			}
		case "tab":
			// Переключение PIN ↔ master-пароль (только когда PIN доступен).
			if m.container.Session.HasPIN() && m.mode != lockModeSetPIN {
				if m.mode == lockModePIN {
					m.mode = lockModePassword
				} else {
					m.mode = lockModePIN
				}
				m.err = nil
				m.input.SetValue("")
			}
			return m, nil
		case "esc":
			if m.mode == lockModeSetPIN {
				// Отказ от установки PIN — сразу на dashboard.
				return m, func() tea.Msg { return switchScreenMsg{screen: screenDashboard} }
			}
			// Смена аккаунта: полная блокировка (сброс PIN) и на логин.
			m.container.Session.Lock()
			return m, func() tea.Msg { return switchScreenMsg{screen: screenLogin} }
		case "ctrl+r":
			// Восстановление по recovery code.
			m.mode = lockModeRecovery
			m.input.SetValue("")
			m.input.Placeholder = "XXXX-XXXX-XXXX-XXXX-..."
			m.err = nil
			return m, nil
		}

	case unlockSuccessMsg:
		m.err = nil
		m.attempts = 0
		// После разблокировки полным паролем, если PIN ещё не задан — предлагаем задать.
		if !m.container.Session.HasPIN() && m.mode == lockModePassword {
			m.mode = lockModeSetPIN
			m.input.SetValue("")
			return m, nil
		}
		return m, func() tea.Msg { return switchScreenMsg{screen: screenDashboard} }

	case pinSetMsg:
		return m, func() tea.Msg { return switchScreenMsg{screen: screenDashboard} }

	case unlockErrMsg:
		m.attempts++
		m.err = msg.err
		if m.attempts >= maxUnlockAttempts {
			m.cold = true
			m.container.Session.Lock()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m lockModel) tryUnlock(passphrase string) tea.Cmd {
	container := m.container
	return func() tea.Msg {
		if err := container.Auth.Unlock(context.Background(), []byte(passphrase)); err != nil {
			return unlockErrMsg{err: err}
		}
		return unlockSuccessMsg{}
	}
}

func (m lockModel) tryRecovery(code string) tea.Cmd {
	container := m.container
	return func() tea.Msg {
		if err := container.Auth.RecoverWithCode(context.Background(), code); err != nil {
			return unlockErrMsg{err: err}
		}
		// Recovery успешен — MasterKey в сессии. Переходим к SetupEncryption (задать новый пароль).
		return switchScreenMsg{screen: screenSetupEncryption}
	}
}

func (m lockModel) tryUnlockPIN(pin string) tea.Cmd {
	container := m.container
	return func() tea.Msg {
		if err := container.Auth.UnlockWithPIN([]byte(pin)); err != nil {
			return unlockErrMsg{err: err}
		}
		return unlockSuccessMsg{}
	}
}

func (m lockModel) trySetPIN(pin string) tea.Cmd {
	container := m.container
	return func() tea.Msg {
		if err := container.Auth.SetPIN([]byte(pin)); err != nil {
			return unlockErrMsg{err: err}
		}
		return pinSetMsg{}
	}
}

func (m lockModel) view(width, height int) string {
	var b strings.Builder
	l := m.container.Localizer

	b.WriteString(styles.Title.Render("🔒 " + l.T("tui_lock_title")))
	b.WriteString("\n")
	login := m.container.Auth.CurrentLogin(context.Background())
	if login != "" {
		b.WriteString(styles.HelpText.Render("  " + l.T("tui_lock_current_user") + ": " + login))
	}
	b.WriteString("\n")

	if m.cold {
		b.WriteString(styles.ErrorText.Render(l.T("tui_lock_frozen")))
		b.WriteString("\n")
		b.WriteString(styles.HelpText.Render(l.T("tui_lock_frozen_hint")))
		return styles.Centered(width).Render(b.String())
	}

	var prompt, help string
	switch m.mode {
	case lockModePIN:
		prompt = l.T("tui_lock_enter_pin")
		help = l.T("tui_help_lock_pin")
	case lockModeSetPIN:
		prompt = l.T("tui_lock_set_pin")
		help = l.T("tui_help_lock_setpin")
	case lockModeRecovery:
		prompt = l.T("tui_lock_enter_recovery")
		help = l.T("tui_help_lock_recovery")
	default:
		prompt = l.T("tui_field_master_passphrase") + ":"
		help = l.T("tui_help_lock")
	}

	b.WriteString(styles.InputLabel.Render(prompt))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(styles.ErrorText.Render(fmt.Sprintf("✗ %v", m.err)))
		if m.attempts > 0 {
			_, _ = fmt.Fprintf(&b, " (%d/%d)", m.attempts, maxUnlockAttempts)
		}
		b.WriteString("\n\n")
	}

	b.WriteString(styles.HelpText.Render(help))
	return styles.Centered(width).Render(b.String())
}
