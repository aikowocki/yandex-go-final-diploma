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

func newLoginTestModel(t *testing.T, server contracts.ServerClient) loginModel {
	t.Helper()
	c := newTestContainer(t, server)
	return newLoginModel(c)
}

func TestLoginModel_TabTogglesMode(t *testing.T) {
	m := newLoginTestModel(t, mocks.NewMockServerClient(t))
	require.Equal(t, loginModeLogin, m.mode)

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, loginModeRegister, m.mode)

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, loginModeLogin, m.mode)
}

func TestLoginModel_EnterOnEmptyFieldsSetsError(t *testing.T) {
	m := newLoginTestModel(t, mocks.NewMockServerClient(t))

	// Оба поля пусты — фокус уже на первом поле (login).
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})
	// enter на непоследнем поле просто двигает фокус вперёд, ошибку не ставит.
	assert.Nil(t, m.err)
	assert.Equal(t, 1, m.focus)

	// На последнем поле enter вызывает submit(), который должен вернуть ошибку
	// из-за пустых полей.
	m, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	require.Error(t, m.err)
}

func TestLoginModel_SubmitLoginSuccess(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		Login(mock.Anything, "alice", []byte("secret")).
		Return(contracts.LoginResult{}, nil).
		Once()
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, assert.AnError).Maybe()

	m := newLoginTestModel(t, server)
	m.inputs[0].SetValue("alice")
	m.inputs[1].SetValue("secret")
	m.focus = 1 // последнее поле

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	assert.Equal(t, loginSuccessMsg{}, msg)
}

func TestLoginModel_SubmitLoginFailure(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		Login(mock.Anything, "alice", []byte("wrong")).
		Return(contracts.LoginResult{}, assert.AnError).
		Once()

	m := newLoginTestModel(t, server)
	m.inputs[0].SetValue("alice")
	m.inputs[1].SetValue("wrong")
	m.focus = 1

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	errMsg, ok := msg.(loginErrMsg)
	require.True(t, ok)
	assert.ErrorIs(t, errMsg.err, assert.AnError)
}

func TestLoginModel_SubmitRegisterSuccess(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		Register(mock.Anything, "bob", []byte("pw")).
		Return(contracts.Tokens{}, nil).
		Once()

	m := newLoginTestModel(t, server)
	m.mode = loginModeRegister
	m.inputs[0].SetValue("bob")
	m.inputs[1].SetValue("pw")
	m.focus = 1

	_, cmd := m.update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	assert.Equal(t, registerSuccessMsg{}, msg)
}

func TestLoginModel_LoginSuccessMsgRoutesToLockWhenConfigured(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		Login(mock.Anything, "alice", []byte("secret")).
		Return(contracts.LoginResult{
			Tokens:       contracts.Tokens{UserID: "u1"},
			EncKDFSalt:   []byte("salt"),
			EncKDFParams: []byte(`{"version":1,"memory":8192,"iterations":1,"parallelism":1,"keyLen":32}`),
			EncMasterKey: []byte("wrapped-master-key"),
		}, nil).
		Once()
	server.EXPECT().ListVaults(mock.Anything, mock.Anything).Return(nil, assert.AnError).Maybe()

	m := newLoginTestModel(t, server)
	// Реальный Login usecase-вызов настраивает encKDFSalt/Params/MasterKey из ответа сервера,
	// после чего EncryptionConfigured() == true.
	require.NoError(t, m.container.Auth.Login(t.Context(), "alice", []byte("secret")))

	_, cmd := m.update(loginSuccessMsg{})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenLock, switchMsg.screen)
}

func TestLoginModel_LoginSuccessMsgRoutesToSetupWhenNotConfigured(t *testing.T) {
	m := newLoginTestModel(t, mocks.NewMockServerClient(t))

	_, cmd := m.update(loginSuccessMsg{})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenSetupEncryption, switchMsg.screen)
}

func TestLoginModel_RegisterSuccessMsgRoutesToSetup(t *testing.T) {
	m := newLoginTestModel(t, mocks.NewMockServerClient(t))

	_, cmd := m.update(registerSuccessMsg{})
	require.NotNil(t, cmd)
	msg := cmd()
	switchMsg, ok := msg.(switchScreenMsg)
	require.True(t, ok)
	assert.Equal(t, screenSetupEncryption, switchMsg.screen)
}

func TestLoginModel_LoginErrMsgSetsErrAndStopsLoading(t *testing.T) {
	m := newLoginTestModel(t, mocks.NewMockServerClient(t))
	m.loading = true

	m, cmd := m.update(loginErrMsg{err: assert.AnError})
	assert.Nil(t, cmd)
	assert.False(t, m.loading)
	assert.ErrorIs(t, m.err, assert.AnError)
}

func TestLoginModel_FocusNavigation(t *testing.T) {
	m := newLoginTestModel(t, mocks.NewMockServerClient(t))
	require.Equal(t, 0, m.focus)

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.focus)

	// Не выходит за границы.
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.focus)

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyShiftTab})
	assert.Equal(t, 0, m.focus)

	m, _ = m.update(tea.KeyMsg{Type: tea.KeyShiftTab})
	assert.Equal(t, 0, m.focus)
}

func TestLoginModel_ViewRendersTitle(t *testing.T) {
	m := newLoginTestModel(t, mocks.NewMockServerClient(t))
	out := m.view(80, 24)
	assert.Contains(t, out, "GophKeeper")
}
