package util

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// PublicKeyToAddress ECDSA public key'den adres oluşturur
func PublicKeyToAddress(publicKey *ecdsa.PublicKey) string {
	// Public key'in x ve y koordinatlarını birleştir
	pubBytes := append(publicKey.X.Bytes(), publicKey.Y.Bytes()...)
	
	// SHA256 hash hesapla
	hash := sha256.Sum256(pubBytes)
	
	// Hash'i hexadecimal string'e çevir
	address := hex.EncodeToString(hash[:20]) // İlk 20 byte'ı kullan
	
	return address
}

// CurrentTimestamp Unix timestamp döndürür
func CurrentTimestamp() int64 {
	return time.Now().Unix()
} 