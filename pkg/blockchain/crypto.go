package blockchain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"
)

// KeyPair represents a public-private key pair
type KeyPair struct {
	PrivateKey     *ecdsa.PrivateKey
	PublicKey      *ecdsa.PublicKey
	PublicKeyBytes []byte
}

// NewKeyPair generates a new ECDSA key pair
func NewKeyPair() (*KeyPair, error) {
	curve := elliptic.P256()
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}

	// Marshal public key to bytes
	publicKeyBytes := elliptic.Marshal(curve, privateKey.PublicKey.X, privateKey.PublicKey.Y)

	return &KeyPair{
		PrivateKey:     privateKey,
		PublicKey:      &privateKey.PublicKey,
		PublicKeyBytes: publicKeyBytes,
	}, nil
}

// SignTransaction signs a transaction with the given private key
func (tx *Transaction) Sign(privateKey *ecdsa.PrivateKey) error {
	// Create a hash of the transaction data
	hash := tx.CalculateHash()
	
	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, []byte(hash))
	if err != nil {
		return err
	}

	// Combine r and s into a single signature
	tx.Signature = append(r.Bytes(), s.Bytes()...)
	return nil
}

// VerifyTransaction verifies the transaction signature
func (tx *Transaction) Verify(publicKey *ecdsa.PublicKey) error {
	if tx.Signature == nil || len(tx.Signature) == 0 {
		return errors.New("transaction is not signed")
	}

	// Split signature into r and s components
	r := new(big.Int).SetBytes(tx.Signature[:len(tx.Signature)/2])
	s := new(big.Int).SetBytes(tx.Signature[len(tx.Signature)/2:])

	// Create a hash of the transaction data
	hash := tx.CalculateHash()

	// Verify the signature
	valid := ecdsa.Verify(publicKey, []byte(hash), r, s)
	if !valid {
		return errors.New("invalid transaction signature")
	}

	return nil
}

// CalculateHash calculates the hash of a transaction for signing
func (tx *Transaction) CalculateHash() string {
	// Create a string representation of the transaction
	data := tx.ID + tx.From + tx.To + string(IntToHex(int64(tx.Value * 1000000)))
	if tx.Data != nil {
		data += string(tx.Data)
	}
	data += string(IntToHex(tx.Timestamp))

	// Calculate SHA-256 hash
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GetPublicKeyFromAddress converts an address to a public key
func GetPublicKeyFromAddress(address string) (*ecdsa.PublicKey, error) {
	// In a real implementation, this would decode the address and reconstruct the public key
	// For now, we'll return an error as this is a placeholder
	return nil, errors.New("address to public key conversion not implemented")
}

// GetAddress returns the address derived from the public key (Ethereum format)
func (kp *KeyPair) GetAddress() string {
	if kp.PublicKeyBytes == nil {
		return ""
	}
	
	// Create a hash of the public key
	hash := sha256.Sum256(kp.PublicKeyBytes)
	
	// Take last 20 bytes of the hash (like Ethereum)
	addressBytes := hash[len(hash)-20:]
	
	// Add 0x prefix
	return "0x" + hex.EncodeToString(addressBytes)
}

// GetPrivateKeyString returns the private key as a hexadecimal string with 0x prefix
func (kp *KeyPair) GetPrivateKeyString() string {
	if kp.PrivateKey == nil {
		return ""
	}
	return "0x" + hex.EncodeToString(kp.PrivateKey.D.Bytes())
}

// GetPublicKeyString returns the public key as a hexadecimal string with 0x prefix
func (kp *KeyPair) GetPublicKeyString() string {
	if kp.PublicKeyBytes == nil {
		return ""
	}
	return "0x" + hex.EncodeToString(kp.PublicKeyBytes)
}

// SaveToFile saves the key pair to a file in the data directory
func (kp *KeyPair) SaveToFile(address string) error {
	// Create data directory if it doesn't exist
	dataDir := "data"
	os.MkdirAll(dataDir, 0755)
	
	// Create key pair data
	keyData := struct {
		Address     string `json:"address"`
		PrivateKey  string `json:"private_key"`
		PublicKey   string `json:"public_key"`
	}{
		Address:     address,
		PrivateKey:  kp.GetPrivateKeyString(),
		PublicKey:   kp.GetPublicKeyString(),
	}
	
	// Marshal to JSON
	data, err := json.MarshalIndent(keyData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal key pair: %v", err)
	}
	
	// Save to file
	filename := filepath.Join(dataDir, fmt.Sprintf("key_%s.json", address))
	err = ioutil.WriteFile(filename, data, 0600) // 0600 for private key files
	if err != nil {
		return fmt.Errorf("failed to save key pair: %v", err)
	}
	
	return nil
}

// VerifySignature verifies a signature using raw byte arrays
func VerifySignature(dataHash []byte, signature []byte, publicKey []byte) (bool, error) {
	if len(signature) == 0 {
		return false, errors.New("empty signature")
	}
	
	if len(publicKey) == 0 {
		return false, errors.New("empty public key")
	}
	
	// Convert public key bytes to ecdsa.PublicKey
	curve := elliptic.P256()
	x, y := elliptic.Unmarshal(curve, publicKey)
	if x == nil {
		return false, errors.New("failed to unmarshal public key")
	}
	
	pubKey := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}
	
	// Split signature into r and s components
	r := new(big.Int).SetBytes(signature[:len(signature)/2])
	s := new(big.Int).SetBytes(signature[len(signature)/2:])
	
	// Verify signature
	return ecdsa.Verify(pubKey, dataHash, r, s), nil
}

// VerifySignature verifies a signature using the key pair's public key
func (kp *KeyPair) VerifySignature(dataHash []byte, signature []byte) (bool, error) {
	return VerifySignature(dataHash, signature, kp.PublicKeyBytes)
}

// ImportPrivateKey reconstructs a private key from a hex string
func ImportPrivateKey(privateKeyHex string) (*ecdsa.PrivateKey, error) {
	// Remove "0x" prefix if present
	if strings.HasPrefix(privateKeyHex, "0x") {
		privateKeyHex = privateKeyHex[2:]
	}

	// Ensure the private key is 64 characters long (32 bytes)
	if len(privateKeyHex) != 64 {
		return nil, fmt.Errorf("invalid private key length: expected 64 characters, got %d", len(privateKeyHex))
	}

	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key format: %v", err)
	}

	curve := elliptic.P256()
	privateKey := new(ecdsa.PrivateKey)
	privateKey.PublicKey.Curve = curve
	privateKey.D = new(big.Int).SetBytes(privateKeyBytes)
	privateKey.PublicKey.X, privateKey.PublicKey.Y = curve.ScalarBaseMult(privateKeyBytes)
	
	return privateKey, nil
} 