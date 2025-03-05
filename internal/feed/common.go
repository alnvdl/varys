package feed

import (
	"crypto/sha256"
	"encoding/hex"
)

// UID returns a unique ID based on the input string.
func UID(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
