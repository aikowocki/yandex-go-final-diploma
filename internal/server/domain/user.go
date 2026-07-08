package domain

import "time"

type User struct {
	ID           string
	Login        string
	AuthHash     string // PHC-строка Argon2id(LoginCredential)
	EncKDFSalt   []byte
	EncKDFParams []byte // JSON crypto.Params для вывода MasterSeed из EncryptionPassphrase
	TOTPSecret   []byte
	TOTPEnabled  bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
