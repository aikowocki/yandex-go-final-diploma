package secret

import (
	"context"
	"fmt"
)

// TypedRow — обёртка над расшифрованной Tier 2a-строкой произвольного типа секрета.
type TypedRow[R any] struct {
	ID      string
	Version int64
	Row     R
}

// listRowsTyped — обобщённый ListRow для конкретного типа секрета (только строки этого типа
// из локального кеша папки, без сетевых вызовов).
func listRowsTyped[R any](ctx context.Context, u *UseCase, vaultID string, secretType int32) ([]TypedRow[R], error) {
	if vaultID == "" {
		return nil, ErrEmptyVaultID
	}
	vaultKey, err := u.vaultKey(vaultID)
	if err != nil {
		return nil, err
	}

	items, err := u.local.ListSecretsByVault(ctx, vaultID)
	if err != nil {
		return nil, err
	}

	result := make([]TypedRow[R], 0, len(items))
	for _, it := range items {
		if it.Type != secretType {
			continue
		}
		var row R
		ad := secretAAD(vaultID, it.ID, it.Version, tierRow)
		if err := u.cipher.DecryptStruct(vaultKey, ad, it.EncRow, &row); err != nil {
			return nil, fmt.Errorf("decrypt row: %w", err)
		}
		result = append(result, TypedRow[R]{ID: it.ID, Version: it.Version, Row: row})
	}
	return result, nil
}

// getPayloadTyped — обобщённый GetPayload для конкретного типа секрета.
func getPayloadTyped[P any](ctx context.Context, u *UseCase, vaultID, secretID string) (id string, version int64, payload P, err error) {
	if secretID == "" {
		return "", 0, payload, ErrEmptySecretID
	}
	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return "", 0, payload, err
	}

	encPayload, ver, err := u.payloadCiphertext(ctx, token, secretID)
	if err != nil {
		return "", 0, payload, err
	}

	ad := secretAAD(vaultID, secretID, ver, tierPayload)
	if err := u.cipher.DecryptStruct(vaultKey, ad, encPayload, &payload); err != nil {
		return "", 0, payload, fmt.Errorf("decrypt payload: %w", err)
	}
	return secretID, ver, payload, nil
}

// TypedDetail — полная расшифрованная карточка произвольного типа секрета (все тиры вместе).
type TypedDetail[R, I, P any] struct {
	ID      string
	Version int64
	Row     R
	Index   I
	Payload P
}

// getDetailTyped — обобщённый GetDetail
func getDetailTyped[R, I, P any](ctx context.Context, u *UseCase, vaultID, secretID string) (TypedDetail[R, I, P], error) {
	var detail TypedDetail[R, I, P]

	vaultKey, err := u.vaultKey(vaultID)
	if err != nil {
		return detail, err
	}

	id, version, payload, err := getPayloadTyped[P](ctx, u, vaultID, secretID)
	if err != nil {
		return detail, err
	}
	detail = TypedDetail[R, I, P]{ID: id, Version: version, Payload: payload}

	local, ok, err := u.local.GetSecret(ctx, secretID)
	if err != nil {
		return detail, err
	}
	if !ok {
		return detail, nil
	}

	ad := secretAAD(vaultID, secretID, local.Version, tierRow)
	if err := u.cipher.DecryptStruct(vaultKey, ad, local.EncRow, &detail.Row); err != nil {
		return detail, fmt.Errorf("decrypt row: %w", err)
	}

	if !local.IndexLoaded {
		if err := u.LoadIndexes(ctx, vaultID); err == nil {
			local, ok, _ = u.local.GetSecret(ctx, secretID)
		}
	}
	if ok && local.IndexLoaded && len(local.EncIndex) > 0 {
		idxAD := secretAAD(vaultID, secretID, local.Version, tierIndex)
		if err := u.cipher.DecryptStruct(vaultKey, idxAD, local.EncIndex, &detail.Index); err != nil {
			return detail, fmt.Errorf("decrypt index: %w", err)
		}
	}
	return detail, nil
}
