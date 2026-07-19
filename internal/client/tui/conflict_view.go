package tui

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/app"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/tui/styles"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

// conflictModel — единый экран разрешения конфликта версий для всех типов секретов.
type conflictModel struct {
	ctx       context.Context
	container *app.Container

	conflict *secret.GenericConflict

	vaultID   string
	vaultName string

	choice    int // 0=mine, 1=server
	err       error
	resolving bool
}

func newConflictModel(ctx context.Context, container *app.Container) conflictModel {
	return conflictModel{ctx: ctx, container: container}
}

// SetConflict устанавливает активный конфликт для разрешения. nil допустим (экран покажет
// заглушку «нет активного конфликта» — защита от случайного перехода без данных).
func (m conflictModel) SetConflict(c *secret.GenericConflict, vaultID, vaultName string) conflictModel {
	m.conflict = c
	m.vaultID = vaultID
	m.vaultName = vaultName
	m.choice = 0
	m.err = nil
	m.resolving = false
	return m
}

func (m conflictModel) Init() tea.Cmd { return nil }

func (m conflictModel) update(msg tea.Msg) (conflictModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			m.choice = 0
		case "right", "l":
			m.choice = 1
		case "enter":
			var cmd tea.Cmd
			m, cmd = m.resolve()
			return m, cmd
		case "esc":
			return m, func() tea.Msg {
				return switchScreenMsg{screen: screenDashboard, vaultID: m.vaultID, vaultName: m.vaultName}
			}
		}

	case conflictResolvedMsg:
		m.resolving = false
		return m, func() tea.Msg {
			return switchScreenMsg{screen: screenDashboard, vaultID: m.vaultID, vaultName: m.vaultName}
		}

	case conflictMsg:
		// GenericResolveConflict вернул новый конфликт (повторная гонка версий) — остаёмся
		// на экране, чтобы пользователь мог разрешить и его.
		m.resolving = false
		m = m.SetConflict(msg.conflict, m.vaultID, m.vaultName)
		return m, nil

	case conflictErrMsg:
		m.resolving = false
		m.err = msg.err
	}
	return m, nil
}

func (m conflictModel) resolve() (conflictModel, tea.Cmd) {
	if m.conflict == nil {
		return m, nil
	}
	m.resolving = true
	container := m.container
	ctx := m.ctx
	conflict := m.conflict
	choice := secret.ChoiceMine
	if m.choice == 1 {
		choice = secret.ChoiceServer
	}
	return m, func() tea.Msg {
		newConflict, err := container.Secret.GenericResolveConflict(ctx, conflict, choice)
		if err != nil {
			return conflictErrMsg{err: err}
		}
		if newConflict != nil {
			// Повторная гонка версий (кто-то успел записать снова) — показываем новый конфликт.
			return conflictMsg{conflict: newConflict}
		}
		return conflictResolvedMsg{}
	}
}

func (m conflictModel) view(width, height int) string {
	var b strings.Builder

	l := m.container.Localizer
	b.WriteString(styles.Title.Render("⚠️  " + l.T("tui_conflict_title")))
	b.WriteString("\n\n")

	if m.conflict == nil {
		b.WriteString(styles.Subtitle.Render(l.T("tui_conflict_none")))
		b.WriteString("\n")
		b.WriteString(styles.HelpText.Render(l.T("tui_help_esc_back")))
		return b.String()
	}

	c := m.conflict
	mineFields := mergeFieldMaps(c.MineRow, c.MineIndex, c.MinePayload)
	serverFields := mergeFieldMaps(c.ServerRow, c.ServerIndex, c.ServerPayload)
	diffKeys := diffingKeys(mineFields, serverFields, c.IsDelete)

	mineCard := renderGenericCard(l, l.T("tui_conflict_mine"), mineFields, diffKeys, c.IsDelete, m.choice == 0)
	serverCard := renderGenericCard(l, l.T("tui_conflict_server"), serverFields, diffKeys, false, m.choice == 1)

	cards := lipgloss.JoinHorizontal(lipgloss.Top, mineCard, "  ", serverCard)
	b.WriteString(cards)
	b.WriteString("\n\n")

	if !c.IsDelete && len(diffKeys) == 0 {
		b.WriteString(styles.HelpText.Render(l.T("tui_conflict_no_diff")))
		b.WriteString("\n\n")
	}

	if m.err != nil {
		b.WriteString(styles.ErrorText.Render(fmt.Sprintf(l.T("tui_error_prefix"), m.err)))
		b.WriteString("\n\n")
	}

	if m.resolving {
		b.WriteString(l.T("tui_resolving") + "\n\n")
	}

	b.WriteString(styles.HelpText.Render(l.T("tui_help_conflict")))
	return b.String()
}

