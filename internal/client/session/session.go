package session

import "sync"

type Session struct {
	mu        sync.RWMutex
	masterKey []byte
	vaultKeys map[string][]byte // vaultID → unwrapped VaultKey
}

func New() *Session {
	return &Session{vaultKeys: make(map[string][]byte)}
}

func (s *Session) SetMasterKey(mk []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.masterKey = mk
}

func (s *Session) MasterKey() ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.masterKey, len(s.masterKey) > 0
}

func (s *Session) Unlocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.masterKey) > 0
}

func (s *Session) OpenVault(vaultID string, vaultKey []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vaultKeys[vaultID] = vaultKey
}

func (s *Session) VaultKey(vaultID string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vk, ok := s.vaultKeys[vaultID]
	return vk, ok
}

func (s *Session) Lock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	zero(s.masterKey)
	s.masterKey = nil
	for id, vk := range s.vaultKeys {
		zero(vk)
		delete(s.vaultKeys, id)
	}
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
