// Package keyring реализует contracts.TokenStore: хранение JWT на стороне клиента.
// Основной путь — OS keyring (zalando/go-keyring), с fallback в файл (0600, DATA_DIR), если
// keyring недоступен. Режим persist=false (--no-persist/NoPersist) отключает ОБА эти пути и
// держит токены только в памяти процесса.
package keyring

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/zalando/go-keyring"
)

const (
	// service/account — координаты записи в OS keyring.
	service = "gophkeeper"
	account = "auth-tokens"
	// tokenFileName — имя файла fallback-хранилища в DATA_DIR.
	tokenFileName = "token.json"
	// tokenFileMode — файл доступен только владельцу.
	tokenFileMode = 0o600
)

// ErrNoToken — токены не сохранены (ни в keyring, ни в файле, ни в памяти).
var ErrNoToken = errors.New("keyring: no token stored")

// Store — реализация contracts.TokenStore.
type Store struct {
	dataDir string
	persist bool

	mu  sync.RWMutex
	mem *contracts.Tokens // используется только когда persist=false
}

var _ contracts.TokenStore = (*Store)(nil)

// New создаёт хранилище токенов. allowFile разрешает fallback в файл, когда OS keyring
// недоступен (передаётся из конфига: !NoPersist).
func New(dataDir string, persist bool) *Store {
	return &Store{dataDir: dataDir, persist: persist}
}

// Save сохраняет токены. Если persist=false — только в памяти процесса. Иначе — в OS keyring,
// при ошибке — в файл.
func (s *Store) Save(t contracts.Tokens) error {
	if !s.persist {
		s.mu.Lock()
		s.mem = &t
		s.mu.Unlock()
		return nil
	}

	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("keyring: marshal tokens: %w", err)
	}

	if keyErr := keyring.Set(service, account, string(data)); keyErr == nil {
		return nil
	}

	return s.saveFile(data)
}

// Load читает токены из OS keyring, при отсутствии/ошибке — из файла (если разрешено).
func (s *Store) Load() (contracts.Tokens, error) {
	if !s.persist {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.mem == nil {
			return contracts.Tokens{}, ErrNoToken
		}
		return *s.mem, nil
	}

	raw, keyErr := keyring.Get(service, account)
	if keyErr == nil {
		return unmarshalTokens([]byte(raw))
	}

	return s.loadFile()
}

// Clear удаляет сохранённые токены (из памяти, либо из OS keyring и файла — в зависимости
// от режима persist).
func (s *Store) Clear() error {
	if !s.persist {
		s.mu.Lock()
		s.mem = nil
		s.mu.Unlock()
		return nil
	}

	var errs []error

	if err := keyring.Delete(service, account); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		errs = append(errs, fmt.Errorf("keyring: delete: %w", err))
	}

	if err := os.Remove(s.tokenPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs = append(errs, fmt.Errorf("keyring: remove file: %w", err))
	}

	return errors.Join(errs...)
}

func (s *Store) tokenPath() string {
	return filepath.Join(s.dataDir, tokenFileName)
}

func (s *Store) saveFile(data []byte) error {
	if err := os.MkdirAll(s.dataDir, 0o700); err != nil {
		return fmt.Errorf("keyring: mkdir data dir: %w", err)
	}
	if err := os.WriteFile(s.tokenPath(), data, tokenFileMode); err != nil {
		return fmt.Errorf("keyring: write token file: %w", err)
	}
	return nil
}

func (s *Store) loadFile() (contracts.Tokens, error) {
	data, err := os.ReadFile(s.tokenPath())
	if errors.Is(err, os.ErrNotExist) {
		return contracts.Tokens{}, ErrNoToken
	}
	if err != nil {
		return contracts.Tokens{}, fmt.Errorf("keyring: read token file: %w", err)
	}
	return unmarshalTokens(data)
}

func unmarshalTokens(data []byte) (contracts.Tokens, error) {
	var t contracts.Tokens
	if err := json.Unmarshal(data, &t); err != nil {
		return contracts.Tokens{}, fmt.Errorf("keyring: unmarshal tokens: %w", err)
	}
	return t, nil
}
