package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToastModel_ShowAndExpire(t *testing.T) {
	m := newToastModel()
	assert.Empty(t, m.view())

	m, cmd := m.update(toastMsg{text: "hello"})
	assert.NotNil(t, cmd)
	assert.True(t, m.visible)
	assert.Equal(t, "hello", m.text)
	assert.Contains(t, m.view(), "hello")

	m, cmd = m.update(toastExpiredMsg{})
	assert.Nil(t, cmd)
	assert.False(t, m.visible)
	assert.Empty(t, m.view())
}

func TestShowToast(t *testing.T) {
	cmd := showToast("hi")
	require := assert.New(t)
	msg := cmd()
	tm, ok := msg.(toastMsg)
	require.True(ok)
	require.Equal("hi", tm.text)
}
