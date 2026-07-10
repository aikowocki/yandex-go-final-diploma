package contracts

import (
	"context"
	"io"
)

type BlobStorage interface {
	PutChunk(ctx context.Context, key string, r io.Reader, size int64) (written int64, err error)
	GetStream(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
