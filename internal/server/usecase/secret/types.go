package secret

import "github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"

// CreateParams параметры создания секрета.
type CreateParams struct {
	UserID     string
	VaultID    string
	SecretID   string // UUID генерируется клиентом.
	Type       domain.SecretType
	EncRow     []byte
	EncIndex   []byte
	EncPayload []byte
}

// UpdateParams параметры обновления секрета.
type UpdateParams struct {
	UserID      string
	SecretID    string
	BaseVersion int64
	EncRow      []byte
	EncIndex    []byte
	EncPayload  []byte
}

// DeleteParams параметры удаления секрета.
type DeleteParams struct {
	UserID      string
	SecretID    string
	BaseVersion int64
}

// Row представление секрета уровня Tier 2a (данные строки и версия).
type Row struct {
	ID      string
	Type    domain.SecretType
	Version int64
	EncRow  []byte
}

// IndexItem представление секрета уровня Tier 2b (данные индекса и версия).
type IndexItem struct {
	ID       string
	Version  int64
	EncIndex []byte
}

// Payload представление секрета уровня Tier 3 (полезные данные и версия).
type Payload struct {
	ID         string
	Type       domain.SecretType
	Version    int64
	EncPayload []byte
}
