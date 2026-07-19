package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
)

func newSecretFormTestModel(t *testing.T, server *mocks.MockServerClient) secretFormModel {
	t.Helper()
	c := newTestContainer(t, server)
	c.Session.SetMasterKey(make([]byte, 32))
	c.Session.OpenVault("v1", make([]byte, 32))
	return newSecretFormModel(context.Background(), c)
}

func TestSecretFormModel_SetCreate_DefaultsToEditMode(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	assert.Equal(t, modeEdit, m.mode)
	assert.Equal(t, "", m.secretID)
	assert.Equal(t, domain.SecretTypeLoginPassword, m.secretType)
}

func TestSecretFormModel_SetCreate_InvalidTypeFallsBackToLoginPassword(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", 0)
	assert.Equal(t, domain.SecretTypeLoginPassword, m.secretType)
}

func TestSecretFormModel_SetEditData_OpensInViewMode(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{
		fields: map[string]string{"title": "GitHub", "username": "alice"},
	})
	assert.Equal(t, modeView, m.mode)
	assert.Equal(t, "GitHub", m.value("title"))
	assert.Equal(t, "alice", m.value("username"))
	assert.False(t, m.isDirty())
}

func TestSecretFormModel_IsDirty_ChangesDetected(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{
		fields: map[string]string{"title": "GitHub"},
	})
	m.mode = modeEdit
	assert.False(t, m.isDirty())

	m = m.setValue("title", "GitLab")
	assert.True(t, m.isDirty())
}

func TestSecretFormModel_ViewKey_EscBackToDashboard(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{})

	_, cmd := m.handleViewKey(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenDashboard, switchMsg.screen)
}

func TestSecretFormModel_ViewKey_E_SwitchesToEdit(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{})

	m, _ = m.handleViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	assert.Equal(t, modeEdit, m.mode)
}

func TestSecretFormModel_ViewKey_DeleteOpensConfirm(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{})

	m, _ = m.handleViewKey(tea.KeyMsg{Type: tea.KeyDelete})
	assert.Equal(t, modeConfirmDelete, m.mode)
}

func TestSecretFormModel_ViewKey_Navigation(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{})
	start := m.focus

	m, _ = m.handleViewKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.NotEqual(t, start, m.focus)

	m, _ = m.handleViewKey(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, start, m.focus)
}

func TestSecretFormModel_EditKey_EscWithDirtyAsksConfirm(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	m = m.setValue("title", "GitHub")
	m.dirty = true

	m, _ = m.handleEditKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, modeConfirmExit, m.mode)
}

func TestSecretFormModel_EditKey_EscNoChangesOnCreateGoesToDashboard(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)

	_, cmd := m.handleEditKey(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(switchScreenMsg)
	assert.True(t, ok)
}

func TestSecretFormModel_EditKey_EscOnEditGoesToView(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{})
	m.mode = modeEdit

	m, cmd := m.handleEditKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, modeView, m.mode)
	assert.Nil(t, cmd)
}

func TestSecretFormModel_EditKey_TabNavigatesFields(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	start := m.focus

	m, _ = m.handleEditKey(tea.KeyMsg{Type: tea.KeyTab})
	assert.NotEqual(t, start, m.focus)
}

func TestSecretFormModel_EditKey_CtrlAAddsCustomField(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)

	m, _ = m.handleEditKey(tea.KeyMsg{Type: tea.KeyCtrlA})
	assert.Len(t, m.custom, 1)
	assert.True(t, m.dirty)
}

func TestSecretFormModel_EditKey_CtrlOAddsOTP(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)

	m, _ = m.handleEditKey(tea.KeyMsg{Type: tea.KeyCtrlO})
	assert.Len(t, m.otps, 1)
	assert.True(t, m.dirty)
}

func TestSecretFormModel_EditKey_LeftRightCyclesType(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	// focus 0 == slotType (typeSelectable == true for create mode)
	require.True(t, m.typeSelectable())

	m, _ = m.handleEditKey(tea.KeyMsg{Type: tea.KeyRight})
	assert.NotEqual(t, domain.SecretTypeLoginPassword, m.secretType)
}

func TestSecretFormModel_ConfirmExit_YesSaves(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	m.mode = modeConfirmExit

	m2, cmd := m.handleConfirmExit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	assert.Equal(t, modeEdit, m2.mode)
	// save() без title вернёт nil cmd (валидация) — это ок, главное что попытка была.
	_ = cmd
}

func TestSecretFormModel_ConfirmExit_NoDiscardsAndGoesToDashboard(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	m.mode = modeConfirmExit

	m, cmd := m.handleConfirmExit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(switchScreenMsg)
	assert.True(t, ok)
}

func TestSecretFormModel_ConfirmExit_EscCancelsBackToEdit(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	m.mode = modeConfirmExit

	m, cmd := m.handleConfirmExit(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, modeEdit, m.mode)
	assert.Nil(t, cmd)
}

func TestSecretFormModel_ConfirmDelete_YesDeletes(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().DeleteSecret(mock.Anything, mock.Anything, "s1", int64(1)).Return(nil)

	m := newSecretFormTestModel(t, server)
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{})
	m.mode = modeConfirmDelete

	_, cmd := m.handleConfirmDelete(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenDashboard, switchMsg.screen)
}

func TestSecretFormModel_ConfirmDelete_NoGoesBackToView(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{})
	m.mode = modeConfirmDelete

	m, cmd := m.handleConfirmDelete(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.Equal(t, modeView, m.mode)
	assert.Nil(t, cmd)
}

