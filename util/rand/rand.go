package rand

import (
	"crypto/rand"
	"encoding/hex"
)

// RandString returns a cryptographically-secure pseudo-random alpha-numeric string of a given length
func RandString(n int) (string, error) {
	bytes := make([]byte, n/2+1) // we need one extra letter to discard
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[0:n], nil
}
