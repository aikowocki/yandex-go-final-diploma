package secret

import "github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"

type CreateParams struct {
	UserID     string
	VaultID    string
	SecretID   string // UUID генерируется клиентом
	Type       domain.SecretType
	EncRow     []byte
	EncIndex   []byte
	EncPayload []byte
}

type UpdateParams struct {
	UserID      string
	SecretID    string
	BaseVersion int64
	EncRow      []byte
	EncIndex    []byte
	EncPayload  []byte
}

type DeleteParams struct {
	UserID      string
	SecretID    string
	BaseVersion int64
}

type Row struct {
	ID      string
	Type    domain.SecretType
	Version int64
	EncRow  []byte
}

type IndexItem struct {
	ID       string
	Version  int64
	EncIndex []byte
}

type Payload struct {
	ID         string
	Type       domain.SecretType
	Version    int64
	EncPayload []byte
}
