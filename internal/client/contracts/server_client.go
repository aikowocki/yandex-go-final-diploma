package contracts

import (
	"context"
	"io"
)

// Tokens — пара JWT-токенов (access/refresh) и id пользователя.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	UserID       string
}

// LoginResult — результат Login/RefreshToken: токены + параметры клиентского KDF, нужные
// для разворачивания MasterKey (EncMasterKey зашифрован под MasterSeed из Argon2id).
type LoginResult struct {
	Tokens
	EncKDFSalt   []byte
	EncKDFParams []byte
	EncMasterKey []byte
}

// RecoveryCodeEntry — одна запись recovery code для отправки на сервер.
type RecoveryCodeEntry struct {
	CodeID       string
	EncMasterKey []byte
}

// VaultItem — папка, как её возвращает сервер (ListVaults/CreateVault).
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

// SecretRowItem — Tier 1 (row) секрета, как его возвращает сервер (ListSecretRows).
type SecretRowItem struct {
	ID      string
	Type    int32
	Version int64
	EncRow  []byte
}

// SecretIndexItem — Tier 2 (index) секрета, как его возвращает сервер (ListSecretIndex).
type SecretIndexItem struct {
	ID       string
	Version  int64
	EncIndex []byte
}

// SecretPayloadItem — Tier 3 (payload) секрета, как его возвращает сервер (GetSecretPayload).
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

// ServerClient — контракт транспортного слоя (gRPC) до сервера gophkeeper.
type ServerClient interface {
	UploadBlob(ctx context.Context, accessToken, secretID string, r io.Reader) (blobRef string, blobSize int64, err error)
	DownloadBlob(ctx context.Context, accessToken, secretID string) (io.ReadCloser, error)
	AttachBlob(ctx context.Context, accessToken, secretID string, baseVersion int64, blobRef string, blobSize int64) (newVersion int64, err error)

	Register(ctx context.Context, login string, loginCredential []byte) (Tokens, error)
	SetupEncryption(ctx context.Context, accessToken string, encKDFSalt, encKDFParams, encMasterKey []byte) error
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

	// Recovery codes
	StoreRecoveryCodes(ctx context.Context, accessToken string, codes []RecoveryCodeEntry) error
	GetRecoveryBlob(ctx context.Context, accessToken, codeID string) ([]byte, error)
	MarkRecoveryCodeUsed(ctx context.Context, accessToken, codeID string) error
}

// TokenStore — локальное хранилище access/refresh токенов.
type TokenStore interface {
	Save(t Tokens) error
	Load() (Tokens, error)
	Clear() error
}
