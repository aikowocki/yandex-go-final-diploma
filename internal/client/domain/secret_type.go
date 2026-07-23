package domain

// SecretType определяет тип содержимого секрета.
type SecretType int16

// Типы секретов.
const (
	SecretTypeLoginPassword SecretType = 1
	SecretTypeText          SecretType = 2
	SecretTypeBinary        SecretType = 3
	SecretTypeBankCard      SecretType = 4
	SecretTypeTOTP          SecretType = 5
)
