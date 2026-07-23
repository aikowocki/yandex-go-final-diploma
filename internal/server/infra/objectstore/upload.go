package objectstore

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

// PutChunk загружает содержимое r как объект key в bucket. size — известный размер потока
// (-1, если неизвестен — minio-go в этом случае буферизует по частям сам). Возвращает итоговый
// размер объекта.
func (s *Store) PutChunk(ctx context.Context, key string, r io.Reader, size int64) (int64, error) {
	info, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream", // ciphertext — не имеет смысла хранить исходный MIME
	})
	if err != nil {
		return 0, fmt.Errorf("objectstore: put object %q: %w", key, err)
	}
	return info.Size, nil
}
