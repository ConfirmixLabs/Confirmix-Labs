package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// CurrentTimestamp returns the current Unix timestamp
func CurrentTimestamp() int64 {
	return time.Now().Unix()
}

// GenerateHash generates a SHA-256 hash of the given data
func GenerateHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// GenerateAddressFromPublicKey generates an address from a public key
func GenerateAddressFromPublicKey(publicKey []byte) string {
	hash := sha256.Sum256(publicKey)
	return hex.EncodeToString(hash[:20]) // Use first 20 bytes of hash
}

// ConcatBytes concatenates multiple byte slices
func ConcatBytes(slices ...[]byte) []byte {
	var totalLen int
	for _, s := range slices {
		totalLen += len(s)
	}
	
	result := make([]byte, totalLen)
	var i int
	for _, s := range slices {
		i += copy(result[i:], s)
	}
	
	return result
} 