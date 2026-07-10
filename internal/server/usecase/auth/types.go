package auth

type RegisterParams struct {
	Login           string
	LoginCredential []byte
}

type RegisterResult struct {
	UserID       string
	AccessToken  string
	RefreshToken string
}

type SetupEncryptionParams struct {
	UserID       string
	EncKDFSalt   []byte
	EncKDFParams []byte
}

type LoginParams struct {
	Login           string
	LoginCredential []byte
}

type AuthResult struct {
	UserID       string
	AccessToken  string
	RefreshToken string
	EncKDFSalt   []byte
	EncKDFParams []byte
}

type RefreshParams struct {
	RefreshToken string
}
