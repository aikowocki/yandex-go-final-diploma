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
	conn *grpc.ClientConn
	auth pb.AuthServiceClient
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
		conn: conn,
		auth: pb.NewAuthServiceClient(conn),
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
	}, nil
}

// SetupEncryption сохраняет на сервере enc_kdf_salt/enc_kdf_params. Требует access-токен
// (кладётся в metadata как Bearer). MasterKey/EncryptionPassphrase не отправляется.
func (c *Client) SetupEncryption(ctx context.Context, accessToken string, encKDFSalt, encKDFParams []byte) error {
	ctx = withBearer(ctx, accessToken)
	_, err := c.auth.SetupEncryption(ctx, &pb.SetupEncryptionRequest{
		EncKdfSalt:   encKDFSalt,
		EncKdfParams: encKDFParams,
	})
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
		},
		EncKDFSalt:   resp.GetEncKdfSalt(),
		EncKDFParams: resp.GetEncKdfParams(),
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
		},
		EncKDFSalt:   resp.GetEncKdfSalt(),
		EncKDFParams: resp.GetEncKdfParams(),
	}, nil
}

func withBearer(ctx context.Context, accessToken string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+accessToken)
}
