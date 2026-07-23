package secret_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts/mocks"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/usecase/secret"
)

// fakeBlobBackend имитирует сервер, транзитом храня зашифрованные байты, полученные через
// UploadBlob, и отдающие их же через DownloadBlob — достаточно для проверки клиентского
// шифрования/расшифровки без реального gRPC/MinIO.
type fakeBlobBackend struct {
	stored []byte
}

func (f *fakeBlobBackend) upload(r io.Reader) (string, int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", 0, err
	}
	f.stored = data
	return "vault-1/secret-1", int64(len(data)), nil
}

func (f *fakeBlobBackend) download() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(f.stored)), nil
}

func TestCreateBinary_UploadAndDownload_RoundTrip(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	backend := &fakeBlobBackend{}

	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		CreateSecret(mock.Anything, "tok", mock.Anything, "v1", int32(3), mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	server.EXPECT().
		UploadBlob(mock.Anything, "tok", mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _, _ string, r io.Reader) (string, int64, error) {
			return backend.upload(r)
		})
	server.EXPECT().
		AttachBlob(mock.Anything, "tok", mock.Anything, int64(1), "vault-1/secret-1", mock.Anything).
		Return(int64(2), nil)

	original := bytes.Repeat([]byte("gophkeeper-binary-data-chunk-"), 5000) // > один чанк потокового AEAD не обязателен, но проверим целостность
	uc := newSecretUC(t, server, sess)

	id, err := uc.CreateBinary(context.Background(), "v1", secret.CreateBinaryInput{
		Title: "photo.png", Filename: "photo.png", Data: bytes.NewReader(original), Size: int64(len(original)),
	})
	require.NoError(t, err)
	require.NotEmpty(t, id)

	server.EXPECT().
		DownloadBlob(mock.Anything, "tok", id).
		RunAndReturn(func(context.Context, string, string) (io.ReadCloser, error) {
			return backend.download()
		})

	var out bytes.Buffer
	err = uc.DownloadBinary(context.Background(), "v1", id, &out)
	require.NoError(t, err)
	require.Equal(t, original, out.Bytes())
}

func TestCreateBinary_EmptyFile_RoundTrip(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	backend := &fakeBlobBackend{}

	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		CreateSecret(mock.Anything, "tok", mock.Anything, "v1", int32(3), mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	server.EXPECT().
		UploadBlob(mock.Anything, "tok", mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _, _ string, r io.Reader) (string, int64, error) {
			return backend.upload(r)
		})
	server.EXPECT().
		AttachBlob(mock.Anything, "tok", mock.Anything, int64(1), "vault-1/secret-1", mock.Anything).
		Return(int64(2), nil)

	uc := newSecretUC(t, server, sess)
	id, err := uc.CreateBinary(context.Background(), "v1", secret.CreateBinaryInput{
		Title: "empty.txt", Filename: "empty.txt", Data: bytes.NewReader(nil), Size: 0,
	})
	require.NoError(t, err)

	server.EXPECT().
		DownloadBlob(mock.Anything, "tok", id).
		RunAndReturn(func(context.Context, string, string) (io.ReadCloser, error) {
			return backend.download()
		})

	var out bytes.Buffer
	err = uc.DownloadBinary(context.Background(), "v1", id, &out)
	require.NoError(t, err)
	require.Empty(t, out.Bytes())
}

func TestCreateBinary_RequiresData(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	uc := newSecretUC(t, mocks.NewMockServerClient(t), sess) // CreateSecret не должен вызываться
	_, err := uc.CreateBinary(context.Background(), "v1", secret.CreateBinaryInput{Title: "x"})
	require.ErrorIs(t, err, secret.ErrEmptyBinaryData)
}

func TestCreateBinary_RequiresTitle(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	uc := newSecretUC(t, mocks.NewMockServerClient(t), sess)
	_, err := uc.CreateBinary(context.Background(), "v1", secret.CreateBinaryInput{Data: bytes.NewReader(nil)})
	require.ErrorIs(t, err, secret.ErrEmptyTitle)
}

// TestDownloadBinary_TamperedStreamFails: если байты потока повреждены на пути (например
// MinIO/сервер скомпрометирован и подменил чанк), расшифровка должна провалиться, не отдать
// повреждённые данные молча.
func TestDownloadBinary_TamperedStreamFails(t *testing.T) {
	sess, _ := openVaultSession(t, "v1")
	backend := &fakeBlobBackend{}

	server := mocks.NewMockServerClient(t)
	server.EXPECT().
		CreateSecret(mock.Anything, "tok", mock.Anything, "v1", int32(3), mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	server.EXPECT().
		UploadBlob(mock.Anything, "tok", mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _, _ string, r io.Reader) (string, int64, error) {
			return backend.upload(r)
		})
	server.EXPECT().
		AttachBlob(mock.Anything, "tok", mock.Anything, int64(1), "vault-1/secret-1", mock.Anything).
		Return(int64(2), nil)

	uc := newSecretUC(t, server, sess)
	id, err := uc.CreateBinary(context.Background(), "v1", secret.CreateBinaryInput{
		Title: "f.bin", Data: bytes.NewReader([]byte("some plaintext bytes")), Size: 21,
	})
	require.NoError(t, err)

	// Портим последний байт зашифрованного потока (после streamID+length-prefix framing).
	backend.stored[len(backend.stored)-1] ^= 0xff

	server.EXPECT().
		DownloadBlob(mock.Anything, "tok", id).
		RunAndReturn(func(context.Context, string, string) (io.ReadCloser, error) {
			return backend.download()
		})

	var out bytes.Buffer
	err = uc.DownloadBinary(context.Background(), "v1", id, &out)
	require.Error(t, err)
}
