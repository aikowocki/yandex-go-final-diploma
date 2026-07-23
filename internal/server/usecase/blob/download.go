package blob

import (
	"context"
	"fmt"
	"io"
)

// DownloadChunked проверяет владение секретом и отдаёт поток на чтение объекта из хранилища.
// Вызывающий обязан закрыть возвращённый io.ReadCloser.
func (u *UseCase) DownloadChunked(ctx context.Context, userID, secretID string) (io.ReadCloser, error) {
	if u.storage == nil {
		return nil, ErrBlobStorageDisabled
	}
	if userID == "" {
		return nil, ErrEmptyUserID
	}
	if secretID == "" {
		return nil, ErrEmptySecretID
	}

	sec, err := u.secrets.GetForUpdate(ctx, secretID, userID)
	if err != nil {
		return nil, err
	}
	if sec.Deleted {
		return nil, ErrSecretNotFound
	}
	if sec.BlobRef == nil || *sec.BlobRef == "" {
		return nil, ErrSecretNotFound
	}

	stream, err := u.storage.GetStream(ctx, *sec.BlobRef)
	if err != nil {
		return nil, fmt.Errorf("blob: download: %w", err)
	}
	return stream, nil
}
