// dashboard_user.go — попап «Юзер [U]»: показывает текущий аккаунт (userID из локального
// кеша) и пункт логаута.
package tui

import (
	"context"
	"strings"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
)

type userMenuPopup struct{}

func (m userMenuPopup) view(ctx context.Context, container *app.Container, l localizerT) string {
	var b strings.Builder
	b.WriteString(styles.Title.Render(l.T("tui_user_menu_title")))
	b.WriteString("\n\n")

	login := container.Auth.CurrentLogin(ctx)
	if login == "" {
		login = "—"
	}
	userID := container.Auth.CurrentUserID(ctx)
	if userID == "" {
		userID = "—"
	}
	b.WriteString(l.T("label_username") + ": " + login + "\n")
	b.WriteString(l.T("label_id") + ": " + userID + "\n\n")
	b.WriteString(styles.InputLabel.Render(l.T("tui_user_menu_logout")))
	b.WriteString("\n")
	b.WriteString(styles.HelpText.Render(l.T("tui_help_esc_back")))
	return b.String()
}
