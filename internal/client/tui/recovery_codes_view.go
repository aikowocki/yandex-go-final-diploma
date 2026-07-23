package tui

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
)

type recoveryCodesModel struct {
	container *app.Container
	codes     []string
}

func newRecoveryCodesModel(container *app.Container, codes []string) recoveryCodesModel {
	return recoveryCodesModel{container: container, codes: codes}
}

func (m recoveryCodesModel) update(msg tea.Msg) (recoveryCodesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			return m, func() tea.Msg { return switchScreenMsg{screen: screenDashboard} }
		case "c":
			// Копируем все коды в буфер.
			_ = clipboard.WriteAll(strings.Join(m.codes, "\n"))
			return m, showToast(m.container.Localizer.T("tui_toast_copied"))
		}
	}
	return m, nil
}

func (m recoveryCodesModel) view(width, height int) string {
	var b strings.Builder
	l := m.container.Localizer

	b.WriteString(styles.Title.Render("🔑 " + l.T("tui_recovery_codes_title")))
	b.WriteString("\n\n")
	b.WriteString(styles.ErrorText.Render(l.T("tui_recovery_codes_warning")))
	b.WriteString("\n\n")

	for i, code := range m.codes {
		b.WriteString(styles.InputLabel.Render("  " + string(rune('1'+i)) + ". " + code))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpText.Render(l.T("tui_recovery_codes_hint")))
	b.WriteString("\n\n")
	b.WriteString(styles.HelpText.Render("c: copy all • enter/esc: continue"))

	return styles.Centered(width).Render(b.String())
}
