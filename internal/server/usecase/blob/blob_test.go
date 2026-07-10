package blob_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/blob"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/blob/mocks"
)

type memStorage struct {
	objects map[string][]byte
}

func newMemStorage() *memStorage { return &memStorage{objects: map[string][]byte{}} }

func (m *memStorage) PutChunk(_ context.Context, key string, r io.Reader, _ int64) (int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	m.objects[key] = data
	return int64(len(data)), nil
}

func (m *memStorage) GetStream(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := m.objects[key]
	if !ok {
		return nil, assert.AnError
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *memStorage) Delete(_ context.Context, key string) error {
	delete(m.objects, key)
	return nil
}

func TestUploadChunked_Success(t *testing.T) {
	t.Parallel()

	lookup := mocks.NewMockSecretLookup(t)
	lookup.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").
		Return(domain.Secret{ID: "secret-1", VaultID: "vault-1", Type: domain.SecretTypeBinary}, nil)

	storage := newMemStorage()
	uc := blob.New(storage, lookup)

	payload := []byte("encrypted-chunk-data")
	ref, size, err := uc.UploadChunked(context.Background(), "user-1", "secret-1", bytes.NewReader(payload))
	require.NoError(t, err)
	assert.Equal(t, "vault-1/secret-1", ref)
	assert.Equal(t, int64(len(payload)), size)
	assert.Equal(t, payload, storage.objects[ref])
}

func TestUploadChunked_StorageDisabled(t *testing.T) {
	t.Parallel()

	uc := blob.New(nil, mocks.NewMockSecretLookup(t))
	_, _, err := uc.UploadChunked(context.Background(), "user-1", "secret-1", bytes.NewReader(nil))
	require.ErrorIs(t, err, blob.ErrBlobStorageDisabled)
}

func TestUploadChunked_Validation(t *testing.T) {
	t.Parallel()

	uc := blob.New(newMemStorage(), mocks.NewMockSecretLookup(t))

	_, _, err := uc.UploadChunked(context.Background(), "", "secret-1", bytes.NewReader(nil))
	require.ErrorIs(t, err, blob.ErrEmptyUserID)

	_, _, err = uc.UploadChunked(context.Background(), "user-1", "", bytes.NewReader(nil))
	require.ErrorIs(t, err, blob.ErrEmptySecretID)
}

func TestUploadChunked_NoData(t *testing.T) {
	t.Parallel()

	lookup := mocks.NewMockSecretLookup(t)
	lookup.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").
		Return(domain.Secret{ID: "secret-1", VaultID: "vault-1"}, nil)

	uc := blob.New(newMemStorage(), lookup)
	_, _, err := uc.UploadChunked(context.Background(), "user-1", "secret-1", bytes.NewReader(nil))
	require.ErrorIs(t, err, blob.ErrNoData)
}

func TestDownloadChunked_Success(t *testing.T) {
	t.Parallel()

	blobRef := "vault-1/secret-1"
	lookup := mocks.NewMockSecretLookup(t)
	lookup.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").
		Return(domain.Secret{ID: "secret-1", VaultID: "vault-1", BlobRef: &blobRef}, nil)

	storage := newMemStorage()
	storage.objects[blobRef] = []byte("ciphertext-bytes")

	uc := blob.New(storage, lookup)
	rc, err := uc.DownloadChunked(context.Background(), "user-1", "secret-1")
	require.NoError(t, err)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "ciphertext-bytes", string(got))
}

func TestDownloadChunked_NoBlobRef(t *testing.T) {
	t.Parallel()

	lookup := mocks.NewMockSecretLookup(t)
	lookup.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").
		Return(domain.Secret{ID: "secret-1", VaultID: "vault-1"}, nil)

	uc := blob.New(newMemStorage(), lookup)
	_, err := uc.DownloadChunked(context.Background(), "user-1", "secret-1")
	require.ErrorIs(t, err, blob.ErrSecretNotFound)
}

func TestDownloadChunked_Deleted(t *testing.T) {
	t.Parallel()

	blobRef := "vault-1/secret-1"
	lookup := mocks.NewMockSecretLookup(t)
	lookup.EXPECT().GetForUpdate(mock.Anything, "secret-1", "user-1").
		Return(domain.Secret{ID: "secret-1", VaultID: "vault-1", BlobRef: &blobRef, Deleted: true}, nil)

	uc := blob.New(newMemStorage(), lookup)
	_, err := uc.DownloadChunked(context.Background(), "user-1", "secret-1")
	require.ErrorIs(t, err, blob.ErrSecretNotFound)
}

func TestDownloadChunked_StorageDisabled(t *testing.T) {
	t.Parallel()

	uc := blob.New(nil, mocks.NewMockSecretLookup(t))
	_, err := uc.DownloadChunked(context.Background(), "user-1", "secret-1")
	require.ErrorIs(t, err, blob.ErrBlobStorageDisabled)
}
