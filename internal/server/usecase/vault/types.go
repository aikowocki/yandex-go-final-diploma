package vault

type CreateVaultParams struct {
	UserID          string
	WrappedVaultKey []byte
	EncName         []byte
}

// VaultTier1 — проекция папки для списка (Tier 1)
type VaultTier1 struct {
	ID              string
	WrappedVaultKey []byte
	EncName         []byte
	Version         int64
}
