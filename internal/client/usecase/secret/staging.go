package secret

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const pendingUploadsDir = "pending_uploads"

// StageFile копирует файл из reader в staging area и возвращает путь к staged-копии.
// Создаёт директорию <dataDir>/pending_uploads/<secretID>/ при необходимости.
func StageFile(dataDir, secretID, filename string, data io.Reader) (stagedPath string, err error) {
	dir := filepath.Join(dataDir, pendingUploadsDir, secretID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("staging: mkdir: %w", err)
	}

	if filename == "" {
		filename = "blob" // защита от пустого имени (staged-путь не должен совпасть с dir)
	}
	stagedPath = filepath.Join(dir, filename)
	f, err := os.Create(stagedPath)
	if err != nil {
		return "", fmt.Errorf("staging: create: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, data); err != nil {
		_ = os.Remove(stagedPath)
		return "", fmt.Errorf("staging: copy: %w", err)
	}
	return stagedPath, nil
}

// CleanupStaged удаляет staging-директорию секрета (после успешного upload).
func CleanupStaged(dataDir, secretID string) {
	dir := filepath.Join(dataDir, pendingUploadsDir, secretID)
	_ = os.RemoveAll(dir)
}

// StagedFilePath возвращает путь к staged файлу (для повторного upload из outbox).
// Если файл не существует — возвращает ошибку.
func StagedFilePath(dataDir, secretID string) (string, error) {
	dir := filepath.Join(dataDir, pendingUploadsDir, secretID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("staging: read dir: %w", err)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("staging: no staged file for secret %s", secretID)
	}
	// Берём первый файл (секрет имеет ровно один blob).
	return filepath.Join(dir, entries[0].Name()), nil
}
