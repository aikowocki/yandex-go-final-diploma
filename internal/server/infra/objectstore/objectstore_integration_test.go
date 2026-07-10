//go:build integration

package objectstore_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	miniomodule "github.com/testcontainers/testcontainers-go/modules/minio"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/infra/objectstore"
)

func newTestStore(t *testing.T) *objectstore.Store {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const user, pass = "gophkeeper", "gophkeeper-minio"
	container, err := miniomodule.Run(ctx, "minio/minio:latest",
		miniomodule.WithUsername(user),
		miniomodule.WithPassword(pass),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	endpoint, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	store, err := objectstore.New(ctx, objectstore.Config{
		Endpoint:  endpoint,
		AccessKey: user,
		SecretKey: pass,
		Bucket:    "gophkeeper-blobs-test",
		UseSSL:    false,
	})
	require.NoError(t, err)
	return store
}

// TestStore_PutGetDelete_RoundTrip проверяет полный цикл против настоящего MinIO: заливка
// байт, чтение обратно (побайтовое совпадение), удаление, повторное чтение после удаления —
// GetStream должен вернуть ошибку (объект отсутствует).
func TestStore_PutGetDelete_RoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	payload := bytes.Repeat([]byte("ciphertext-bytes-not-plaintext-"), 10000) // ~320KB
	key := "vault-1/secret-1"

	written, err := store.PutChunk(ctx, key, bytes.NewReader(payload), int64(len(payload)))
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), written)

	rc, err := store.GetStream(ctx, key)
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, rc.Close())
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	require.NoError(t, store.Delete(ctx, key))

	_, err = store.GetStream(ctx, key)
	// После удаления GetStream всё равно возвращает объект-обёртку без ошибки (ленивое чтение
	// у minio-go), ошибка проявляется при первом Read — проверяем именно это.
	if err == nil {
		rc2, _ := store.GetStream(ctx, key)
		_, rerr := io.ReadAll(rc2)
		assert.Error(t, rerr, "reading a deleted object must fail")
		_ = rc2.Close()
	}
}

// TestStore_PutChunk_UnknownSize проверяет заливку с size=-1 (как делает usecase/blob.UploadChunked
// для потока без известной заранее длины).
func TestStore_PutChunk_UnknownSize(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	payload := []byte("streamed data of unknown length")
	written, err := store.PutChunk(ctx, "vault-1/secret-2", bytes.NewReader(payload), -1)
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), written)

	rc, err := store.GetStream(ctx, "vault-1/secret-2")
	require.NoError(t, err)
	defer rc.Close()
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestStore_BucketCreatedOnStart_Idempotent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const user, pass = "gophkeeper", "gophkeeper-minio"
	container, err := miniomodule.Run(ctx, "minio/minio:latest",
		miniomodule.WithUsername(user),
		miniomodule.WithPassword(pass),
	)
	require.NoError(t, err)
	defer container.Terminate(context.Background())

	endpoint, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	cfg := objectstore.Config{Endpoint: endpoint, AccessKey: user, SecretKey: pass, Bucket: "idempotent-bucket"}
	_, err = objectstore.New(ctx, cfg)
	require.NoError(t, err)

	// Повторный New() с тем же bucket — не должен упасть (BucketExists проверка перед MakeBucket).
	_, err = objectstore.New(ctx, cfg)
	require.NoError(t, err)
}
