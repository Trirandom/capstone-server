package apitoolbox

import (
	"golang.org/x/crypto/argon2"
)

func HashPassword(password string) []byte {
	return argon2.Key([]byte(password), []byte("mysalt8YI56780IJLKETRD4gsdrstyy'3-(é'zdhgs"), 3, 32*1024, 4, 32)
}

func ByteArrayCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if b[i] != v {
			return false
		}
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
