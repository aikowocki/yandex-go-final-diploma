package secret

import "github.com/aikowocki/yandex-go-final-diploma/internal/client/domain/secretcontent"

// CreateLoginPasswordInput входные данные для создания секрета логин/пароль.
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

// CreateTextInput входные данные для создания текстового секрета.
type CreateTextInput struct {
	Title        string
	Tags         []string
	Note         string
	CustomFields []secretcontent.KeyValue
	Body         string
	OTPCodes     []secretcontent.OTPCode
}

// CreateBankCardInput входные данные для создания секрета «банковская карта».
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

// CreateTOTPInput входные данные для создания секрета TOTP.
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

// TextDetail — детальная карточка текстового секрета.
type TextDetail struct {
	ID      string
	Version int64
	Row     secretcontent.TextRow
	Index   secretcontent.TextIndex
	Payload secretcontent.TextPayload
}

// BankCardDetail — детальная карточка секрета «банковская карта».
type BankCardDetail struct {
	ID      string
	Version int64
	Row     secretcontent.BankCardRow
	Index   secretcontent.BankCardIndex
	Payload secretcontent.BankCardPayload
}

// TOTPDetail — детальная карточка секрета TOTP.
type TOTPDetail struct {
	ID      string
	Version int64
	Row     secretcontent.TOTPRow
	Index   secretcontent.TOTPIndex
	Payload secretcontent.TOTPPayload
}

// DecryptedRow — расшифрованная строка (row-тир) секрета логин/пароль.
type DecryptedRow struct {
	ID      string
	Version int64
	Row     secretcontent.LoginPasswordRow
}

// DecryptedPayload — расшифрованный payload-тир секрета логин/пароль.
type DecryptedPayload struct {
	ID      string
	Version int64
	Payload secretcontent.LoginPasswordPayload
}

// ConflictChoice — выбор пользователя при разрешении конфликта версий.
type ConflictChoice string

// Варианты разрешения конфликта версий секрета.
const (
	// ChoiceMine — оставить локальную версию, повторно записав её с новым baseVersion.
	ChoiceMine ConflictChoice = "mine"
	// ChoiceServer — принять серверную версию, отбросив локальные изменения.
	ChoiceServer ConflictChoice = "server"
)