// mergeFieldMaps объединяет row/index/payload одной версии секрета в одну карту "ключ: значение"
// (тиры внутри секрета не пересекаются по именам полей, так что склеивать их безопасно).
// Служебное поле схемы "v" отфильтровывается — оно не несёт пользовательской информации.
func mergeFieldMaps(maps ...map[string]any) map[string]any {
	merged := make(map[string]any)
	for _, m := range maps {
		for k, v := range m {
			if k == "v" {
				continue
			}
			merged[k] = v
		}
	}
	return merged
}

// diffingKeys возвращает отсортированный список ключей, значения которых различаются между
// "моей" и серверной версией (включая ключи, присутствующие только в одной из карт — например
// пустое/nil значение считается отсутствием поля). Конфликт удаления не имеет "моих" полей —
// в этом случае возвращает nil (экран покажет только conflict_delete_intent).
func diffingKeys(mine, server map[string]any, isDelete bool) []string {
	if isDelete {
		return nil
	}
	seen := make(map[string]bool)
	var keys []string
	for k := range mine {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for k := range server {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}

	var diff []string
	for _, k := range keys {
		if !fieldsEqual(mine[k], server[k]) {
			diff = append(diff, k)
		}
	}
	sort.Strings(diff)
	return diff
}

// fieldsEqual сравнивает два значения поля, считая nil и "пустое" (0/""/[]/{}) эквивалентными —
// значения приходят из json.Unmarshal в map[string]any, поэтому числа — float64, а срезы/карты
// могут быть nil или пустыми в зависимости от того, был ли исходный слайс nil или []T{}.
func fieldsEqual(a, b any) bool {
	if isEmptyFieldValue(a) && isEmptyFieldValue(b) {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func isEmptyFieldValue(v any) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case string:
		return t == ""
	case []any:
		return len(t) == 0
	case map[string]any:
		return len(t) == 0
	}
	return false
}

// renderGenericCard рендерит карточку секрета (любого типа) для показа в экране разрешения
// конфликта, ограничиваясь только ключами из diffKeys (различающиеся поля).
// Порядок полей одинаковый (diffKeys уже отсортирован) на обеих карточках, чтобы строки
// визуально совпадали построчно между колонками.
func renderGenericCard(l localizerT, title string, fields map[string]any, diffKeys []string, isDelete, selected bool) string {
	style := styles.CardBox
	if selected {
		style = styles.SelectedCard
	}

	var content strings.Builder
	content.WriteString(styles.InputLabel.Render(title))
	content.WriteString("\n\n")
	if isDelete {
		content.WriteString(l.T("conflict_delete_intent") + "\n")
	} else {
		for _, k := range diffKeys {
			v := fields[k]
			if isEmptyFieldValue(v) {
				_, _ = fmt.Fprintf(&content, "%s: —\n", k)
				continue
			}
			content.WriteString(formatFieldEntry(l, k, v))
		}
	}
	return style.Render(content.String())
}

// formatFieldEntry рендерит одну пару "ключ: значение" карточки конфликта. custom_fields и
// otp_codes хранятся как []map[string]any (сериализованные KeyValue/OTPCode) — печатать их
// через "%v" даёт нечитаемый Go-дамп (map[key:x value:y]); здесь они разбираются в
// человекочитаемый построчный вид. Остальные поля печатаются как есть.
func formatFieldEntry(l localizerT, key string, v any) string {
	switch key {
	case "custom_fields":
		return formatListEntry(l.T("tui_conflict_custom_fields")+":", v, func(item map[string]any) string {
			return fmt.Sprintf("%v = %v", item["key"], item["value"])
		})
	case "otp_codes":
		return formatListEntry(l.T("tui_conflict_otp_codes")+":", v, func(item map[string]any) string {
			box := "[ ]"
			if used, _ := item["used"].(bool); used {
				box = "[x]"
			}
			return fmt.Sprintf("%s %v", box, item["code"])
		})
	default:
		return fmt.Sprintf("%s: %v\n", key, v)
	}
}

// formatListEntry печатает заголовок + одну строку на элемент списка (каждый элемент —
// map[string]any, отформатированная через itemFmt). Непустой список печатается с отступом,
// чтобы визуально отделяться от простых скалярных полей выше.
func formatListEntry(header string, v any, itemFmt func(map[string]any) string) string {
	items, ok := v.([]any)
	if !ok || len(items) == 0 {
		return header + " —\n"
	}
	var b strings.Builder
	b.WriteString(header + "\n")
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		b.WriteString("  " + itemFmt(item) + "\n")
	}
	return b.String()
}
