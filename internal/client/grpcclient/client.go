package grpcclient

import (
	"context"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Client — gRPC-клиент
type Client struct {
	conn   *grpc.ClientConn
	auth   pb.AuthServiceClient
	vault  pb.VaultServiceClient
	secret pb.SecretServiceClient
	blob   pb.BlobServiceClient
}

var _ contracts.ServerClient = (*Client)(nil)

// New создаёт подключение к серверу.
func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:   conn,
		auth:   pb.NewAuthServiceClient(conn),
		vault:  pb.NewVaultServiceClient(conn),
		secret: pb.NewSecretServiceClient(conn),
		blob:   pb.NewBlobServiceClient(conn),
	}, nil
}

// Close закрывает соединение.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Ping проверяет связность с сервером.
func (c *Client) Ping(ctx context.Context) (string, error) {
	resp, err := c.auth.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		return "", mapErr(err)
	}
	return resp.GetMessage(), nil
}

// Register регистрирует пользователя. loginCredential уходит на сервер как есть (по TLS).
func (c *Client) Register(ctx context.Context, login string, loginCredential []byte) (contracts.Tokens, error) {
	resp, err := c.auth.Register(ctx, &pb.RegisterRequest{
		Login:           login,
		LoginCredential: loginCredential,
	})
	if err != nil {
		return contracts.Tokens{}, mapErr(err)
	}
	return contracts.Tokens{
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
		UserID:       resp.GetUserId(),
	}, nil
}

// SetupEncryption сохраняет на сервере enc_kdf_salt/enc_kdf_params. Требует access-токен
// (кладётся в metadata как Bearer). MasterKey/EncryptionPassphrase не отправляется.
func (c *Client) SetupEncryption(ctx context.Context, accessToken string, encKDFSalt, encKDFParams, encMasterKey []byte) error {
	ctx = withBearer(ctx, accessToken)
	_, err := c.auth.SetupEncryption(ctx, &pb.SetupEncryptionRequest{
		EncKdfSalt:   encKDFSalt,
		EncKdfParams: encKDFParams,
		EncMasterKey: encMasterKey,
	})
	return mapErr(err)
}

// StoreRecoveryCodes сохраняет recovery codes на сервере.
func (c *Client) StoreRecoveryCodes(ctx context.Context, accessToken string, codes []contracts.RecoveryCodeEntry) error {
	ctx = withBearer(ctx, accessToken)
	pbCodes := make([]*pb.RecoveryCodeEntry, 0, len(codes))
	for _, code := range codes {
		pbCodes = append(pbCodes, &pb.RecoveryCodeEntry{
			CodeId:       code.CodeID,
			EncMasterKey: code.EncMasterKey,
		})
	}
	_, err := c.auth.StoreRecoveryCodes(ctx, &pb.StoreRecoveryCodesRequest{Codes: pbCodes})
	return mapErr(err)
}

// GetRecoveryBlob возвращает зашифрованный MasterKey для указанного code_id.
func (c *Client) GetRecoveryBlob(ctx context.Context, accessToken, codeID string) ([]byte, error) {
	ctx = withBearer(ctx, accessToken)
	resp, err := c.auth.GetRecoveryBlob(ctx, &pb.GetRecoveryBlobRequest{CodeId: codeID})
	if err != nil {
		return nil, mapErr(err)
	}
	return resp.GetEncMasterKey(), nil
}

// MarkRecoveryCodeUsed помечает recovery code как использованный.
func (c *Client) MarkRecoveryCodeUsed(ctx context.Context, accessToken, codeID string) error {
	ctx = withBearer(ctx, accessToken)
	_, err := c.auth.MarkRecoveryCodeUsed(ctx, &pb.MarkRecoveryCodeUsedRequest{CodeId: codeID})
	return mapErr(err)
}

