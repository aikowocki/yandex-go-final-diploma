package domain

import "time"

type Vault struct {
	ID              string
	UserID          string
	WrappedVaultKey []byte
	EncName         []byte
	Version         int64
	Deleted         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
