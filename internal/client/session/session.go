package session

import "sync"

// Session хранит MasterKey, открытые VaultKey и PIN-материал текущего процесса в памяти.
type Session struct {
	mu        sync.RWMutex
	masterKey []byte
	vaultKeys map[string][]byte // vaultID → unwrapped VaultKey

	// PIN-материал живёт только в памяти процесса («тёплая» сессия). Позволяет после
	// авто-блокировки (SoftLock) разблокироваться PIN'ом без полного master-пароля.
	// pinWrapped — MasterKey, обёрнутый ключом, выведенным из PIN; pinSalt/pinParams —
	// параметры вывода pin-ключа. На диск НЕ сохраняется: при перезапуске процесса PIN
	// сбрасывается, нужен полный master-пароль.
	pinWrapped []byte
	pinSalt    []byte
	pinParams  []byte // JSON crypto.Params
}

// New создаёт пустую сессию без MasterKey и открытых VaultKey.
func New() *Session {
	return &Session{vaultKeys: make(map[string][]byte)}
}

// SetMasterKey сохраняет MasterKey в сессии.
func (s *Session) SetMasterKey(mk []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.masterKey = mk
}

// MasterKey возвращает сохранённый MasterKey и признак его наличия.
func (s *Session) MasterKey() ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.masterKey, len(s.masterKey) > 0
}

// Unlocked сообщает, разблокирована ли сессия (установлен ли MasterKey).
func (s *Session) Unlocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.masterKey) > 0
}

// OpenVault сохраняет VaultKey открытой папки в сессии.
func (s *Session) OpenVault(vaultID string, vaultKey []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vaultKeys[vaultID] = vaultKey
}

// VaultKey возвращает VaultKey открытой папки и признак его наличия.
func (s *Session) VaultKey(vaultID string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vk, ok := s.vaultKeys[vaultID]
	return vk, ok
}

// Lock — полная блокировка: стирает MasterKey, VaultKey и PIN-материал. Используется при
// явном выходе/смене аккаунта — после неё разблокировка возможна только полным master-паролем.
func (s *Session) Lock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.zeroKeys()
	zero(s.pinWrapped)
	zero(s.pinSalt)
	zero(s.pinParams)
	s.pinWrapped, s.pinSalt, s.pinParams = nil, nil, nil
}

// SoftLock — мягкая блокировка (авто-лок по таймауту): стирает MasterKey и VaultKey, но
// СОХРАНЯЕТ PIN-материал, чтобы пользователь мог разблокироваться PIN'ом без полного пароля.
func (s *Session) SoftLock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.zeroKeys()
}

// zeroKeys затирает MasterKey и все VaultKey (вызывать под удержанным mu).
func (s *Session) zeroKeys() {
	zero(s.masterKey)
	s.masterKey = nil
	for id, vk := range s.vaultKeys {
		zero(vk)
		delete(s.vaultKeys, id)
	}
}

// SetPINMaterial сохраняет обёрнутый PIN'ом MasterKey и параметры вывода pin-ключа (в памяти).
func (s *Session) SetPINMaterial(wrapped, salt, params []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pinWrapped = append([]byte(nil), wrapped...)
	s.pinSalt = append([]byte(nil), salt...)
	s.pinParams = append([]byte(nil), params...)
}

// PINMaterial возвращает сохранённый PIN-материал (копии) и признак его наличия.
func (s *Session) PINMaterial() (wrapped, salt, params []byte, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.pinWrapped) == 0 {
		return nil, nil, nil, false
	}
	return append([]byte(nil), s.pinWrapped...),
		append([]byte(nil), s.pinSalt...),
		append([]byte(nil), s.pinParams...),
		true
}

// HasPIN сообщает, установлен ли PIN в текущей сессии.
func (s *Session) HasPIN() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pinWrapped) > 0
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
