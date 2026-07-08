package auth

import (
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
)

type UseCase struct {
	server contracts.ServerClient
	crypto contracts.Crypto
	tokens contracts.TokenStore

	// session — состояние в памяти процесса.
	session session
}

// session хранит выведенный MasterKey и параметры KDF, полученные при логине,
// чтобы последующий Unlock мог вывести ключ по тем же salt/params.
type session struct {
	masterKey    []byte
	encKDFSalt   []byte
	encKDFParams []byte
}

// New создаёт клиентский auth-usecase.
func New(server contracts.ServerClient, crypto contracts.Crypto, tokens contracts.TokenStore) *UseCase {
	return &UseCase{server: server, crypto: crypto, tokens: tokens}
}

// MasterKeySet сообщает, выведен ли MasterKey в текущей сессии.
func (u *UseCase) MasterKeySet() bool {
	return len(u.session.masterKey) > 0
}

// EncryptionConfigured сообщает, настроено ли шифрование для аккаунта
func (u *UseCase) EncryptionConfigured() bool {
	return len(u.session.encKDFSalt) > 0 && len(u.session.encKDFParams) > 0
}
