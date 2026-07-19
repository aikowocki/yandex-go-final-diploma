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

// setupEncryptionModel — экран первичной настройки шифрования (вывод MasterKey из
// passphrase). Показывается после Register или после Login на аккаунте, где шифрование
// ещё не настроено (EncryptionConfigured() == false).
type setupEncryptionModel struct {
	container *app.Container
	input     textinput.Model
	confirm   textinput.Model
	focus     int
	err       error
	saving    bool
}

func newSetupEncryptionModel(container *app.Container) setupEncryptionModel {
	ti := textinput.New()
	ti.Placeholder = "Master passphrase"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.CharLimit = 128
	ti.Width = 40
	ti.Focus()

	confirm := textinput.New()
	confirm.Placeholder = "Confirm passphrase"
	confirm.EchoMode = textinput.EchoPassword
	confirm.EchoCharacter = '•'
	confirm.CharLimit = 128
	confirm.Width = 40

	return setupEncryptionModel{container: container, input: ti, confirm: confirm}
}

func (m setupEncryptionModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m setupEncryptionModel) update(msg tea.Msg) (setupEncryptionModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.focus = 1 - m.focus
			if m.focus == 0 {
				m.confirm.Blur()
				m.input.Focus()
			} else {
				m.input.Blur()
				m.confirm.Focus()
			}
			return m, nil
		case "shift+tab", "up":
			m.focus = 1 - m.focus
			if m.focus == 0 {
				m.confirm.Blur()
				m.input.Focus()
			} else {
				m.input.Blur()
				m.confirm.Focus()
			}
			return m, nil
		case "enter":
			var cmd tea.Cmd
			m, cmd = m.submit()
			return m, cmd
		}

	case unlockSuccessMsg:
		m.saving = false
		return m, func() tea.Msg { return switchScreenMsg{screen: screenDashboard} }

	case recoveryCodesGeneratedMsg:
		m.saving = false
		return m, func() tea.Msg { return switchScreenMsg{screen: screenRecoveryCodes} }

	case loginErrMsg:
		m.saving = false
		m.err = msg.err
		return m, nil
	}

	var cmd tea.Cmd
	if m.focus == 0 {
		m.input, cmd = m.input.Update(msg)
	} else {
		m.confirm, cmd = m.confirm.Update(msg)
	}
	return m, cmd
}

func (m setupEncryptionModel) submit() (setupEncryptionModel, tea.Cmd) {
	pass := m.input.Value()
	confirm := m.confirm.Value()
	if pass == "" {
		m.err = fmt.Errorf("%s", m.container.Localizer.T("tui_err_passphrase_required"))
		return m, nil
	}
	if pass != confirm {
		m.err = fmt.Errorf("%s", m.container.Localizer.T("tui_err_passphrase_mismatch"))
		return m, nil
	}
	m.saving = true
	m.err = nil

	container := m.container
	return m, func() tea.Msg {
		if err := container.Auth.SetupEncryption(context.Background(), []byte(pass)); err != nil {
			return loginErrMsg{err: err}
		}
		// Генерируем recovery codes после успешной настройки шифрования.
		codes, err := container.Auth.GenerateRecoveryCodes(context.Background())
		if err != nil {
			// Шифрование настроено, но коды не сохранились — не блокируем, просто идём дальше.
			return unlockSuccessMsg{}
		}
		return recoveryCodesGeneratedMsg{codes: codes}
	}
}

func (m setupEncryptionModel) view(width, height int) string {
	var b strings.Builder

	l := m.container.Localizer
	b.WriteString(styles.Title.Render("🔐 " + l.T("tui_setup_encryption_title")))
	b.WriteString("\n\n")
	b.WriteString(styles.Subtitle.Render(l.T("tui_setup_encryption_hint")))
	b.WriteString("\n\n")

	b.WriteString(styles.InputLabel.Render(l.T("tui_field_master_passphrase") + ":"))
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	b.WriteString(styles.InputLabel.Render(l.T("tui_field_confirm_passphrase") + ":"))
	b.WriteString("\n")
	b.WriteString(m.confirm.View())
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(styles.ErrorText.Render(fmt.Sprintf("✗ %v", m.err)))
		b.WriteString("\n\n")
	}

	if m.saving {
		b.WriteString(l.T("tui_saving") + "\n\n")
	}

	b.WriteString(styles.HelpText.Render(l.T("tui_help_setup_encryption")))

	return styles.Centered(width).Render(b.String())
}
