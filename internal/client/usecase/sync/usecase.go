package sync

import (
	"errors"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

type UseCase struct {
	server contracts.ServerClient
	local  contracts.LocalStorage
	tokens contracts.TokenStore
}

func New(server contracts.ServerClient, local contracts.LocalStorage, tokens contracts.TokenStore) *UseCase {
	return &UseCase{server: server, local: local, tokens: tokens}
}

func (u *UseCase) accessToken() (string, error) {
	tokens, err := u.tokens.Load()
	if err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}

// isOffline сообщает, что ошибка вызвана недоступностью сети/сервера (Unavailable/DeadlineExceeded).
func isOffline(err error) bool {
	return errors.Is(err, grpcclient.ErrUnavailable)
}
