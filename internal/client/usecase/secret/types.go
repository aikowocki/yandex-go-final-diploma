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
	OTPCodes     []secretcontent.OTPCode
}

type CreateTextInput struct {
	Title        string
	Tags         []string
	Note         string
	CustomFields []secretcontent.KeyValue
	Body         string
	OTPCodes     []secretcontent.OTPCode
}

type CreateBankCardInput struct {
	Title        string
	Tags         []string
	Bank         string
	Cardholder   string
	Brand        string
	Expiry       string
	Note         string
	CustomFields []secretcontent.KeyValue
	PAN          string
	CVV          string
	PIN          string
	OTPCodes     []secretcontent.OTPCode
}

type CreateTOTPInput struct {
	Title        string
	Tags         []string
	Issuer       string
	Account      string
	Note         string
	CustomFields []secretcontent.KeyValue
	Secret       string
	Algo         string
	Digits       int
	Period       int
	OTPCodes     []secretcontent.OTPCode
}

type TextDetail struct {
	ID      string
	Version int64
	Row     secretcontent.TextRow
	Index   secretcontent.TextIndex
	Payload secretcontent.TextPayload
}

type BankCardDetail struct {
	ID      string
	Version int64
	Row     secretcontent.BankCardRow
	Index   secretcontent.BankCardIndex
	Payload secretcontent.BankCardPayload
}

type TOTPDetail struct {
	ID      string
	Version int64
	Row     secretcontent.TOTPRow
	Index   secretcontent.TOTPIndex
	Payload secretcontent.TOTPPayload
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
