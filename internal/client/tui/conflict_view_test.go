package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

func newConflictTestModel(t *testing.T, server *mocks.MockServerClient) conflictModel {
	t.Helper()
	c := newTestContainer(t, server)
	return newConflictModel(context.Background(), c)
}

func TestConflictModel_SetConflict(t *testing.T) {
	m := newConflictTestModel(t, mocks.NewMockServerClient(t))
	conflict := &secret.GenericConflict{SecretID: "s1", VaultID: "v1"}
	m = m.SetConflict(conflict, "v1", "Personal")
	assert.Equal(t, conflict, m.conflict)
	assert.Equal(t, "v1", m.vaultID)
	assert.Equal(t, 0, m.choice)
}

func TestConflictModel_UpdateKey_LeftRight(t *testing.T) {
	m := newConflictTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetConflict(&secret.GenericConflict{}, "v1", "Personal")

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.choice)

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 0, m.choice)
}

func TestConflictModel_UpdateKey_Esc(t *testing.T) {
	m := newConflictTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetConflict(&secret.GenericConflict{}, "v1", "Personal")

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenDashboard, switchMsg.screen)
}

func TestConflictModel_ConflictResolvedMsg(t *testing.T) {
	m := newConflictTestModel(t, mocks.NewMockServerClient(t))
	m.resolving = true

	m, cmd := m.update(conflictResolvedMsg{})
	assert.False(t, m.resolving)
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(switchScreenMsg)
	assert.True(t, ok)
}

func TestConflictModel_ConflictErrMsg(t *testing.T) {
	m := newConflictTestModel(t, mocks.NewMockServerClient(t))
	m.resolving = true

	m, _ = m.update(conflictErrMsg{err: assert.AnError})
	assert.False(t, m.resolving)
	assert.ErrorIs(t, m.err, assert.AnError)
}

func TestConflictModel_ConflictMsg_ShowsNewConflict(t *testing.T) {
	m := newConflictTestModel(t, mocks.NewMockServerClient(t))
	m.resolving = true
	newConflict := &secret.GenericConflict{SecretID: "s2"}

	m, _ = m.update(conflictMsg{conflict: newConflict})
	assert.False(t, m.resolving)
	assert.Equal(t, newConflict, m.conflict)
}

func TestConflictModel_Resolve_NilConflict(t *testing.T) {
	m := newConflictTestModel(t, mocks.NewMockServerClient(t))
	_, cmd := m.resolve()
	assert.Nil(t, cmd)
}

func TestConflictModel_View_NoConflict(t *testing.T) {
	m := newConflictTestModel(t, mocks.NewMockServerClient(t))
	out := m.view(80, 24)
	assert.NotEmpty(t, out)
}

func TestConflictModel_View_WithConflict(t *testing.T) {
	m := newConflictTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetConflict(&secret.GenericConflict{
		MineRow:   map[string]any{"title": "Mine"},
		ServerRow: map[string]any{"title": "Server"},
	}, "v1", "Personal")

	out := m.view(80, 24)
	assert.Contains(t, out, "Mine")
	assert.Contains(t, out, "Server")
}

func TestMergeFieldMaps(t *testing.T) {
	merged := mergeFieldMaps(
		map[string]any{"a": 1, "v": 1},
		map[string]any{"b": 2},
	)
	assert.Equal(t, 1, merged["a"])
	assert.Equal(t, 2, merged["b"])
	_, hasV := merged["v"]
	assert.False(t, hasV, "поле схемы v должно отфильтровываться")
}

func TestDiffingKeys_IsDelete(t *testing.T) {
	keys := diffingKeys(map[string]any{"a": 1}, map[string]any{"a": 2}, true)
	assert.Nil(t, keys)
}

func TestDiffingKeys_FindsDifferences(t *testing.T) {
	keys := diffingKeys(
		map[string]any{"title": "A", "same": "x"},
		map[string]any{"title": "B", "same": "x"},
		false,
	)
	assert.Equal(t, []string{"title"}, keys)
}

func TestDiffingKeys_EmptyValuesEquivalent(t *testing.T) {
	keys := diffingKeys(
		map[string]any{"note": ""},
		map[string]any{}, // отсутствует = пусто
		false,
	)
	assert.Empty(t, keys)
}

func TestFieldsEqual(t *testing.T) {
	assert.True(t, fieldsEqual(nil, ""))
	assert.True(t, fieldsEqual("", nil))
	assert.True(t, fieldsEqual([]any{}, nil))
	assert.False(t, fieldsEqual("a", "b"))
	assert.True(t, fieldsEqual("a", "a"))
}

func TestIsEmptyFieldValue(t *testing.T) {
	assert.True(t, isEmptyFieldValue(nil))
	assert.True(t, isEmptyFieldValue(""))
	assert.True(t, isEmptyFieldValue([]any{}))
	assert.True(t, isEmptyFieldValue(map[string]any{}))
	assert.False(t, isEmptyFieldValue("x"))
	assert.False(t, isEmptyFieldValue(42))
}

func TestFormatFieldEntry_CustomFields(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	got := formatFieldEntry(c.Localizer, "custom_fields", []any{
		map[string]any{"key": "k1", "value": "v1"},
	})
	assert.Contains(t, got, "k1")
	assert.Contains(t, got, "v1")
}

func TestFormatFieldEntry_OTPCodes(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	got := formatFieldEntry(c.Localizer, "otp_codes", []any{
		map[string]any{"code": "111111", "used": true},
	})
	assert.Contains(t, got, "111111")
	assert.Contains(t, got, "[x]")
}

func TestFormatFieldEntry_Default(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	got := formatFieldEntry(c.Localizer, "title", "GitHub")
	assert.Contains(t, got, "GitHub")
}

func TestFormatListEntry_EmptyList(t *testing.T) {
	got := formatListEntry("Header:", []any{}, func(m map[string]any) string { return "" })
	assert.Contains(t, got, "—")
}

func TestFormatListEntry_NotAList(t *testing.T) {
	got := formatListEntry("Header:", "not-a-list", func(m map[string]any) string { return "" })
	assert.Contains(t, got, "—")
}

func TestRenderGenericCard_IsDelete(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	out := renderGenericCard(c.Localizer, "Mine", nil, nil, true, false)
	assert.NotEmpty(t, out)
}

func TestRenderGenericCard_WithFields(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	out := renderGenericCard(c.Localizer, "Mine", map[string]any{"title": "GitHub"}, []string{"title"}, false, true)
	assert.Contains(t, out, "GitHub")
}
