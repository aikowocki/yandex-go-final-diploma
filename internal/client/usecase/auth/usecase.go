package auth

import (
	"context"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/session"
)

// Ключи kv-кеша для параметров KDF ветки шифрования.
const (
	kvEncKDFSalt    = "auth.enc_kdf_salt"
	kvEncKDFParams  = "auth.enc_kdf_params"
	kvAccountUserID = "auth.account_user_id"
)

type UseCase struct {
	server contracts.ServerClient
	crypto contracts.Crypto
	tokens contracts.TokenStore
	sess   *session.Session
	local  contracts.LocalStorage

	// encKDFSalt/encKDFParams — параметры KDF, полученные при Login,
	// нужны для последующего Unlock. MasterKey же живёт в общей session.Session
	encKDFSalt   []byte
	encKDFParams []byte
}

// New создаёт клиентский auth-usecase. sess — общая сессия процесса (MasterKey кладётся туда).
func New(server contracts.ServerClient, crypto contracts.Crypto, tokens contracts.TokenStore, sess *session.Session, local contracts.LocalStorage) *UseCase {
	return &UseCase{server: server, crypto: crypto, tokens: tokens, sess: sess, local: local}
}

// MasterKeySet сообщает, выведен ли MasterKey в текущей сессии.
func (u *UseCase) MasterKeySet() bool {
	return u.sess.Unlocked()
}

// EncryptionConfigured сообщает, настроено ли шифрование для аккаунта
// (известны непустые enc_kdf_salt/params — из Login/Refresh или локального кеша).
func (u *UseCase) EncryptionConfigured() bool {
	return len(u.encKDFSalt) > 0 && len(u.encKDFParams) > 0
}

// persistEncryption кеширует текущие параметры KDF локально (для офлайн-разблокировки).
// Ничего не делает, если шифрование ещё не настроено.
func (u *UseCase) persistEncryption(ctx context.Context) error {
	if !u.EncryptionConfigured() {
		return nil
	}
	if err := u.local.KVSet(ctx, kvEncKDFSalt, u.encKDFSalt); err != nil {
		return fmt.Errorf("cache kdf salt: %w", err)
	}
	if err := u.local.KVSet(ctx, kvEncKDFParams, u.encKDFParams); err != nil {
		return fmt.Errorf("cache kdf params: %w", err)
	}
	return nil
}

// reconcileAccount сверяет userID (полученный от Register/Login/Refresh) с тем, чьи данные
// сейчас лежат в локальном кеше (kvAccountUserID). Если кеш принадлежит ДРУГОМУ аккаунту,
// стирает весь локальный кеш перед тем как продолжить, чтобы не смешивать шифротексты разных
// MasterKey. Пустой кеш (первый запуск) или совпадающий userID — no-op.
func (u *UseCase) reconcileAccount(ctx context.Context, userID string) error {
	if userID == "" {
		return nil // сервер старой версии/тест без UserID — не блокируем
	}

	cached, ok, err := u.local.KVGet(ctx, kvAccountUserID)
	if err != nil {
		return fmt.Errorf("read cached account: %w", err)
	}

	if ok && string(cached) != userID {
		if err := u.local.WipeAccountData(ctx); err != nil {
			return fmt.Errorf("wipe stale account cache: %w", err)
		}
		// Сброс сессии: MasterKey/VaultKey предыдущего аккаунта больше не актуальны.
		u.sess.Lock()
		u.encKDFSalt, u.encKDFParams = nil, nil
	}

	if err := u.local.KVSet(ctx, kvAccountUserID, []byte(userID)); err != nil {
		return fmt.Errorf("remember account: %w", err)
	}
	return nil
}

// LoadCachedEncryption поднимает параметры KDF из локального кеша (офлайн-путь: когда Refresh
// недоступен). Возвращает ErrEncryptionNotSetup, если кеш пуст.
func (u *UseCase) LoadCachedEncryption(ctx context.Context) error {
	salt, ok, err := u.local.KVGet(ctx, kvEncKDFSalt)
	if err != nil {
		return err
	}
	params, okP, err := u.local.KVGet(ctx, kvEncKDFParams)
	if err != nil {
		return err
	}
	if !ok || !okP || len(salt) == 0 || len(params) == 0 {
		return ErrEncryptionNotSetup
	}
	u.encKDFSalt = salt
	u.encKDFParams = params
	return nil
}
