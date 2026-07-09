package domain

type Secret struct {
	ID      string
	VaultID string
	Type    SecretType
	Version int64
}
