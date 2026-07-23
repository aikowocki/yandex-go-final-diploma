package secretcontent

// BinarySchemaV1 — версия схемы секрета.
const BinarySchemaV1 = 1

// BinaryRow содержит данные секрета уровня Tier 2a (row).
type BinaryRow struct {
	V        int      `json:"v"`
	Title    string   `json:"title"`
	Tags     []string `json:"tags,omitempty"`
	Filename string   `json:"filename,omitempty"`
}

// BinaryIndex содержит данные секрета уровня Tier 2b (index).
type BinaryIndex struct {
	V            int        `json:"v"`
	Size         int64      `json:"size,omitempty"`
	Mime         string     `json:"mime,omitempty"`
	Note         string     `json:"note,omitempty"`
	CustomFields []KeyValue `json:"custom_fields,omitempty"`
}

// BinaryPayload содержит данные секрета уровня Tier 3 (payload).
type BinaryPayload struct {
	V        int       `json:"v"`
	OTPCodes []OTPCode `json:"otp_codes,omitempty"`
}
