package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"golang.org/x/crypto/hkdf"
)

const (
	recoveryCodeBytes = 16 // 128 бит энтропии
	recoveryCodeCount = 5
	recoveryHKDFInfo  = "gophkeeper-recovery-v1"
)

// GenerateRecoveryCodes генерирует 5 recovery кодов, шифрует MasterKey каждым из них
// и сохраняет на сервере. Возвращает plain-text коды для показа пользователю.
func (u *UseCase) GenerateRecoveryCodes(ctx context.Context) ([]string, error) {
	masterKey, ok := u.sess.MasterKey()
	if !ok {
		return nil, ErrLocked
	}

	tokens, err := u.tokens.Load()
	if err != nil {
		return nil, fmt.Errorf("load tokens: %w", err)
	}

	codes := make([]string, 0, recoveryCodeCount)
	entries := make([]contracts.RecoveryCodeEntry, 0, recoveryCodeCount)

	for i := 0; i < recoveryCodeCount; i++ {
		// Генерируем случайный код (16 байт → hex).
		raw := make([]byte, recoveryCodeBytes)
		if _, err := rand.Read(raw); err != nil {
			return nil, fmt.Errorf("generate recovery code: %w", err)
		}
		code := hex.EncodeToString(raw) // 32 hex символа

		// code_id = SHA256(code)[:8] hex (для поиска на сервере).
		codeID := recoveryCodeID(code)

		// Derive recovery_key из кода через HKDF.
		recoveryKey, err := deriveRecoveryKey(code)
		if err != nil {
			return nil, err
		}

		// Шифруем MasterKey recovery_key'ом (reuse WrapVaultKey — same AEAD).
		encMK, err := u.cipher.WrapVaultKey(masterKey, recoveryKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt master key with recovery code: %w", err)
		}

		codes = append(codes, formatRecoveryCode(code))
		entries = append(entries, contracts.RecoveryCodeEntry{
			CodeID:       codeID,
			EncMasterKey: encMK,
		})
	}

	// Сохраняем на сервере.
	if err := u.server.StoreRecoveryCodes(ctx, tokens.AccessToken, entries); err != nil {
		return nil, fmt.Errorf("store recovery codes: %w", err)
	}

	return codes, nil
}

// RecoverWithCode восстанавливает MasterKey из recovery code, устанавливает его в сессию.
// После этого пользователь может задать новый мастер-пароль (SetupEncryption).
func (u *UseCase) RecoverWithCode(ctx context.Context, code string) error {
	// Нормализуем код (убираем дефисы/пробелы).
	code = normalizeCode(code)

	tokens, err := u.tokens.Load()
	if err != nil {
		return fmt.Errorf("load tokens: %w", err)
	}

	codeID := recoveryCodeID(code)

	// Получаем зашифрованный MasterKey с сервера.
	encMK, err := u.server.GetRecoveryBlob(ctx, tokens.AccessToken, codeID)
	if err != nil {
		return fmt.Errorf("get recovery blob: %w", err)
	}

	// Derive recovery_key и расшифровываем.
	recoveryKey, err := deriveRecoveryKey(code)
	if err != nil {
		return err
	}

	masterKey, err := u.cipher.UnwrapVaultKey(encMK, recoveryKey)
	if err != nil {
		return fmt.Errorf("invalid recovery code (decrypt failed)")
	}

	// Помечаем код как использованный.
	_ = u.server.MarkRecoveryCodeUsed(ctx, tokens.AccessToken, codeID)

	// Устанавливаем MasterKey в сессию.
	u.sess.SetMasterKey(masterKey)
	return nil
}

// recoveryCodeID возвращает первые 16 hex символов SHA256(code) — уникальный ID для поиска.
func recoveryCodeID(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:8])
}

// deriveRecoveryKey выводит 32-байтный ключ из recovery code через HKDF-SHA512.
func deriveRecoveryKey(code string) ([]byte, error) {
	hkdfReader := hkdf.New(sha512.New, []byte(code), nil, []byte(recoveryHKDFInfo))
	key := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("derive recovery key: %w", err)
	}
	return key, nil
}

// formatRecoveryCode форматирует hex-строку как XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX.
func formatRecoveryCode(hex string) string {
	var parts []string
	for i := 0; i < len(hex); i += 4 {
		end := i + 4
		if end > len(hex) {
			end = len(hex)
		}
		parts = append(parts, hex[i:end])
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "-"
		}
		result += p
	}
	return result
}

// normalizeCode убирает дефисы и пробелы из введённого кода.
func normalizeCode(code string) string {
	var out []byte
	for _, c := range code {
		if c != '-' && c != ' ' {
			out = append(out, byte(c))
		}
	}
	return string(out)
}
