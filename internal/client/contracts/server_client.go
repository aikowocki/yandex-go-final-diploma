package contracts

import (
	"context"
	"io"
)

type Tokens struct {
	AccessToken  string
	RefreshToken string
	UserID       string
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

// VaultVersion — лёгкая проекция {id, version} для sync (CheckFreshness).
type VaultVersion struct {
	ID      string
	Version int64
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

// ServerSecret — полная серверная версия секрета (все три тира), которую сервер возвращает
// в деталях конфликта. Клиент расшифровывает её, чтобы показать «серверную карточку».
type ServerSecret struct {
	ID         string
	Type       int32
	Version    int64
	EncRow     []byte
	EncIndex   []byte
	EncPayload []byte
}

type ServerClient interface {
	UploadBlob(ctx context.Context, accessToken, secretID string, r io.Reader) (blobRef string, blobSize int64, err error)
	DownloadBlob(ctx context.Context, accessToken, secretID string) (io.ReadCloser, error)
	AttachBlob(ctx context.Context, accessToken, secretID string, baseVersion int64, blobRef string, blobSize int64) (newVersion int64, err error)

	Register(ctx context.Context, login string, loginCredential []byte) (Tokens, error)
	SetupEncryption(ctx context.Context, accessToken string, encKDFSalt, encKDFParams []byte) error
	Login(ctx context.Context, login string, loginCredential []byte) (LoginResult, error)
	RefreshToken(ctx context.Context, refreshToken string) (LoginResult, error)

	CreateVault(ctx context.Context, accessToken string, wrappedVaultKey, encName []byte) (vaultID string, err error)
	ListVaults(ctx context.Context, accessToken string) ([]VaultItem, error)
	CheckFreshness(ctx context.Context, accessToken string) ([]VaultVersion, error)

	// CreateSecret создаёт секрет с client-generated secretID (нужен для AAD).
	CreateSecret(ctx context.Context, accessToken, secretID, vaultID string, secretType int32, encRow, encIndex, encPayload []byte) error
	// UpdateSecret обновляет секрет с оптимистичной блокировкой. При конфликте версий
	// возвращает *ConflictError (в grpcclient) с актуальной серверной версией.
	UpdateSecret(ctx context.Context, accessToken, secretID string, baseVersion int64, encRow, encIndex, encPayload []byte) (newVersion int64, err error)
	// DeleteSecret выполняет soft-delete с оптимистичной блокировкой; при конфликте — *ConflictError.
	DeleteSecret(ctx context.Context, accessToken, secretID string, baseVersion int64) error
	ListSecretRows(ctx context.Context, accessToken, vaultID string) ([]SecretRowItem, error)
	ListSecretIndex(ctx context.Context, accessToken, vaultID string) ([]SecretIndexItem, error)
	GetSecretPayload(ctx context.Context, accessToken, secretID string) (SecretPayloadItem, error)
}

type TokenStore interface {
	Save(t Tokens) error
	Load() (Tokens, error)
	Clear() error
}
