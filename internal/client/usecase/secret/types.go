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

// ConflictChoice — выбор пользователя при разрешении конфликта версий.
type ConflictChoice string

const (
	ChoiceMine   ConflictChoice = "mine"
	ChoiceServer ConflictChoice = "server"
)

// ConflictResult — результат обновления, отклонённого из-за конфликта версий: обе версии
// расшифрованы клиентом (plaintext) для показа пользователю. Внутренние поля хранят контекст,
// нужный ResolveConflict, чтобы применить выбор без повторной расшифровки.
type ConflictResult struct {
	SecretID string
	Mine     Detail // Версия, которую пытался записать пользователь
	Server   Detail // Актуальная серверная версия

	vaultID   string
	mineInput CreateLoginPasswordInput
	server    ServerVersion
	isDelete  bool // Конфликт возник при удалении (ChoiceMine → повторить удаление)
}

// ServerVersion — сырые (зашифрованные) поля серверной версии секрета для разрешения конфликта.
type ServerVersion struct {
	Type       int32
	Version    int64
	EncRow     []byte
	EncIndex   []byte
	EncPayload []byte
}
