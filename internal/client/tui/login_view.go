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

type loginMode int

const (
	loginModeLogin loginMode = iota
	loginModeRegister
)

type loginModel struct {
	container *app.Container
	mode      loginMode
	inputs    []textinput.Model // 0: login, 1: password
	focus     int
	err       error
	loading   bool
}

func newLoginModel(container *app.Container) loginModel {
	loginInput := textinput.New()
	loginInput.Placeholder = "Username"
	loginInput.CharLimit = 64
	loginInput.Width = 40
	loginInput.Focus()

	passInput := textinput.New()
	passInput.Placeholder = "Password"
	passInput.EchoMode = textinput.EchoPassword
	passInput.EchoCharacter = '•'
	passInput.CharLimit = 128
	passInput.Width = 40

	return loginModel{
		container: container,
		mode:      loginModeLogin,
		inputs:    []textinput.Model{loginInput, passInput},
	}
}

func (m loginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m loginModel) update(msg tea.Msg) (loginModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.mode = 1 - m.mode // toggle login/register
			m.err = nil
			return m, nil
		case "shift+tab", "up":
			m = m.focusPrev()
			return m, nil
		case "down":
			m = m.focusNext()
			return m, nil
		case "enter":
			if m.atLastField() {
				var cmd tea.Cmd
				m, cmd = m.submit()
				return m, cmd
			}
			m = m.focusNext()
			return m, nil
		}

	case loginSuccessMsg:
		m.loading = false
		m.err = nil
		// Если для аккаунта уже настроено шифрование — просим master passphrase на unlock.
		// Если нет (первый логин на новом устройстве без локального кеша) — настраиваем его.
		if m.container.Auth.EncryptionConfigured() {
			return m, func() tea.Msg { return switchScreenMsg{screen: screenLock} }
		}
		return m, func() tea.Msg { return switchScreenMsg{screen: screenSetupEncryption} }

	case registerSuccessMsg:
		m.loading = false
		m.err = nil
		// После регистрации всегда нужно настроить шифрование (аккаунт только что создан).
		return m, func() tea.Msg { return switchScreenMsg{screen: screenSetupEncryption} }

	case loginErrMsg:
		m.loading = false
		m.err = msg.err
		return m, nil
	}

	// Обновляем только активный input.
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
}

func (m loginModel) focusNext() loginModel {
	max := m.maxFields() - 1
	m.inputs[m.focus].Blur()
	m.focus++
	if m.focus > max {
		m.focus = max
	}
	m.inputs[m.focus].Focus()
	return m
}

func (m loginModel) focusPrev() loginModel {
	m.inputs[m.focus].Blur()
	m.focus--
	if m.focus < 0 {
		m.focus = 0
	}
	m.inputs[m.focus].Focus()
	return m
}

func (m loginModel) maxFields() int {
	return 2
}

func (m loginModel) atLastField() bool {
	return m.focus == m.maxFields()-1
}

func (m loginModel) submit() (loginModel, tea.Cmd) {
	login := m.inputs[0].Value()
	password := m.inputs[1].Value()
	if login == "" || password == "" {
		m.err = fmt.Errorf("%s", m.container.Localizer.T("err_empty_login"))
		return m, nil
	}
	m.loading = true
	m.err = nil

	container := m.container
	mode := m.mode

	return m, func() tea.Msg {
		ctx := context.Background()
		switch mode {
		case loginModeRegister:
			if err := container.Auth.Register(ctx, login, []byte(password)); err != nil {
				return loginErrMsg{err: err}
			}
			return registerSuccessMsg{}
		default:
			if err := container.Auth.Login(ctx, login, []byte(password)); err != nil {
				return loginErrMsg{err: err}
			}
			return loginSuccessMsg{}
		}
	}
}

func (m loginModel) view(width, height int) string {
	var b strings.Builder

	l := m.container.Localizer
	title := l.T("tui_login_title")
	if m.mode == loginModeRegister {
		title = l.T("tui_register_title")
	}
	b.WriteString(styles.Title.Render("🔐 GophKeeper — " + title))
	b.WriteString("\n\n")

	labels := []string{l.T("tui_field_username") + ":", l.T("tui_field_password") + ":"}

	for i := 0; i < m.maxFields(); i++ {
		b.WriteString(styles.InputLabel.Render(labels[i]))
		b.WriteString("\n")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	if m.err != nil {
		b.WriteString(styles.ErrorText.Render(fmt.Sprintf("✗ %v", m.err)))
		b.WriteString("\n\n")
	}

	if m.loading {
		b.WriteString(l.T("tui_connecting") + "\n\n")
	}

	b.WriteString(styles.HelpText.Render(l.T("tui_help_login")))

	return styles.Centered(width).Render(b.String())
}
