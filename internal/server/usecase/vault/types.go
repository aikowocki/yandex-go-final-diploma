package vault

type CreateParams struct {
	UserID          string
	WrappedVaultKey []byte
	EncName         []byte
}

// Tier1 — проекция папки для списка (Tier 1)
type Tier1 struct {
	ID              string
	WrappedVaultKey []byte
	EncName         []byte
	Version         int64
}

// Version — пара {id, version} для клиентского sync (CheckFreshness).
type Version struct {
	ID      string
	Version int64
}
