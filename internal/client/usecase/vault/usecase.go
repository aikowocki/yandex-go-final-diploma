package vault

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
)

type UseCase struct {
	server contracts.ServerClient
	cipher contracts.Cipher
	tokens contracts.TokenStore
	sess   *session.Session
	local  contracts.LocalStorage
}

func New(server contracts.ServerClient, cipher contracts.Cipher, tokens contracts.TokenStore, sess *session.Session, local contracts.LocalStorage) *UseCase {
	return &UseCase{server: server, cipher: cipher, tokens: tokens, sess: sess, local: local}
}

type DecryptedVault struct {
	ID      string
	Name    string
	Version int64
}

func (u *UseCase) accessToken() (string, error) {
	tokens, err := u.tokens.Load()
	if err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}
