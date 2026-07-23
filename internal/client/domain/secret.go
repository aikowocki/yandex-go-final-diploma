package domain

// Secret - метаданные секрета в хранилище.
type Secret struct {
	ID      string
	VaultID string
	Type    SecretType
	Version int64
}
