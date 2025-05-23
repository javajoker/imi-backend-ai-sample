// internal/utils/crypto.go
package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

func GenerateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)

	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}

	return string(b), nil
}

func GenerateVerificationCode() (string, error) {
	return GenerateRandomString(32)
}

func HashString(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	return hex.EncodeToString(hasher.Sum(nil))
}

func GenerateAPIKey() (string, error) {
	prefix := "ip_"
	randomPart, err := GenerateRandomString(32)
	if err != nil {
		return "", err
	}
	return prefix + randomPart, nil
}

func ValidateFileHash(fileData []byte, expectedHash string) bool {
	hasher := sha256.New()
	hasher.Write(fileData)
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	return actualHash == expectedHash
}
