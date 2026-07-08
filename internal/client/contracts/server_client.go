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

type ServerClient interface {
	Register(ctx context.Context, login string, loginCredential []byte) (Tokens, error)
	SetupEncryption(ctx context.Context, accessToken string, encKDFSalt, encKDFParams []byte) error
	Login(ctx context.Context, login string, loginCredential []byte) (LoginResult, error)
	RefreshToken(ctx context.Context, refreshToken string) (LoginResult, error)
}

type TokenStore interface {
	Save(t Tokens) error
	Load() (Tokens, error)
	Clear() error
}
