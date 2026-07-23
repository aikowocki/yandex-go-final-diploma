package secret

import (
	"encoding/json"
	"fmt"
)

// maxEncIndexSize — лимит размера enc_index ДО шифрования.
const maxEncIndexSize = 8 * 1024

// checkIndexSize возвращает ErrIndexTooLarge, если сериализованный (JSON, до шифрования) индекс
// превышает maxEncIndexSize. plaintext — результат json.Marshal(indexStruct).
func checkIndexSize(plaintext []byte) error {
	if len(plaintext) > maxEncIndexSize {
		return fmt.Errorf("%w: %d bytes (limit %d)", ErrIndexTooLarge, len(plaintext), maxEncIndexSize)
	}
	return nil
}

// encryptTiers шифрует row/index/payload под VaultKey с AAD-контекстом (vault|secret|version|tier)
// общим для всех типов секретов. Перед шифрованием индекса проверяет лимит его размера.
func encryptTiers[R, I, P any](u *UseCase, vaultKey []byte, vaultID, secretID string, version int64, row R, index I, payload P) (encRow, encIndex, encPayload []byte, err error) {
	encRow, err = u.cipher.EncryptStruct(vaultKey, secretAAD(vaultID, secretID, version, tierRow), row)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encrypt row: %w", err)
	}

	indexPlain, err := json.Marshal(index)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal index: %w", err)
	}
	if err := checkIndexSize(indexPlain); err != nil {
		return nil, nil, nil, err
	}
	encIndex, err = u.cipher.EncryptStruct(vaultKey, secretAAD(vaultID, secretID, version, tierIndex), index)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encrypt index: %w", err)
	}

	encPayload, err = u.cipher.EncryptStruct(vaultKey, secretAAD(vaultID, secretID, version, tierPayload), payload)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encrypt payload: %w", err)
	}
	return encRow, encIndex, encPayload, nil
}

// decryptTiers расшифровывает три тира секрета в переданные указатели. Пустые блобы (nil/len==0)
// пропускаются молча (например enc_payload у binary до подгрузки).
func decryptTiers(u *UseCase, vaultKey []byte, vaultID, secretID string, version int64, encRow, encIndex, encPayload []byte, row, index, payload any) error {
	if len(encRow) > 0 {
		if err := u.cipher.DecryptStruct(vaultKey, secretAAD(vaultID, secretID, version, tierRow), encRow, row); err != nil {
			return fmt.Errorf("decrypt row: %w", err)
		}
	}
	if len(encIndex) > 0 {
		if err := u.cipher.DecryptStruct(vaultKey, secretAAD(vaultID, secretID, version, tierIndex), encIndex, index); err != nil {
			return fmt.Errorf("decrypt index: %w", err)
		}
	}
	if len(encPayload) > 0 {
		if err := u.cipher.DecryptStruct(vaultKey, secretAAD(vaultID, secretID, version, tierPayload), encPayload, payload); err != nil {
			return fmt.Errorf("decrypt payload: %w", err)
		}
	}
	return nil
}
