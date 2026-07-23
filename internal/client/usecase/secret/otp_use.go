package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

// MarkOTPCodeUsed помечает одноразовый код восстановления (по индексу 0-based в массиве
// otp_codes) как использованный.
func (u *UseCase) MarkOTPCodeUsed(ctx context.Context, vaultID, secretID string, codeIndex int) error {
	if secretID == "" {
		return ErrEmptySecretID
	}
	if codeIndex < 0 {
		return ErrInvalidOTPCodeIndex
	}

	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return err
	}

	local, ok, err := u.local.GetSecret(ctx, secretID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSecretNotFound
	}

	// Расшифровываем все три тира в generic map, чтобы не завязываться на конкретный тип секрета.
	var rowMap, indexMap, payloadMap map[string]any
	if err := decryptTiers(u, vaultKey, vaultID, secretID, local.Version, local.EncRow, local.EncIndex, local.EncPayload, &rowMap, &indexMap, &payloadMap); err != nil {
		return fmt.Errorf("decrypt tiers: %w", err)
	}

	// Находим и обновляем otp_codes[codeIndex].used = true.
	if err := markUsedInMap(payloadMap, codeIndex); err != nil {
		return err
	}

	// Перешифровываем все три тира под новую версию (baseVersion+1).
	newVersion := local.Version + 1
	encRow, err := u.cipher.EncryptStruct(vaultKey, secretAAD(vaultID, secretID, newVersion, tierRow), rowMap)
	if err != nil {
		return fmt.Errorf("re-encrypt row: %w", err)
	}
	encIndex, err := u.cipher.EncryptStruct(vaultKey, secretAAD(vaultID, secretID, newVersion, tierIndex), indexMap)
	if err != nil {
		return fmt.Errorf("re-encrypt index: %w", err)
	}
	encPayload, err := u.cipher.EncryptStruct(vaultKey, secretAAD(vaultID, secretID, newVersion, tierPayload), payloadMap)
	if err != nil {
		return fmt.Errorf("re-encrypt payload: %w", err)
	}

	_, err = u.server.UpdateSecret(ctx, token, secretID, local.Version, encRow, encIndex, encPayload)
	if err != nil {
		if errors.Is(err, grpcclient.ErrUnavailable) {
			return u.updateOffline(ctx, secretID, vaultID, local.Type, local.Version, encRow, encIndex, encPayload)
		}
		return err
	}

	return u.cacheFullSecret(ctx, secretID, vaultID, local.Type, encRow, encIndex, encPayload, newVersion, false)
}

// markUsedInMap находит otp_codes[codeIndex] в generic payload-map и ставит used=true.
func markUsedInMap(payloadMap map[string]any, codeIndex int) error {
	codesRaw, ok := payloadMap["otp_codes"]
	if !ok {
		return ErrNoOTPCodes
	}
	codes, ok := codesRaw.([]any)
	if !ok {
		return ErrNoOTPCodes
	}
	if codeIndex >= len(codes) {
		return ErrInvalidOTPCodeIndex
	}

	codeEntry, ok := codes[codeIndex].(map[string]any)
	if !ok {
		return ErrNoOTPCodes
	}
	codeEntry["used"] = true
	return nil
}

// GetOTPCodes возвращает otp_codes из payload любого типа секрета (type-agnostic: расшифровывает
// payload как map, извлекает поле otp_codes → []OTPCode через JSON re-marshal).
func (u *UseCase) GetOTPCodes(ctx context.Context, vaultID, secretID string) ([]secretcontent.OTPCode, error) {
	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return nil, err
	}

	encPayload, version, err := u.payloadCiphertext(ctx, token, secretID)
	if err != nil {
		return nil, err
	}

	var payloadMap map[string]any
	ad := secretAAD(vaultID, secretID, version, tierPayload)
	if err := u.cipher.DecryptStruct(vaultKey, ad, encPayload, &payloadMap); err != nil {
		return nil, fmt.Errorf("decrypt payload: %w", err)
	}

	codesRaw, ok := payloadMap["otp_codes"]
	if !ok || codesRaw == nil {
		return nil, nil
	}

	b, err := json.Marshal(codesRaw)
	if err != nil {
		return nil, fmt.Errorf("marshal otp_codes: %w", err)
	}
	var codes []secretcontent.OTPCode
	if err := json.Unmarshal(b, &codes); err != nil {
		return nil, fmt.Errorf("unmarshal otp_codes: %w", err)
	}
	return codes, nil
}
