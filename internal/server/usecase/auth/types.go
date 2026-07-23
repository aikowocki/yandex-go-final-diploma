package auth

// RegisterParams параметры регистрации.
type RegisterParams struct {
	Login           string
	LoginCredential []byte
}

// RegisterResult результат регистрации.
type RegisterResult struct {
	UserID       string
	AccessToken  string
	RefreshToken string
}

// SetupEncryptionParams параметры настройки шифрования.
type SetupEncryptionParams struct {
	UserID       string
	EncKDFSalt   []byte
	EncKDFParams []byte
	EncMasterKey []byte
}

// LoginParams параметры входа.
type LoginParams struct {
	Login           string
	LoginCredential []byte
}

// Result результат входа, регистрации и обновления токенов.
type Result struct {
	UserID       string
	AccessToken  string
	RefreshToken string
	EncKDFSalt   []byte
	EncKDFParams []byte
	EncMasterKey []byte
}

// RefreshParams параметры обновления токена.
type RefreshParams struct {
	RefreshToken string
}
