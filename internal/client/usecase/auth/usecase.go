package auth

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
)

type UseCase struct {
	server contracts.ServerClient
	crypto contracts.Crypto
	tokens contracts.TokenStore
	sess   *session.Session

	// encKDFSalt/encKDFParams — параметры KDF, полученные при Login,
	// нужны для последующего Unlock. MasterKey же живёт в общей session.Session
	encKDFSalt   []byte
	encKDFParams []byte
}

// New создаёт клиентский auth-usecase. sess — общая сессия процесса (MasterKey кладётся туда).
func New(server contracts.ServerClient, crypto contracts.Crypto, tokens contracts.TokenStore, sess *session.Session) *UseCase {
	return &UseCase{server: server, crypto: crypto, tokens: tokens, sess: sess}
}

// MasterKeySet сообщает, выведен ли MasterKey в текущей сессии.
func (u *UseCase) MasterKeySet() bool {
	return u.sess.Unlocked()
}

// EncryptionConfigured сообщает, настроено ли шифрование для аккаунта
// (сервер прислал непустые enc_kdf_salt/params при последнем Login).
func (u *UseCase) EncryptionConfigured() bool {
	return len(u.encKDFSalt) > 0 && len(u.encKDFParams) > 0
}