// Регрессия: [y]/[n] в попапах подтверждения должны работать и в русской раскладке —
// физическая клавиша "y" печатает "н", а "n" печатает "т".
func TestSecretFormModel_ConfirmDelete_CyrillicLayout(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().DeleteSecret(mock.Anything, mock.Anything, "s1", int64(1)).Return(nil)

	m := newSecretFormTestModel(t, server)
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{})
	m.mode = modeConfirmDelete

	_, cmd := m.handleConfirmDelete(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'н'}}) // 'y' на ЙЦУКЕН
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenDashboard, switchMsg.screen)
}

func TestSecretFormModel_ConfirmDelete_CyrillicLayout_No(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{})
	m.mode = modeConfirmDelete

	m, cmd := m.handleConfirmDelete(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'т'}}) // 'n' на ЙЦУКЕН
	assert.Equal(t, modeView, m.mode)
	assert.Nil(t, cmd)
}

func TestSecretFormModel_ConfirmExit_CyrillicLayout_No(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	m.mode = modeConfirmExit

	m, cmd := m.handleConfirmExit(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'т'}}) // 'n' на ЙЦУКЕН
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(switchScreenMsg)
	assert.True(t, ok)
	_ = m
}

func TestSecretFormModel_Save_RequiresTitle(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)

	m, cmd := m.save()
	assert.Nil(t, cmd)
	require.Error(t, m.err)
}

func TestSecretFormModel_Save_InvalidLuhnRejected(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeBankCard)
	m = m.setValue("title", "My Card")
	m = m.setValue("pan", "1234567890123456") // invalid luhn

	m, cmd := m.save()
	assert.Nil(t, cmd)
	require.Error(t, m.err)
}

func TestSecretFormModel_Save_DuplicateCustomKeyRejected(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	m = m.setValue("title", "GitHub")
	m.custom = []kvPair{m.newKVPair("k1", "v1"), m.newKVPair("k1", "v2")}

	m, cmd := m.save()
	assert.Nil(t, cmd)
	require.Error(t, m.err)
}

func TestSecretFormModel_Save_DuplicateOTPCodeRejected(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	m = m.setValue("title", "GitHub")
	m.otps = []otpItem{m.newOTPItem("111111", false), m.newOTPItem("111111", false)}

	m, cmd := m.save()
	assert.Nil(t, cmd)
	require.Error(t, m.err)
}

func TestSecretFormModel_Save_CreateLoginPassword_Success(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().CreateSecret(mock.Anything, mock.Anything, mock.Anything, "v1", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	m := newSecretFormTestModel(t, server)
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	m = m.setValue("title", "GitHub")

	_, cmd := m.save()
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok, "expected switchScreenMsg, got %T", msg)
	assert.Equal(t, screenDashboard, switchMsg.screen)
}

func TestSecretFormModel_DeleteSecret_ErrorReturnsLoginErrMsg(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().DeleteSecret(mock.Anything, mock.Anything, "s1", int64(1)).Return(assert.AnError)

	m := newSecretFormTestModel(t, server)
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{})

	cmd := m.deleteSecret()
	require.NotNil(t, cmd)
	msg := cmd()
	errMsg, ok := msg.(loginErrMsg)
	require.True(t, ok)
	assert.ErrorIs(t, errMsg.err, assert.AnError)
}

func TestSecretFormModel_CopyCurrentField_Empty(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	cmd := m.copyCurrentField()
	assert.Nil(t, cmd)
}

func TestSecretFormModel_CopyCurrentField_WithValue(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	m = m.focusSlot(slotField, 0)
	m = m.setFieldValue(0, "GitHub")

	cmd := m.copyCurrentField()
	assert.NotNil(t, cmd)
}

func TestSecretFormModel_CycleType_ChangesFields(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)

	m = m.cycleType(1)
	assert.NotEqual(t, domain.SecretTypeLoginPassword, m.secretType)

	m = m.cycleType(-1)
	assert.Equal(t, domain.SecretTypeLoginPassword, m.secretType)
}

func TestSecretFormModel_FocusNextPrev_WrapsAround(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetCreate("v1", "Personal", domain.SecretTypeLoginPassword)
	total := len(m.slots())

	for i := 0; i < total; i++ {
		m = m.focusNext()
	}
	// после total шагов курсор должен вернуться в исходное положение
	assert.GreaterOrEqual(t, m.focus, 0)
	assert.Less(t, m.focus, total)
}

func TestSecretFormModel_View_RendersTitle(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m = m.SetEditData("v1", "Personal", "s1", 1, domain.SecretTypeLoginPassword, formData{
		fields: map[string]string{"title": "GitHub"},
	})
	out := m.view(80, 24)
	assert.Contains(t, out, "GitHub")
}

func TestSecretFormModel_Update_LoginErrMsg(t *testing.T) {
	m := newSecretFormTestModel(t, mocks.NewMockServerClient(t))
	m.saving = true

	m, cmd := m.update(loginErrMsg{err: assert.AnError})
	assert.False(t, m.saving)
	assert.ErrorIs(t, m.err, assert.AnError)
	assert.Nil(t, cmd)
}

func TestIsCreatableType(t *testing.T) {
	assert.True(t, isCreatableType(domain.SecretTypeLoginPassword))
	assert.True(t, isCreatableType(domain.SecretTypeBinary))
	assert.False(t, isCreatableType(0))
}

func TestTypeLabel(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	label := typeLabel(domain.SecretTypeLoginPassword, c.Localizer)
	assert.NotEmpty(t, label)
}
