package secret

// Экспорт внутренних помощников для white-box использования во внешних тестах пакета.

// SecretAAD открывает secretAAD для тестов (сборка того же AAD-контекста, что и продакшн-код).
func SecretAAD(vaultID, secretID string, version int64, tier string) []byte {
	return secretAAD(vaultID, secretID, version, tier)
}

// Метки тиров для тестов.
const (
	TierRow     = tierRow
	TierIndex   = tierIndex
	TierPayload = tierPayload
)
