package secretcontent

// BankCardSchemaV1 — версия схемы секрета банковской карты.
const BankCardSchemaV1 = 1

// BankCardRow содержит данные банковской карты уровня Tier 2a (row).
type BankCardRow struct {
	V     int      `json:"v"`
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitempty"`
	Last4 string   `json:"last4,omitempty"`
}

// BankCardIndex содержит данные банковской карты уровня Tier 2b (index).
type BankCardIndex struct {
	V            int        `json:"v"`
	Bank         string     `json:"bank,omitempty"`
	Cardholder   string     `json:"cardholder,omitempty"`
	Brand        string     `json:"brand,omitempty"`
	Expiry       string     `json:"expiry,omitempty"`
	Note         string     `json:"note,omitempty"`
	CustomFields []KeyValue `json:"custom_fields,omitempty"`
}

// BankCardPayload содержит данные банковской карты уровня Tier 3 (payload).
type BankCardPayload struct {
	V        int       `json:"v"`
	PAN      string    `json:"pan"`
	CVV      string    `json:"cvv,omitempty"`
	PIN      string    `json:"pin,omitempty"`
	OTPCodes []OTPCode `json:"otp_codes,omitempty"`
}
