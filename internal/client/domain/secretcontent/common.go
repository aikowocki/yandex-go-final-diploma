package secretcontent

// KeyValue — произвольная именованная пометка. Общий тип для custom-полей секретов
type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// OTPCode — одноразовый код восстановления. Доступен для ЛЮБОГО типа секрета.
// Used переключается пользователем при использовании кода.
type OTPCode struct {
	Code string `json:"code"`
	Used bool   `json:"used"`
}
