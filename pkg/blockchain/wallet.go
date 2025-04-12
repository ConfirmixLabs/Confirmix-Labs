package blockchain

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/elliptic"
	"encoding/hex"
	"strings"
)

// Wallet represents a user's wallet with a key pair
type Wallet struct {
	Address    string
	KeyPair    *KeyPair
}

// CreateWallet creates a new wallet and returns the wallet address
func CreateWallet() (*Wallet, error) {
	keyPair, err := NewKeyPair()
	if err != nil {
		return nil, err
	}

	address := GenerateAddress(keyPair.PublicKey)
	wallet := &Wallet{
		Address: address,
		KeyPair: keyPair,
	}

	return wallet, nil
}

// GenerateAddress generates a blockchain address from a public key
func GenerateAddress(pubKey *ecdsa.PublicKey) string {
	// Get the public key bytes
	pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	
	// Take the last 20 bytes of the SHA3-256 hash of the public key
	hash := sha256.Sum256(pubKeyBytes)
	addressBytes := hash[len(hash)-20:]
	
	// Convert to hex and add 0x prefix
	address := "0x" + hex.EncodeToString(addressBytes)
	
	// Ensure the address is 42 characters long (0x + 40 hex chars)
	if len(address) != 42 {
		// Pad with zeros if needed
		address = "0x" + strings.Repeat("0", 40-len(address[2:])) + address[2:]
	}
	
	return address
} 