package main

import "crypto/sha256"

// hashPassword описывает формат пароля
type hashPassword [sha256.Size224]byte

// newHashPassword возвращает хеш от пароля
func newHashPassword(password string) hashPassword {
	return sha256.Sum224([]byte(password))
}

// Equal сравнивает указанный пароль с сохраненным и возвращает true, если они
// полностью совпадают.
func (h hashPassword) Equal(password string) bool {
	passwd := newHashPassword(password)
	for i := range h {
		if h[i] != passwd[i] {
			return false
		}
	}
	return true
}
