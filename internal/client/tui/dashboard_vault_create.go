package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
)

type vaultCreateModel struct {
	input textinput.Model
}

func newVaultCreateModel() vaultCreateModel {
	ti := textinput.New()
	ti.CharLimit = 64
	ti.Width = 40
	ti.Focus()
	return vaultCreateModel{input: ti}
}

func (m vaultCreateModel) value() string {
	return strings.TrimSpace(m.input.Value())
}

func (m vaultCreateModel) update(msg tea.Msg) (vaultCreateModel, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m vaultCreateModel) view(l localizerT) string {
	var b strings.Builder
	b.WriteString(styles.InputLabel.Render(l.T("tui_new_vault") + ":"))
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(styles.HelpText.Render(l.T("tui_help_vault_create")))
	return b.String()
}
