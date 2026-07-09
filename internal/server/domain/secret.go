package domain

import "time"

type Secret struct {
	ID         string
	VaultID    string
	Type       SecretType
	EncRow     []byte
	EncIndex   []byte
	EncPayload []byte
	BlobRef    *string
	BlobSize   *int64
	Version    int64
	Deleted    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
