package session_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
)

func TestSession_MasterKey(t *testing.T) {
	s := session.New()

	_, ok := s.MasterKey()
	assert.False(t, ok)
	assert.False(t, s.Unlocked())

	s.SetMasterKey([]byte("master"))
	mk, ok := s.MasterKey()
	assert.True(t, ok)
	assert.Equal(t, []byte("master"), mk)
	assert.True(t, s.Unlocked())
}

func TestSession_VaultKeys(t *testing.T) {
	s := session.New()

	_, ok := s.VaultKey("v1")
	assert.False(t, ok)

	s.OpenVault("v1", []byte("key1"))
	vk, ok := s.VaultKey("v1")
	assert.True(t, ok)
	assert.Equal(t, []byte("key1"), vk)
}

func TestSession_Lock(t *testing.T) {
	s := session.New()
	s.SetMasterKey([]byte("master"))
	s.OpenVault("v1", []byte("key1"))

	s.Lock()

	assert.False(t, s.Unlocked())
	_, ok := s.VaultKey("v1")
	assert.False(t, ok)
}
