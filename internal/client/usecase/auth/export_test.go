package auth

// MasterKeyForTest даёт тестам доступ к выведенному MasterKey (из общей сессии).
func (u *UseCase) MasterKeyForTest() []byte {
	mk, _ := u.sess.MasterKey()
	return mk
}
