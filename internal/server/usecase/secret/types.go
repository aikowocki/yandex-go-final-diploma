package secret

import "github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"

type CreateSecretParams struct {
	UserID     string
	VaultID    string
	Type       domain.SecretType
	EncRow     []byte
	EncIndex   []byte
	EncPayload []byte
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
