package secretcontent

// KeyValue — произвольная именованная пометка. Общий тип для custom-полей секретов
type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
