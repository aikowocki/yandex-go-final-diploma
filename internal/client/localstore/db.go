// Package localstore реализует локальное SQLite-хранилище клиента (кеш секретов/папок +
// оффлайн-очередь). Хранит те же E2E-шифротексты, что и сервер; MasterKey здесь не лежит.
package localstore

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

// Store — реализация contracts.LocalStorage поверх SQLite.
type Store struct {
	db *sql.DB
}

var _ contracts.LocalStorage = (*Store)(nil)

const dbFileName = "local.db"

// Open открывает локальную БД. При persist=false используется in-memory база,
// которая не переживает перезапуск процесса (режим --no-persist). Схема мигрируется автоматически.
func Open(dataDir string, persist bool) (*Store, error) {
	dsn := ":memory:"
	if persist {
		if err := os.MkdirAll(dataDir, 0o700); err != nil {
			return nil, fmt.Errorf("localstore: create data dir: %w", err)
		}
		dsn = "file:" + filepath.Join(dataDir, dbFileName) + "?_pragma=busy_timeout(5000)"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("localstore: open: %w", err)
	}
	// Один коннект: SQLite — single-writer, а для :memory: это гарантирует единственную БД.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("localstore: ping: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("localstore: migrate: %w", err)
	}
	return s, nil
}

// Close закрывает БД.
func (s *Store) Close() error {
	return s.db.Close()
}
