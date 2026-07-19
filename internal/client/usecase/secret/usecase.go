package secret

import (
	"context"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
)

// UseCase реализует клиентские сценарии работы с секретами (создание, обновление, чтение,
// синхронизация и разрешение конфликтов версий).
type UseCase struct {
	server  contracts.ServerClient
	cipher  contracts.Cipher
	tokens  contracts.TokenStore
	sess    *session.Session
	local   contracts.LocalStorage
	dataDir string
}

// New создаёт клиентский secret-usecase.
func New(server contracts.ServerClient, cipher contracts.Cipher, tokens contracts.TokenStore, sess *session.Session, local contracts.LocalStorage, dataDir string) *UseCase {
	return &UseCase{server: server, cipher: cipher, tokens: tokens, sess: sess, local: local, dataDir: dataDir}
}

func (u *UseCase) accessToken() (string, error) {
	tokens, err := u.tokens.Load()
	if err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}

// vaultKey возвращает VaultKey открытой папки из сессии либо ErrVaultLocked.
func (u *UseCase) vaultKey(vaultID string) ([]byte, error) {
	vk, ok := u.sess.VaultKey(vaultID)
	if !ok {
		return nil, ErrVaultLocked
	}
	return vk, nil
}

// LocalVersion возвращает версию секрета из локального кеша и признак его наличия.
func (u *UseCase) LocalVersion(ctx context.Context, secretID string) (version int64, ok bool, err error) {
	sec, ok, err := u.local.GetSecret(ctx, secretID)
	if err != nil || !ok {
		return 0, ok, err
	}
	return sec.Version, true, nil
}

func (u *UseCase) vaultContext(vaultID string) (vaultKey []byte, token string, err error) {
	if vaultID == "" {
		return nil, "", ErrEmptyVaultID
	}
	if vaultKey, err = u.vaultKey(vaultID); err != nil {
		return nil, "", err
	}
	if token, err = u.accessToken(); err != nil {
		return nil, "", err
	}
	return vaultKey, token, nil
}
