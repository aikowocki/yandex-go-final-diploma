package secretcontent

const BankCardSchemaV1 = 1

type BankCardRow struct {
	V     int      `json:"v"`
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitempty"`
	Last4 string   `json:"last4,omitempty"`
}

type BankCardIndex struct {
	V            int        `json:"v"`
	Bank         string     `json:"bank,omitempty"`
	Cardholder   string     `json:"cardholder,omitempty"`
	Brand        string     `json:"brand,omitempty"`
	Expiry       string     `json:"expiry,omitempty"`
	Note         string     `json:"note,omitempty"`
	CustomFields []KeyValue `json:"custom_fields,omitempty"`
}

type BankCardPayload struct {
	V        int       `json:"v"`
	PAN      string    `json:"pan"`
	CVV      string    `json:"cvv,omitempty"`
	PIN      string    `json:"pin,omitempty"`
	OTPCodes []OTPCode `json:"otp_codes,omitempty"`
}
