package secret

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
)

type AttachBlobParams struct {
	UserID      string
	SecretID    string
	BaseVersion int64
	BlobRef     string
	BlobSize    int64
}

func (u *UseCase) AttachBlob(ctx context.Context, params AttachBlobParams) (int64, error) {
	if params.UserID == "" {
		return 0, ErrEmptyUserID
	}
	if params.SecretID == "" {
		return 0, ErrEmptySecretID
	}
	if params.BlobRef == "" {
		return 0, ErrEmptyBlobRef
	}

	var newVersion int64
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		current, err := u.secrets.GetForUpdate(ctx, params.SecretID, params.UserID)
		if err != nil {
			return err
		}
		if current.Deleted {
			return ErrSecretNotFound
		}
		if current.Type != domain.SecretTypeBinary {
			return ErrNotBinarySecret
		}
		if current.Version != params.BaseVersion {
			return &ErrConflict{Current: current}
		}

		v, err := u.secrets.AttachBlob(ctx, params.SecretID, params.BlobRef, params.BlobSize)
		if err != nil {
			return err
		}
		if err := u.secrets.BumpVaultVersion(ctx, current.VaultID); err != nil {
			return err
		}
		newVersion = v
		return nil
	})
	if err != nil {
		return 0, err
	}
	return newVersion, nil
}
