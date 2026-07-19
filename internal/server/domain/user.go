package domain

import "time"

// User represents a registered account on the server.
type User struct {
	ID           string
	Login        string
	AuthHash     string // PHC-строка Argon2id(LoginCredential)
	EncKDFSalt   []byte
	EncKDFParams []byte // JSON crypto.Params для вывода KEK из EncryptionPassphrase
	EncMasterKey []byte // случайный MasterKey, обёрнутый KEK
	TOTPSecret   []byte
	TOTPEnabled  bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
