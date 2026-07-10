package contracts

import "context"

type Tokens struct {
	AccessToken  string
	RefreshToken string
}

type LoginResult struct {
	Tokens
	EncKDFSalt   []byte
	EncKDFParams []byte
}

type VaultItem struct {
	ID              string
	WrappedVaultKey []byte
	EncName         []byte
	Version         int64
}

type SecretRowItem struct {
	ID      string
	Type    int32
	Version int64
	EncRow  []byte
}

type SecretIndexItem struct {
	ID       string
	Version  int64
	EncIndex []byte
}

type SecretPayloadItem struct {
	ID         string
	Type       int32
	Version    int64
	EncPayload []byte
}

type ServerClient interface {
	Register(ctx context.Context, login string, loginCredential []byte) (Tokens, error)
	SetupEncryption(ctx context.Context, accessToken string, encKDFSalt, encKDFParams []byte) error
	Login(ctx context.Context, login string, loginCredential []byte) (LoginResult, error)
	RefreshToken(ctx context.Context, refreshToken string) (LoginResult, error)

	CreateVault(ctx context.Context, accessToken string, wrappedVaultKey, encName []byte) (vaultID string, err error)
	ListVaults(ctx context.Context, accessToken string) ([]VaultItem, error)

	CreateSecret(ctx context.Context, accessToken, vaultID string, secretType int32, encRow, encIndex, encPayload []byte) (secretID string, err error)
	ListSecretRows(ctx context.Context, accessToken, vaultID string) ([]SecretRowItem, error)
	ListSecretIndex(ctx context.Context, accessToken, vaultID string) ([]SecretIndexItem, error)
	GetSecretPayload(ctx context.Context, accessToken, secretID string) (SecretPayloadItem, error)
}

type TokenStore interface {
	Save(t Tokens) error
	Load() (Tokens, error)
	Clear() error
}
