package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
)

func newSetupEncryptionTestModel(t *testing.T, server contracts.ServerClient) setupEncryptionModel {
	t.Helper()
	c := newTestContainer(t, server)
	return newSetupEncryptionModel(c)
}

func TestSetupEncryptionModel_TabTogglesFocus(t *testing.T) {
	m := newSetupEncryptionTestModel(t, mocks.NewMockServerClient(t))
	require.Equal(t, 0, m.focus)

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 1, m.focus)

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 0, m.focus)
}

func TestSetupEncryptionModel_SubmitEmptyPassphrase(t *testing.T) {
	m := newSetupEncryptionTestModel(t, mocks.NewMockServerClient(t))
	m, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	require.Error(t, m.err)
}

func TestSetupEncryptionModel_SubmitMismatchedPassphrase(t *testing.T) {
	m := newSetupEncryptionTestModel(t, mocks.NewMockServerClient(t))
	m.input.SetValue("pw1")
	m.confirm.SetValue("pw2")

	m, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	require.Error(t, m.err)
}

func TestSetupEncryptionModel_SubmitSuccessGeneratesRecoveryCodes(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		SetupEncryption(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	server.EXPECT().
		StoreRecoveryCodes(mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	m := newSetupEncryptionTestModel(t, server)
	m.input.SetValue("passphrase123")
	m.confirm.SetValue("passphrase123")

	m, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	assert.True(t, m.saving)

	msg := cmd()
	generatedMsg, ok := msg.(recoveryCodesGeneratedMsg)
	require.True(t, ok)
	assert.Len(t, generatedMsg.codes, 5)
}

func TestSetupEncryptionModel_SubmitSuccessButRecoveryCodesFail(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		SetupEncryption(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	server.EXPECT().
		StoreRecoveryCodes(mock.Anything, mock.Anything, mock.Anything).
		Return(assert.AnError).Once()

	m := newSetupEncryptionTestModel(t, server)
	m.input.SetValue("passphrase123")
	m.confirm.SetValue("passphrase123")

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	assert.Equal(t, unlockSuccessMsg{}, msg)
}

func TestSetupEncryptionModel_SubmitServerError(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		SetupEncryption(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(assert.AnError).Once()

	m := newSetupEncryptionTestModel(t, server)
	m.input.SetValue("passphrase123")
	m.confirm.SetValue("passphrase123")

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	errMsg, ok := msg.(loginErrMsg)
	require.True(t, ok)
	assert.ErrorIs(t, errMsg.err, assert.AnError)
}

func TestSetupEncryptionModel_UnlockSuccessMsgGoesToDashboard(t *testing.T) {
	m := newSetupEncryptionTestModel(t, mocks.NewMockServerClient(t))
	m.saving = true

	m, cmd := m.update(unlockSuccessMsg{})
	require.NotNil(t, cmd)
	assert.False(t, m.saving)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenDashboard, switchMsg.screen)
}

func TestSetupEncryptionModel_LoginErrMsgSetsErr(t *testing.T) {
	m := newSetupEncryptionTestModel(t, mocks.NewMockServerClient(t))
	m.saving = true

	m, cmd := m.update(loginErrMsg{err: assert.AnError})
	assert.Nil(t, cmd)
	assert.False(t, m.saving)
	assert.ErrorIs(t, m.err, assert.AnError)
}

func TestSetupEncryptionModel_ViewRendersTitle(t *testing.T) {
	m := newSetupEncryptionTestModel(t, mocks.NewMockServerClient(t))
	out := m.view(80, 24)
	assert.NotEmpty(t, out)
}
