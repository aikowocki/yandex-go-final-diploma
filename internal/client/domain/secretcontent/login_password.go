// Package secretcontent содержит plaintext-структуры содержимого секретов.
// Отделены от сущностей домена (Vault, Secret): здесь — расшифрованный контент, который клиент
// сериализует в JSON и шифрует под VaultKey (enc_row/enc_index/enc_payload = Encrypt(VaultKey,
// json.Marshal(fields))).

package secretcontent

// LoginPasswordSchemaV1 — текущая версия формата структур login_password.
const LoginPasswordSchemaV1 = 1

// LoginPasswordRow —  поля строки списка, видимые всегда, без ленивой загрузки.
type LoginPasswordRow struct {
	V        int      `json:"v"`
	Title    string   `json:"title"`
	Tags     []string `json:"tags,omitempty"`
	URI      string   `json:"uri,omitempty"`
	Username string   `json:"username,omitempty"`
}

// LoginPasswordIndex — расширенный searchable-индекс, догружается в фоне.
type LoginPasswordIndex struct {
	V            int        `json:"v"`
	Note         string     `json:"note,omitempty"`
	CustomFields []KeyValue `json:"custom_fields,omitempty"`
}

// LoginPasswordPayload — чувствительное тело, грузится лениво при просмотре.
type LoginPasswordPayload struct {
	V        int    `json:"v"`
	Password string `json:"password"`
}
