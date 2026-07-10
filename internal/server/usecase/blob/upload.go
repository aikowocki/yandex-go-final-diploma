package blob

import (
	"context"
	"fmt"
	"io"
)

// UploadChunked проверяет владение секретом и заливает поток r (уже зашифрованные клиентом
// байты) в объектное хранилище под ключом vaultID/secretID.
func (u *UseCase) UploadChunked(ctx context.Context, userID, secretID string, r io.Reader) (blobRef string, size int64, err error) {
	if u.storage == nil {
		return "", 0, ErrBlobStorageDisabled
	}
	if userID == "" {
		return "", 0, ErrEmptyUserID
	}
	if secretID == "" {
		return "", 0, ErrEmptySecretID
	}

	sec, err := u.secrets.GetForUpdate(ctx, secretID, userID)
	if err != nil {
		return "", 0, err
	}

	key := objectKey(sec.VaultID, secretID)
	written, err := u.storage.PutChunk(ctx, key, r, -1)
	if err != nil {
		return "", 0, fmt.Errorf("blob: upload: %w", err)
	}
	if written == 0 {
		return "", 0, ErrNoData
	}
	return key, written, nil
}
