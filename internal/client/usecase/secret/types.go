package secret

import "github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"

type CreateLoginPasswordInput struct {
	Title        string
	Tags         []string
	URI          string
	Username     string
	Note         string
	CustomFields []secretcontent.KeyValue
	Password     string
}

type DecryptedRow struct {
	ID      string
	Version int64
	Row     secretcontent.LoginPasswordRow
}

type DecryptedPayload struct {
	ID      string
	Version int64
	Payload secretcontent.LoginPasswordPayload
}
