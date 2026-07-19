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
	kvEncMasterKey  = "auth.enc_master_key"
	kvAccountUserID = "auth.account_user_id"
	kvAccountLogin  = "auth.account_login"
)

// UseCase реализует клиентские сценарии аутентификации и разблокировки шифрования.
type UseCase struct {
	server contracts.ServerClient
	crypto contracts.Crypto
	cipher contracts.Cipher
	tokens contracts.TokenStore
	sess   *session.Session
	local  contracts.LocalStorage

	// encKDFSalt/encKDFParams — параметры вывода KEK из passphrase, полученные при Login.
	// encMasterKey — случайный MasterKey, обёрнутый KEK; при Unlock разворачивается KEK'ом.
	// Сам MasterKey живёт в общей session.Session.
	encKDFSalt   []byte
	encKDFParams []byte
	encMasterKey []byte
}

// New создаёт клиентский auth-usecase. sess — общая сессия процесса (MasterKey кладётся туда).
func New(server contracts.ServerClient, crypto contracts.Crypto, cipher contracts.Cipher, tokens contracts.TokenStore, sess *session.Session, local contracts.LocalStorage) *UseCase {
	return &UseCase{server: server, crypto: crypto, cipher: cipher, tokens: tokens, sess: sess, local: local}
}

// MasterKeySet сообщает, выведен ли MasterKey в текущей сессии.
func (u *UseCase) MasterKeySet() bool {
	return u.sess.Unlocked()
}

// EncryptionConfigured сообщает, настроено ли шифрование для аккаунта
// (известны непустые enc_kdf_salt/params/enc_master_key — из Login/Refresh или локального кеша).
func (u *UseCase) EncryptionConfigured() bool {
	return len(u.encKDFSalt) > 0 && len(u.encKDFParams) > 0 && len(u.encMasterKey) > 0
}

// CurrentUserID возвращает userID текущего аккаунта из локального кеша (для отображения
// в UI, например меню «Юзер» в TUI). Пустая строка, если кеш пуст (например сразу после
// установки, до первого Login/Register).
func (u *UseCase) CurrentUserID(ctx context.Context) string {
	id, ok, err := u.local.KVGet(ctx, kvAccountUserID)
	if err != nil || !ok {
		return ""
	}
	return string(id)
}

// CurrentLogin возвращает login текущего аккаунта из локального кеша.
func (u *UseCase) CurrentLogin(ctx context.Context) string {
	login, ok, err := u.local.KVGet(ctx, kvAccountLogin)
	if err != nil || !ok {
		return ""
	}
	return string(login)
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
	if err := u.local.KVSet(ctx, kvEncMasterKey, u.encMasterKey); err != nil {
		return fmt.Errorf("cache enc master key: %w", err)
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
		u.encKDFSalt, u.encKDFParams, u.encMasterKey = nil, nil, nil
	}

	if err := u.local.KVSet(ctx, kvAccountUserID, []byte(userID)); err != nil {
		return fmt.Errorf("remember account: %w", err)
	}
	return nil
}

// LoadCachedEncryption поднимает параметры KDF + enc_master_key из локального кеша
// (офлайн-путь: когда Refresh недоступен). Возвращает ErrEncryptionNotSetup, если кеш пуст.
func (u *UseCase) LoadCachedEncryption(ctx context.Context) error {
	salt, ok, err := u.local.KVGet(ctx, kvEncKDFSalt)
	if err != nil {
		return err
	}
	params, okP, err := u.local.KVGet(ctx, kvEncKDFParams)
	if err != nil {
		return err
	}
	encMK, okMK, err := u.local.KVGet(ctx, kvEncMasterKey)
	if err != nil {
		return err
	}
	if !ok || !okP || !okMK || len(salt) == 0 || len(params) == 0 || len(encMK) == 0 {
		return ErrEncryptionNotSetup
	}
	u.encKDFSalt = salt
	u.encKDFParams = params
	u.encMasterKey = encMK
	return nil
}
