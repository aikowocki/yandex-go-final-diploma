package grpcclient_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	secretusecase "github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
)

// TestGRPCClient_UploadBlob_StorageDisabled — когда сервер собран
// без объектного хранилища (blobUseCase == nil, штатный режим без MinIO), UploadBlob должен
// возвращать codes.Unimplemented, а не паниковать.
func TestGRPCClient_UploadBlob_StorageDisabled(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)

	_, _, err := b.Client.UploadBlob(context.Background(), tok, "s1", bytes.NewReader([]byte("data")))
	require.Error(t, err)
}

// TestGRPCClient_DownloadBlob_StorageDisabled — тот же тест для DownloadBlob.
func TestGRPCClient_DownloadBlob_StorageDisabled(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)

	rc, err := b.Client.DownloadBlob(context.Background(), tok, "s1")
	if err == nil {
		_, rerr := io.ReadAll(rc)
		require.Error(t, rerr)
		_ = rc.Close()
	} else {
		require.Error(t, err)
	}
}

// TestGRPCClient_UploadBlob_NoUser — запрос без токена (нет userID в контексте) должен
// вернуть ошибку до попытки чтения из стрима/хранилища.
func TestGRPCClient_UploadBlob_NoUser(t *testing.T) {
	b := newTestGRPCClientWithBlobStorage(t)

	_, _, err := b.Client.UploadBlob(context.Background(), "", "s1", bytes.NewReader([]byte("data")))
	require.Error(t, err)
}

// TestGRPCClient_DownloadBlob_NoUser — то же для DownloadBlob.
func TestGRPCClient_DownloadBlob_NoUser(t *testing.T) {
	b := newTestGRPCClientWithBlobStorage(t)

	rc, err := b.Client.DownloadBlob(context.Background(), "", "s1")
	if err == nil {
		_, rerr := io.ReadAll(rc)
		require.Error(t, rerr)
		_ = rc.Close()
	} else {
		require.Error(t, err)
	}
}

func TestGRPCClient_AttachBlob_SecretNotFound(t *testing.T) {
	b := newTestGRPCClient(t)
	tok := mustAccessToken(t, b)
	b.Secret.EXPECT().GetForUpdate(mock.Anything, "s1", mock.Anything).
		Return(domain.Secret{}, secretusecase.ErrSecretNotFound)

	_, err := b.Client.AttachBlob(context.Background(), tok, "s1", 1, "ref", 100)
	require.Error(t, err)
}

// TestGRPCClient_UploadBlob_Success — сервер собран с рабочим blob.UseCase (мок-хранилище),
// проверяем полный путь стриминга: чанки данных клиента доходят до storage.PutChunk.
func TestGRPCClient_UploadBlob_Success(t *testing.T) {
	b := newTestGRPCClientWithBlobStorage(t)
	tok := mustAccessToken(t, b)

	b.Secret.EXPECT().GetForUpdate(mock.Anything, "s1", mock.Anything).
		Return(domain.Secret{ID: "s1", VaultID: "v1"}, nil)
	b.Storage.EXPECT().PutChunk(mock.Anything, "v1/s1", mock.Anything, int64(-1)).
		RunAndReturn(func(_ context.Context, _ string, r io.Reader, _ int64) (int64, error) {
			data, err := io.ReadAll(r)
			require.NoError(t, err)
			return int64(len(data)), nil
		})

	payload := bytes.Repeat([]byte("x"), 200*1024) // больше одного chunk (64KB)
	blobRef, size, err := b.Client.UploadBlob(context.Background(), tok, "s1", bytes.NewReader(payload))
	require.NoError(t, err)
	assert.Equal(t, "v1/s1", blobRef)
	assert.Equal(t, int64(len(payload)), size)
}

// TestGRPCClient_UploadBlob_EmptyStream — нулевой файл: клиент всё равно шлёт одно сообщение
// с secret_id, PutChunk получает пустой Reader.
func TestGRPCClient_UploadBlob_EmptyStream(t *testing.T) {
	b := newTestGRPCClientWithBlobStorage(t)
	tok := mustAccessToken(t, b)

	b.Secret.EXPECT().GetForUpdate(mock.Anything, "s1", mock.Anything).
		Return(domain.Secret{ID: "s1", VaultID: "v1"}, nil)
	b.Storage.EXPECT().PutChunk(mock.Anything, "v1/s1", mock.Anything, int64(-1)).
		Return(int64(0), nil)

	_, _, err := b.Client.UploadBlob(context.Background(), tok, "s1", bytes.NewReader(nil))
	require.Error(t, err, "нулевой размер блоба должен считаться ошибкой (ErrNoData)")
}

// TestGRPCClient_UploadBlob_SecretNotFound — секрет не найден, стрим должен быть закрыт с
// NotFound ещё до записи в storage.
func TestGRPCClient_UploadBlob_SecretNotFound(t *testing.T) {
	b := newTestGRPCClientWithBlobStorage(t)
	tok := mustAccessToken(t, b)

	b.Secret.EXPECT().GetForUpdate(mock.Anything, "s1", mock.Anything).
		Return(domain.Secret{}, secretusecase.ErrSecretNotFound)

	_, _, err := b.Client.UploadBlob(context.Background(), tok, "s1", bytes.NewReader([]byte("data")))
	require.Error(t, err)
}

// TestGRPCClient_DownloadBlob_Success — читаем поток из storage.GetStream чанками через
// DownloadBlob (server-streaming RPC), сверяем итоговые байты.
func TestGRPCClient_DownloadBlob_Success(t *testing.T) {
	b := newTestGRPCClientWithBlobStorage(t)
	tok := mustAccessToken(t, b)

	content := bytes.Repeat([]byte("y"), 150*1024)
	blobRef := "v1/s1"
	b.Secret.EXPECT().GetForUpdate(mock.Anything, "s1", mock.Anything).
		Return(domain.Secret{ID: "s1", VaultID: "v1", BlobRef: &blobRef}, nil)
	b.Storage.EXPECT().GetStream(mock.Anything, blobRef).
		Return(io.NopCloser(bytes.NewReader(content)), nil)

	rc, err := b.Client.DownloadBlob(context.Background(), tok, "s1")
	require.NoError(t, err)
	defer func() { _ = rc.Close() }()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

// TestGRPCClient_DownloadBlob_SecretNotFound — секрет без blob_ref не должен запускать
// стриминг (сервер сразу возвращает NotFound).
func TestGRPCClient_DownloadBlob_SecretNotFound(t *testing.T) {
	b := newTestGRPCClientWithBlobStorage(t)
	tok := mustAccessToken(t, b)

	b.Secret.EXPECT().GetForUpdate(mock.Anything, "s1", mock.Anything).
		Return(domain.Secret{ID: "s1", VaultID: "v1"}, nil) // BlobRef == nil

	rc, err := b.Client.DownloadBlob(context.Background(), tok, "s1")
	if err == nil {
		_, rerr := io.ReadAll(rc)
		require.Error(t, rerr)
		_ = rc.Close()
	} else {
		require.Error(t, err)
	}
}
