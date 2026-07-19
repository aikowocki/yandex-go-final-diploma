package secret

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStageFile_CreatesFileWithContent(t *testing.T) {
	dataDir := t.TempDir()
	content := "hello binary data"

	path, err := StageFile(dataDir, "sec1", "myfile.bin", strings.NewReader(content))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dataDir, pendingUploadsDir, "sec1", "myfile.bin"), path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestStageFile_EmptyFilenameUsesBlob(t *testing.T) {
	dataDir := t.TempDir()

	path, err := StageFile(dataDir, "sec2", "", strings.NewReader("x"))
	require.NoError(t, err)
	assert.Contains(t, path, "blob")
}

func TestCleanupStaged_RemovesDirectory(t *testing.T) {
	dataDir := t.TempDir()

	_, err := StageFile(dataDir, "sec3", "f.txt", strings.NewReader("data"))
	require.NoError(t, err)

	CleanupStaged(dataDir, "sec3")

	dir := filepath.Join(dataDir, pendingUploadsDir, "sec3")
	_, err = os.Stat(dir)
	assert.True(t, os.IsNotExist(err))
}

func TestStagedFilePath_ReturnsFirstFile(t *testing.T) {
	dataDir := t.TempDir()

	staged, err := StageFile(dataDir, "sec4", "doc.pdf", strings.NewReader("pdf"))
	require.NoError(t, err)

	found, err := StagedFilePath(dataDir, "sec4")
	require.NoError(t, err)
	assert.Equal(t, staged, found)
}

func TestStagedFilePath_ErrorIfNoFiles(t *testing.T) {
	dataDir := t.TempDir()

	_, err := StagedFilePath(dataDir, "nonexistent")
	assert.Error(t, err)
}

func TestStagedFilePath_ErrorIfDirEmpty(t *testing.T) {
	dataDir := t.TempDir()
	dir := filepath.Join(dataDir, pendingUploadsDir, "empty")
	require.NoError(t, os.MkdirAll(dir, 0o700))

	_, err := StagedFilePath(dataDir, "empty")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no staged file")
}
