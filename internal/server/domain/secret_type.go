package domain

type SecretType int16

const (
	SecretTypeLoginPassword SecretType = 1
	SecretTypeText          SecretType = 2
	SecretTypeBinary        SecretType = 3
	SecretTypeBankCard      SecretType = 4
	SecretTypeTOTP          SecretType = 5
)
