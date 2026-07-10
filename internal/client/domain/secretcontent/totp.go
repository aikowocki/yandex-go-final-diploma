package secretcontent

// TOTPSchemaV1 — текущая версия формата структур totp.
const TOTPSchemaV1 = 1

// TOTPRow — поля строки списка, видимые всегда, без ленивой загрузки.
type TOTPRow struct {
	V      int      `json:"v"`
	Title  string   `json:"title"`
	Tags   []string `json:"tags,omitempty"`
	Issuer string   `json:"issuer,omitempty"`
}

// TOTPIndex — расширенный searchable-индекс, догружается в фоне.
type TOTPIndex struct {
	V            int        `json:"v"`
	Account      string     `json:"account,omitempty"`
	Note         string     `json:"note,omitempty"`
	CustomFields []KeyValue `json:"custom_fields,omitempty"`
}

// TOTPPayload — чувствительное тело (секрет генерации кодов), грузится лениво при просмотре.
type TOTPPayload struct {
	V        int       `json:"v"`
	Secret   string    `json:"secret"` // base32-секрет TOTP
	Algo     string    `json:"algo,omitempty"`
	Digits   int       `json:"digits,omitempty"`
	Period   int       `json:"period,omitempty"`
	OTPCodes []OTPCode `json:"otp_codes,omitempty"`
}
