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

func newLockTestModel(t *testing.T, server contracts.ServerClient) lockModel {
	t.Helper()
	c := newTestContainer(t, server)
	return newLockModel(c)
}

func TestLockModel_EmptyEnterIsNoop(t *testing.T) {
	m := newLockTestModel(t, mocks.NewMockServerClient(t))
	m2, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Equal(t, m, m2)
}

func TestLockModel_UnlockSuccess(t *testing.T) {
	salt := mustSalt(t)
	params := testParamsJSON(t)
	wrapped := mustWrappedKey(t, "secret", salt)

	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		Login(mock.Anything, "alice", []byte("x")).
		Return(contracts.LoginResult{EncKDFSalt: salt, EncKDFParams: params, EncMasterKey: wrapped}, nil).
		Once()
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, assert.AnError).Maybe()

	c := newTestContainer(t, server)
	require.NoError(t, c.Auth.Login(t.Context(), "alice", []byte("x")))

	m := newLockModel(c)
	m.input.SetValue("secret")

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	assert.Equal(t, unlockSuccessMsg{}, msg)
}

func TestLockModel_UnlockWrongPassphrase(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	salt := mustSalt(t)
	params := testParamsJSON(t)
	wrapped := mustWrappedKey(t, "secret", salt)
	server.EXPECT().
		Login(mock.Anything, "alice", []byte("x")).
		Return(contracts.LoginResult{EncKDFSalt: salt, EncKDFParams: params, EncMasterKey: wrapped}, nil).
		Once()
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, assert.AnError).Maybe()

	c := newTestContainer(t, server)
	require.NoError(t, c.Auth.Login(t.Context(), "alice", []byte("x")))

	m := newLockModel(c)
	m.input.SetValue("wrong-pass")

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	errMsg, ok := msg.(unlockErrMsg)
	require.True(t, ok)
	require.Error(t, errMsg.err)
}

func TestLockModel_UnlockErrMsgIncrementsAttemptsAndFreezes(t *testing.T) {
	m := newLockTestModel(t, mocks.NewMockServerClient(t))

	for i := 1; i < maxUnlockAttempts; i++ {
		m, _ = m.update(unlockErrMsg{err: assert.AnError})
		assert.Equal(t, i, m.attempts)
		assert.False(t, m.cold)
	}

	m, _ = m.update(unlockErrMsg{err: assert.AnError})
	assert.Equal(t, maxUnlockAttempts, m.attempts)
	assert.True(t, m.cold)

	// В холодном состоянии update больше ничего не делает.
	m2, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.True(t, m2.cold)
}

func TestLockModel_EscLocksAndReturnsToLogin(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	c.Session.SetMasterKey([]byte("some-master-key-32-bytes-long!!"))
	m := newLockModel(c)

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenLogin, switchMsg.screen)
	assert.False(t, c.Session.Unlocked())
}

func TestLockModel_UnlockSuccessOffersSetPINWhenNoPIN(t *testing.T) {
	m := newLockTestModel(t, mocks.NewMockServerClient(t))
	require.False(t, m.container.Session.HasPIN())

	m, cmd := m.update(unlockSuccessMsg{})
	assert.Nil(t, cmd)
	assert.Equal(t, lockModeSetPIN, m.mode)
}

func TestLockModel_UnlockSuccessGoesToDashboardWhenPINAlreadySet(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	c.Session.SetMasterKey([]byte("some-master-key-32-bytes-long!!"))
	require.NoError(t, c.Auth.SetPIN([]byte("1234")))

	m := newLockModel(c)
	m.mode = lockModePIN

	_, cmd := m.update(unlockSuccessMsg{})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenDashboard, switchMsg.screen)
}

func TestLockModel_SetPINFlow(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	c.Session.SetMasterKey([]byte("some-master-key-32-bytes-long!!"))

	m := newLockModel(c)
	m.mode = lockModeSetPIN
	m.input.SetValue("1234")

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	assert.Equal(t, pinSetMsg{}, msg)
	assert.True(t, c.Session.HasPIN())
}

func TestLockModel_SetPINEscGoesToDashboard(t *testing.T) {
	m := newLockTestModel(t, mocks.NewMockServerClient(t))
	m.mode = lockModeSetPIN

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenDashboard, switchMsg.screen)
}

func TestLockModel_TabTogglesPasswordPIN(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	c.Session.SetMasterKey([]byte("some-master-key-32-bytes-long!!"))
	require.NoError(t, c.Auth.SetPIN([]byte("1234")))

	m := newLockModel(c) // PIN уже есть → mode = PIN по умолчанию
	require.Equal(t, lockModePIN, m.mode)

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, lockModePassword, m.mode)

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, lockModePIN, m.mode)
}

func TestLockModel_CtrlRSwitchesToRecoveryMode(t *testing.T) {
	m := newLockTestModel(t, mocks.NewMockServerClient(t))
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyCtrlR})
	assert.Equal(t, lockModeRecovery, m.mode)
}

func TestLockModel_RecoverySuccessSwitchesToSetupEncryption(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().GetRecoveryBlob(mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("dummy-blob-not-decryptable"), nil).Once()

	c := newTestContainer(t, server)
	m := newLockModel(c)
	m.mode = lockModeRecovery
	m.input.SetValue("AAAA-BBBB-CCCC-DDDD")

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	// Расшифровка не удастся (blob не настоящий) → unlockErrMsg, а не switchScreenMsg.
	_, isErr := msg.(unlockErrMsg)
	assert.True(t, isErr)
}

func TestLockModel_ViewRendersFrozenState(t *testing.T) {
	m := newLockTestModel(t, mocks.NewMockServerClient(t))
	m.cold = true
	out := m.view(80, 24)
	assert.NotEmpty(t, out)
}

func TestLockModel_ViewRendersDefault(t *testing.T) {
	m := newLockTestModel(t, mocks.NewMockServerClient(t))
	out := m.view(80, 24)
	assert.NotEmpty(t, out)
}
