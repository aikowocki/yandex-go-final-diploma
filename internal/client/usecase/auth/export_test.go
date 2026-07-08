package auth

// MasterKeyForTest даёт тестам доступ к выведенному MasterKey.
func (u *UseCase) MasterKeyForTest() []byte {
	return u.session.masterKey
}
