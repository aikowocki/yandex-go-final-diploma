package blob

import "errors"

var (
	// ErrBlobStorageDisabled — сервер собран без объектного хранилища (MinIO не настроен).
	ErrBlobStorageDisabled = errors.New("blob: object storage is not configured")
	ErrEmptySecretID       = errors.New("blob: secret id must not be empty")
	ErrEmptyUserID         = errors.New("blob: user id must not be empty")
	ErrSecretNotFound      = errors.New("blob: secret not found")
	ErrNoData              = errors.New("blob: no data received")
)
