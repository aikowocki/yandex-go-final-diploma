package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
)

func newAppTestModel(t *testing.T, startScreen screenID) App {
	t.Helper()
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	return New(context.Background(), c, startScreen)
}

func TestNew_InitializesAllChildModels(t *testing.T) {
	a := newAppTestModel(t, screenLogin)
	assert.Equal(t, screenLogin, a.screen)
	assert.False(t, a.lastActivity.IsZero())
}

func TestApp_Init_Login(t *testing.T) {
	a := newAppTestModel(t, screenLogin)
	cmd := a.Init()
	assert.NotNil(t, cmd)
}

func TestApp_Init_Dashboard(t *testing.T) {
	a := newAppTestModel(t, screenDashboard)
	cmd := a.Init()
	assert.NotNil(t, cmd)
}

func TestApp_Update_CtrlCQuits(t *testing.T) {
	a := newAppTestModel(t, screenLogin)
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.NotNil(t, cmd)
}

func TestApp_Update_WindowSizeMsg(t *testing.T) {
	a := newAppTestModel(t, screenLogin)
	model, _ := a.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	updated := model.(App)
	assert.Equal(t, 100, updated.width)
	assert.Equal(t, 40, updated.height)
}

func TestApp_Update_SwitchScreenMsg_ToLogin(t *testing.T) {
	a := newAppTestModel(t, screenDashboard)
	model, _ := a.Update(switchScreenMsg{screen: screenLogin})
	updated := model.(App)
	assert.Equal(t, screenLogin, updated.screen)
}

func TestApp_Update_SwitchScreenMsg_ToLock(t *testing.T) {
	a := newAppTestModel(t, screenLogin)
	model, _ := a.Update(switchScreenMsg{screen: screenLock})
	updated := model.(App)
	assert.Equal(t, screenLock, updated.screen)
}

func TestApp_Update_SwitchScreenMsg_ToForm_Create(t *testing.T) {
	a := newAppTestModel(t, screenDashboard)
	model, _ := a.Update(switchScreenMsg{screen: screenForm, vaultID: "v1", vaultName: "Personal"})
	updated := model.(App)
	assert.Equal(t, screenForm, updated.screen)
	assert.Equal(t, "v1", updated.form.vaultID)
}

func TestApp_Update_SwitchScreenMsg_ToConflict(t *testing.T) {
	a := newAppTestModel(t, screenDashboard)
	model, _ := a.Update(switchScreenMsg{screen: screenConflict, vaultID: "v1"})
	updated := model.(App)
	assert.Equal(t, screenConflict, updated.screen)
}

func TestApp_Update_RecoveryCodesGeneratedMsg(t *testing.T) {
	a := newAppTestModel(t, screenSetupEncryption)
	model, _ := a.Update(recoveryCodesGeneratedMsg{codes: []string{"AAAA", "BBBB"}})
	updated := model.(App)
	assert.Equal(t, screenRecoveryCodes, updated.screen)
	assert.Equal(t, []string{"AAAA", "BBBB"}, updated.recoveryCodes.codes)
}

func TestApp_Update_AutolockChangedMsg(t *testing.T) {
	a := newAppTestModel(t, screenDashboard)
	model, _ := a.Update(autolockChangedMsg{timeout: 10 * time.Minute})
	updated := model.(App)
	assert.Equal(t, 10*time.Minute, updated.autolockTimeout)
}

func TestApp_Update_AutolockMsg_TriggersLockWhenIdle(t *testing.T) {
	a := newAppTestModel(t, screenDashboard)
	a.autolockTimeout = time.Millisecond
	a.lastActivity = time.Now().Add(-time.Hour)
	a.container.Session.SetMasterKey(make([]byte, 32))

	model, cmd := a.Update(autolockMsg{})
	updated := model.(App)
	assert.Equal(t, screenLock, updated.screen)
	assert.False(t, updated.container.Session.Unlocked())
	assert.NotNil(t, cmd)
}

func TestApp_Update_AutolockMsg_NoTimeoutNoOp(t *testing.T) {
	a := newAppTestModel(t, screenDashboard)
	a.autolockTimeout = 0

	model, _ := a.Update(autolockMsg{})
	updated := model.(App)
	assert.Equal(t, screenDashboard, updated.screen)
}

func TestApp_View_Login(t *testing.T) {
	a := newAppTestModel(t, screenLogin)
	out := a.View()
	assert.NotEmpty(t, out)
}

func TestApp_View_Dashboard(t *testing.T) {
	a := newAppTestModel(t, screenDashboard)
	out := a.View()
	assert.NotEmpty(t, out)
}

func TestAutolockTimeoutFromConfig(t *testing.T) {
	assert.Equal(t, time.Duration(0), autolockTimeoutFromConfig(0))
	assert.Equal(t, time.Duration(0), autolockTimeoutFromConfig(-5))
	assert.Equal(t, 5*time.Minute, autolockTimeoutFromConfig(5))
}

func TestStartScreen_Unlocked(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	c.Session.SetMasterKey(make([]byte, 32))
	assert.Equal(t, screenDashboard, StartScreen(c))
}

func TestStartScreen_NoEncryptionConfigured(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	assert.Equal(t, screenLogin, StartScreen(c))
}
