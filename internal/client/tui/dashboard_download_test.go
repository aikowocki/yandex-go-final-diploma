package tui

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

func TestNewDownloadPickerPopup_DefaultsFilenameWhenEmpty(t *testing.T) {
	m := newDownloadPickerPopup("v1", "s1", "")
	assert.Equal(t, "v1", m.vaultID)
	assert.Equal(t, "s1", m.secretID)
	assert.Equal(t, "download", m.filename)
}

func TestNewDownloadPickerPopup_KeepsGivenFilename(t *testing.T) {
	m := newDownloadPickerPopup("v1", "s1", "photo.png")
	assert.Equal(t, "photo.png", m.filename)
}

func TestDownloadPickerPopup_Init_ReturnsCmd(t *testing.T) {
	m := newDownloadPickerPopup("v1", "s1", "f.txt")
	assert.NotNil(t, m.Init())
}

func TestDownloadPickerPopup_Update_EscClosesPopup(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	m := newDownloadPickerPopup("v1", "s1", "f.txt")

	updated, cmd, done := m.update(context.Background(), c, tea.KeyMsg{Type: tea.KeyEsc})
	assert.True(t, done)
	assert.Nil(t, cmd)
	assert.Equal(t, m.vaultID, updated.vaultID)
}

func TestDownloadPickerPopup_Update_OtherKeyDelegatesToFilepicker(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	m := newDownloadPickerPopup("v1", "s1", "f.txt")

	updated, _, done := m.update(context.Background(), c, tea.KeyMsg{Type: tea.KeyDown})
	assert.False(t, done)
	assert.Equal(t, "v1", updated.vaultID)
}

func TestDownloadPickerPopup_View_RendersTitleAndHelp(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))
	m := newDownloadPickerPopup("v1", "s1", "f.txt")
	out := m.view(c.Localizer)
	assert.NotEmpty(t, out)
}

// fakeBlobBackendTUI имитирует сервер, транзитом храня зашифрованные байты, полученные через
// UploadBlob и отдающие их же через DownloadBlob (тот же паттерн, что usecase/secret/binary_test.go).
type fakeBlobBackendTUI struct {
	stored []byte
}

func (f *fakeBlobBackendTUI) upload(r io.Reader) (string, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", 0, err
	}
	f.stored = data
	return "vault-1/secret-1", int64(len(data)), nil
}

func (f *fakeBlobBackendTUI) download() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(f.stored)), nil
}

func TestDownloadPickerPopup_StartDownload_Success(t *testing.T) {
	backend := &fakeBlobBackendTUI{}
	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		CreateSecret(mock.Anything, mock.Anything, mock.Anything, "v1", int32(3), mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	server.EXPECT().
		UploadBlob(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _, _ string, r io.Reader) (string, int64, error) {
			return backend.upload(r)
		})
	server.EXPECT().
		AttachBlob(mock.Anything, mock.Anything, mock.Anything, int64(1), "vault-1/secret-1", mock.Anything).
		Return(int64(2), nil)

	c := newTestContainer(t, server)
	vaultKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	c.Session.OpenVault("v1", vaultKey)

	original := []byte("gophkeeper secret file content for download test")
	id, err := c.Secret.CreateBinary(context.Background(), "v1", secret.CreateBinaryInput{
		Title: "myfile.bin", Filename: "myfile.bin", Data: bytes.NewReader(original), Size: int64(len(original)),
	})
	require.NoError(t, err)

	server.EXPECT().
		DownloadBlob(mock.Anything, mock.Anything, id).
		RunAndReturn(func(context.Context, string, string) (io.ReadCloser, error) {
			return backend.download()
		})

	dir := t.TempDir()
	m := newDownloadPickerPopup("v1", id, "myfile.bin")
	cmd := m.startDownload(context.Background(), c, dir)
	require.NotNil(t, cmd)

	msg := cmd()
	toast, ok := msg.(toastMsg)
	require.True(t, ok)
	assert.Contains(t, toast.text, dir)

	data, err := os.ReadFile(filepath.Join(dir, "myfile.bin"))
	require.NoError(t, err)
	assert.Equal(t, original, data)
}

func TestDownloadPickerPopup_StartDownload_CreateFileError(t *testing.T) {
	c := newTestContainer(t, mocks.NewMockServerClient(t))

	// Директория не существует — os.Create должен провалиться.
	m := newDownloadPickerPopup("v1", "s1", "f.bin")
	cmd := m.startDownload(context.Background(), c, "/nonexistent/path/for/test")
	require.NotNil(t, cmd)

	msg := cmd()
	toast, ok := msg.(toastMsg)
	require.True(t, ok)
	assert.Contains(t, toast.text, "✗")
}

func TestDownloadPickerPopup_StartDownload_DownloadError(t *testing.T) {
	server := mocks.NewMockServerClient(t)
	server.EXPECT().DownloadBlob(mock.Anything, mock.Anything, "s1").
		Return(nil, assert.AnError)

	c := newTestContainer(t, server)
	vaultKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	c.Session.OpenVault("v1", vaultKey)

	dir := t.TempDir()
	m := newDownloadPickerPopup("v1", "s1", "f.bin")
	cmd := m.startDownload(context.Background(), c, dir)
	require.NotNil(t, cmd)

	msg := cmd()
	toast, ok := msg.(toastMsg)
	require.True(t, ok)
	assert.Contains(t, toast.text, "✗")

	_, statErr := os.Stat(filepath.Join(dir, "f.bin"))
	assert.True(t, os.IsNotExist(statErr), "файл должен быть удалён после ошибки скачивания")
}