// Login аутентифицирует пользователя и возвращает токены + параметры клиентского KDF.
func (c *Client) Login(ctx context.Context, login string, loginCredential []byte) (contracts.LoginResult, error) {
	resp, err := c.auth.Login(ctx, &pb.LoginRequest{
		Login:           login,
		LoginCredential: loginCredential,
	})
	if err != nil {
		return contracts.LoginResult{}, mapErr(err)
	}
	return contracts.LoginResult{
		Tokens: contracts.Tokens{
			AccessToken:  resp.GetAccessToken(),
			RefreshToken: resp.GetRefreshToken(),
			UserID:       resp.GetUserId(),
		},
		EncKDFSalt:   resp.GetEncKdfSalt(),
		EncKDFParams: resp.GetEncKdfParams(),
		EncMasterKey: resp.GetEncMasterKey(),
	}, nil
}

// RefreshToken обновляет пару токенов по refresh-токену.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (contracts.LoginResult, error) {
	resp, err := c.auth.RefreshToken(ctx, &pb.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return contracts.LoginResult{}, mapErr(err)
	}
	return contracts.LoginResult{
		Tokens: contracts.Tokens{
			AccessToken:  resp.GetAccessToken(),
			RefreshToken: resp.GetRefreshToken(),
			UserID:       resp.GetUserId(),
		},
		EncKDFSalt:   resp.GetEncKdfSalt(),
		EncKDFParams: resp.GetEncKdfParams(),
		EncMasterKey: resp.GetEncMasterKey(),
	}, nil
}

// --- VaultService ---

// CreateVault создаёт новую папку на сервере и возвращает её id.
func (c *Client) CreateVault(ctx context.Context, accessToken string, wrappedVaultKey, encName []byte) (string, error) {
	ctx = withBearer(ctx, accessToken)
	resp, err := c.vault.CreateVault(ctx, &pb.CreateVaultRequest{
		WrappedVaultKey: wrappedVaultKey,
		EncName:         encName,
	})
	if err != nil {
		return "", mapErr(err)
	}
	return resp.GetVaultId(), nil
}

// ListVaults возвращает список папок пользователя с сервера.
func (c *Client) ListVaults(ctx context.Context, accessToken string) ([]contracts.VaultItem, error) {
	ctx = withBearer(ctx, accessToken)
	resp, err := c.vault.ListVaults(ctx, &pb.ListVaultsRequest{})
	if err != nil {
		return nil, mapErr(err)
	}
	items := make([]contracts.VaultItem, 0, len(resp.GetVaults()))
	for _, v := range resp.GetVaults() {
		items = append(items, contracts.VaultItem{
			ID:              v.GetVaultId(),
			WrappedVaultKey: v.GetWrappedVaultKey(),
			EncName:         v.GetEncName(),
			Version:         v.GetVersion(),
		})
	}
	return items, nil
}

// CheckFreshness возвращает {id, version} для всех папок — используется sync, чтобы понять,
// какие папки устарели локально.
func (c *Client) CheckFreshness(ctx context.Context, accessToken string) ([]contracts.VaultVersion, error) {
	ctx = withBearer(ctx, accessToken)
	resp, err := c.vault.CheckFreshness(ctx, &pb.CheckFreshnessRequest{})
	if err != nil {
		return nil, mapErr(err)
	}
	items := make([]contracts.VaultVersion, 0, len(resp.GetVaults()))
	for _, v := range resp.GetVaults() {
		items = append(items, contracts.VaultVersion{
			ID:      v.GetVaultId(),
			Version: v.GetVersion(),
		})
	}
	return items, nil
}

// --- SecretService ---

// CreateSecret создаёт секрет на сервере с client-generated id.
func (c *Client) CreateSecret(ctx context.Context, accessToken, secretID, vaultID string, secretType int32, encRow, encIndex, encPayload []byte) error {
	ctx = withBearer(ctx, accessToken)
	_, err := c.secret.CreateSecret(ctx, &pb.CreateSecretRequest{
		SecretId:   secretID,
		VaultId:    vaultID,
		Type:       pb.SecretType(secretType),
		EncRow:     encRow,
		EncIndex:   encIndex,
		EncPayload: encPayload,
	})
	return mapErr(err)
}

