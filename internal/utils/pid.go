package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// PathSum256 returns a SHA256 hex encoded string based on the provided path.
func PathSum256(path string) string {
	f256 := sha256.Sum256([]byte(path))
	return hex.EncodeToString(f256[0:])
}
