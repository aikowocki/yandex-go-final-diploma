// Package keyring реализует contracts.TokenStore: хранение JWT на стороне клиента.
// Основной путь — OS keyring (zalando/go-keyring). Если он недоступен (нет keychain,
// headless-окружение и т.п.) и разрешён fallback — токены пишутся в файл с правами 0600
// в DATA_DIR.
package keyring

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

// ErrNoToken — токены не сохранены (ни в keyring, ни в файле).
var ErrNoToken = errors.New("keyring: no token stored")

// Store — реализация contracts.TokenStore.
type Store struct {
	dataDir   string
	allowFile bool
}

var _ contracts.TokenStore = (*Store)(nil)

// New создаёт хранилище токенов. allowFile разрешает fallback в файл, когда OS keyring
// недоступен (передаётся из конфига: !NoPersist).
func New(dataDir string, allowFile bool) *Store {
	return &Store{dataDir: dataDir, allowFile: allowFile}
}

// Save сохраняет токены: сначала в OS keyring, при ошибке — в файл (если разрешено).
func (s *Store) Save(t contracts.Tokens) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("keyring: marshal tokens: %w", err)
	}

	if keyErr := keyring.Set(service, account, string(data)); keyErr == nil {
		return nil
	} else if !s.allowFile {
		return fmt.Errorf("keyring: set: %w", keyErr)
	}

	return s.saveFile(data)
}

// Load читает токены из OS keyring, при отсутствии/ошибке — из файла (если разрешено).
func (s *Store) Load() (contracts.Tokens, error) {
	raw, keyErr := keyring.Get(service, account)
	if keyErr == nil {
		return unmarshalTokens([]byte(raw))
	}

	if !s.allowFile {
		if errors.Is(keyErr, keyring.ErrNotFound) {
			return contracts.Tokens{}, ErrNoToken
		}
		return contracts.Tokens{}, fmt.Errorf("keyring: get: %w", keyErr)
	}

	return s.loadFile()
}

// Clear удаляет токены из обоих хранилищ.
func (s *Store) Clear() error {
	var errs []error

	if err := keyring.Delete(service, account); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		errs = append(errs, fmt.Errorf("keyring: delete: %w", err))
	}

	if s.allowFile {
		if err := os.Remove(s.tokenPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("keyring: remove file: %w", err))
		}
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
