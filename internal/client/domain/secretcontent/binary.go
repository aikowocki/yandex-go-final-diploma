package secretcontent

const BinarySchemaV1 = 1

type BinaryRow struct {
	V        int      `json:"v"`
	Title    string   `json:"title"`
	Tags     []string `json:"tags,omitempty"`
	Filename string   `json:"filename,omitempty"`
}

type BinaryIndex struct {
	V            int        `json:"v"`
	Size         int64      `json:"size,omitempty"`
	Mime         string     `json:"mime,omitempty"`
	Note         string     `json:"note,omitempty"`
	CustomFields []KeyValue `json:"custom_fields,omitempty"`
}

type BinaryPayload struct {
	V        int       `json:"v"`
	OTPCodes []OTPCode `json:"otp_codes,omitempty"`
}
