package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
	vaultuc "github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/vault"
)

type syncScopePopup struct {
	vaults  []vaultuc.DecryptedVault
	checked map[string]bool
	cursor  int
}

func newSyncScopePopup(vaults []vaultuc.DecryptedVault) syncScopePopup {
	checked := make(map[string]bool, len(vaults))
	for _, v := range vaults {
		checked[v.ID] = true // по умолчанию всё включено
	}
	return syncScopePopup{vaults: vaults, checked: checked}
}

func (m syncScopePopup) update(msg tea.KeyMsg) syncScopePopup {
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.vaults)-1 {
			m.cursor++
		}
	case tea.KeySpace:
		if m.cursor < len(m.vaults) {
			id := m.vaults[m.cursor].ID
			m.checked[id] = !m.checked[id]
		}
	}
	return m
}

// confirm возвращает команду, которая применяет выбор (отключает sync для снятых папок)
// и отмечает, что выбор сделан.
func (m syncScopePopup) confirm(ctx context.Context, container *app.Container) tea.Cmd {
	checked := m.checked
	vaults := m.vaults
	return func() tea.Msg {
		for _, v := range vaults {
			if !checked[v.ID] {
				if err := container.Sync.SetVaultSyncEnabled(ctx, v.ID, false); err != nil {
					return vaultsErrMsg{err: err}
				}
			}
		}
		if err := container.Sync.MarkSyncScopeChosen(ctx); err != nil {
			return vaultsErrMsg{err: err}
		}
		return syncScopeConfirmedMsg{}
	}
}

func (m syncScopePopup) view(l localizerT) string {
	var b strings.Builder
	b.WriteString(styles.Title.Render(l.T("tui_syncscope_title")))
	b.WriteString("\n\n")
	b.WriteString(styles.Subtitle.Render(l.T("tui_syncscope_hint")))
	b.WriteString("\n\n")

	for i, v := range m.vaults {
		box := "[ ]"
		if m.checked[v.ID] {
			box = "[x]"
		}
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}
		line := cursor + box + " " + v.Name
		if i == m.cursor {
			b.WriteString(styles.InputLabel.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpText.Render(l.T("tui_help_syncscope")))
	return b.String()
}
