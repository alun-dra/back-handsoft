package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func NewRefreshToken() (plain string, hash string, err error) {
	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b)
	hash = HashRefreshToken(plain)
	return plain, hash, nil
}

func HashRefreshToken(plain string) string {
	h := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(h[:])
}
