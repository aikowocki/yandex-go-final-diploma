package auth

import (
	"encoding/json"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain"
	"github.com/aikowocki/yandex-go-final-diploma/pkg/crypto"
)

// ErrPINNotSet — PIN не установлен в текущей сессии.
var ErrPINNotSet = fmt.Errorf("auth: pin is not set")

// minPINLen — минимальная длина PIN.
const minPINLen = 4

// HasPIN сообщает, установлен ли PIN в текущей (тёплой) сессии.
func (u *UseCase) HasPIN() bool {
	return u.sess.HasPIN()
}

// SetPIN устанавливает PIN для быстрой разблокировки: выводит из PIN ключ (Argon2id) и
// оборачивает им текущий MasterKey. Обёртка хранится только в памяти сессии — при перезапуске
// процесса PIN сбрасывается. Требует разблокированной сессии (MasterKey известен).
func (u *UseCase) SetPIN(pin []byte) error {
	if len(pin) < minPINLen {
		return fmt.Errorf("auth: pin must be at least %d characters", minPINLen)
	}
	masterKey, ok := u.sess.MasterKey()
	if !ok {
		return ErrLocked
	}

	salt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("generate pin salt: %w", err)
	}
	params := crypto.DefaultParams()

	pinKey, err := u.derivePINKey(pin, salt, params)
	if err != nil {
		return err
	}

	// Оборачиваем MasterKey ключом, выведенным из PIN (envelope: WrapVaultKey(dek, kek)).
	wrapped, err := u.cipher.WrapVaultKey(masterKey, pinKey)
	if err != nil {
		return fmt.Errorf("wrap master key with pin: %w", err)
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal pin params: %w", err)
	}

	u.sess.SetPINMaterial(wrapped, salt, paramsJSON)
	return nil
}

// UnlockWithPIN разворачивает MasterKey из сохранённой PIN-обёртки и восстанавливает сессию.
// Возвращает ErrPINNotSet, если PIN не установлен, или ошибку расшифровки при неверном PIN.
func (u *UseCase) UnlockWithPIN(pin []byte) error {
	if len(pin) == 0 {
		return fmt.Errorf("auth: pin must not be empty")
	}
	wrapped, salt, paramsJSON, ok := u.sess.PINMaterial()
	if !ok {
		return ErrPINNotSet
	}

	var params crypto.Params
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return fmt.Errorf("unmarshal pin params: %w", err)
	}

	pinKey, err := u.derivePINKey(pin, salt, params)
	if err != nil {
		return err
	}

	masterKey, err := u.cipher.UnwrapVaultKey(wrapped, pinKey)
	if err != nil {
		return fmt.Errorf("invalid pin")
	}

	u.sess.SetMasterKey(masterKey)
	return nil
}

// derivePINKey выводит 32-байтный ключ из PIN через тот же Argon2id, что и MasterSeed
// (pin играет роль passphrase). KeyLen params = 32 = размер AEAD-ключа.
func (u *UseCase) derivePINKey(pin, salt []byte, params crypto.Params) ([]byte, error) {
	key, err := u.crypto.DeriveMasterSeed(domain.EncryptionPassphrase(pin), salt, params)
	if err != nil {
		return nil, fmt.Errorf("derive pin key: %w", err)
	}
	return key, nil
}
