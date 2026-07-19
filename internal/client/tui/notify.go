package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
)

const toastDuration = 3 * time.Second

type toastModel struct {
	text    string
	visible bool
}

func newToastModel() toastModel {
	return toastModel{}
}

func (m toastModel) update(msg tea.Msg) (toastModel, tea.Cmd) {
	switch msg := msg.(type) {
	case toastMsg:
		m.text = msg.text
		m.visible = true
		return m, tea.Tick(toastDuration, func(time.Time) tea.Msg {
			return toastExpiredMsg{}
		})
	case toastExpiredMsg:
		m.visible = false
		m.text = ""
	}
	return m, nil
}

func (m toastModel) view() string {
	if !m.visible {
		return ""
	}
	return styles.Toast.Render(m.text)
}

// showToast — вспомогательная команда для отправки тоста.
func showToast(text string) tea.Cmd {
	return func() tea.Msg { return toastMsg{text: text} }
}