// UpdateSecret обновляет секрет на сервере с оптимистичной блокировкой по версии.
func (c *Client) UpdateSecret(ctx context.Context, accessToken, secretID string, baseVersion int64, encRow, encIndex, encPayload []byte) (int64, error) {
	ctx = withBearer(ctx, accessToken)
	resp, err := c.secret.UpdateSecret(ctx, &pb.UpdateSecretRequest{
		SecretId:    secretID,
		BaseVersion: baseVersion,
		EncRow:      encRow,
		EncIndex:    encIndex,
		EncPayload:  encPayload,
	})
	if err != nil {
		if conflict := conflictFromStatus(err); conflict != nil {
			return 0, conflict
		}
		return 0, mapErr(err)
	}
	return resp.GetVersion(), nil
}

// DeleteSecret выполняет soft-delete секрета на сервере с оптимистичной блокировкой по версии.
func (c *Client) DeleteSecret(ctx context.Context, accessToken, secretID string, baseVersion int64) error {
	ctx = withBearer(ctx, accessToken)
	_, err := c.secret.DeleteSecret(ctx, &pb.DeleteSecretRequest{
		SecretId:    secretID,
		BaseVersion: baseVersion,
	})
	if err != nil {
		if conflict := conflictFromStatus(err); conflict != nil {
			return conflict
		}
		return mapErr(err)
	}
	return nil
}

// ListSecretRows возвращает Tier 1 (row) всех секретов папки.
func (c *Client) ListSecretRows(ctx context.Context, accessToken, vaultID string) ([]contracts.SecretRowItem, error) {
	ctx = withBearer(ctx, accessToken)
	resp, err := c.secret.ListRow(ctx, &pb.ListRowRequest{VaultId: vaultID})
	if err != nil {
		return nil, mapErr(err)
	}
	items := make([]contracts.SecretRowItem, 0, len(resp.GetSecrets()))
	for _, s := range resp.GetSecrets() {
		items = append(items, contracts.SecretRowItem{
			ID:      s.GetSecretId(),
			Type:    int32(s.GetType()),
			Version: s.GetVersion(),
			EncRow:  s.GetEncRow(),
		})
	}
	return items, nil
}

// ListSecretIndex возвращает Tier 2 (index) всех секретов папки.
func (c *Client) ListSecretIndex(ctx context.Context, accessToken, vaultID string) ([]contracts.SecretIndexItem, error) {
	ctx = withBearer(ctx, accessToken)
	resp, err := c.secret.ListIndex(ctx, &pb.ListIndexRequest{VaultId: vaultID})
	if err != nil {
		return nil, mapErr(err)
	}
	items := make([]contracts.SecretIndexItem, 0, len(resp.GetSecrets()))
	for _, s := range resp.GetSecrets() {
		items = append(items, contracts.SecretIndexItem{
			ID:       s.GetSecretId(),
			Version:  s.GetVersion(),
			EncIndex: s.GetEncIndex(),
		})
	}
	return items, nil
}

// GetSecretPayload возвращает Tier 3 (payload) секрета по id.
func (c *Client) GetSecretPayload(ctx context.Context, accessToken, secretID string) (contracts.SecretPayloadItem, error) {
	ctx = withBearer(ctx, accessToken)
	resp, err := c.secret.GetPayload(ctx, &pb.GetPayloadRequest{SecretId: secretID})
	if err != nil {
		return contracts.SecretPayloadItem{}, mapErr(err)
	}
	return contracts.SecretPayloadItem{
		ID:         resp.GetSecretId(),
		Type:       int32(resp.GetType()),
		Version:    resp.GetVersion(),
		EncPayload: resp.GetEncPayload(),
	}, nil
}

func withBearer(ctx context.Context, accessToken string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+accessToken)
}
