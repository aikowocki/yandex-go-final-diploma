package secretcontent

// TextSchemaV1 — текущая версия формата структур text.
const TextSchemaV1 = 1

// TextRow — поля строки списка, видимые всегда, без ленивой загрузки.
type TextRow struct {
	V     int      `json:"v"`
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitempty"`
}

// TextIndex — расширенный searchable-индекс, догружается в фоне.
type TextIndex struct {
	V            int        `json:"v"`
	Note         string     `json:"note,omitempty"`
	CustomFields []KeyValue `json:"custom_fields,omitempty"`
}

// TextPayload — чувствительное тело (сам текст заметки), грузится лениво при просмотре.
type TextPayload struct {
	V        int       `json:"v"`
	Body     string    `json:"body"`
	OTPCodes []OTPCode `json:"otp_codes,omitempty"`
}
