package blob

import "errors"

var (
	// ErrBlobStorageDisabled — сервер собран без объектного хранилища (MinIO не настроен).
	ErrBlobStorageDisabled = errors.New("blob: object storage is not configured")
	// ErrEmptySecretID — secret id не передан.
	ErrEmptySecretID = errors.New("blob: secret id must not be empty")
	// ErrEmptyUserID — user id не передан.
	ErrEmptyUserID = errors.New("blob: user id must not be empty")
	// ErrSecretNotFound — секрет не найден.
	ErrSecretNotFound = errors.New("blob: secret not found")
	// ErrNoData — не получено ни одного чанка данных.
	ErrNoData = errors.New("blob: no data received")
)
