package consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// HumanVerification represents a human verification record
type HumanVerification struct {
	Address    string
	ProofToken string
	Timestamp  int64
	ExpiresAt  int64
	Verified   bool
}

// ProofOfHumanity implements a Proof of Humanity verification system
type ProofOfHumanity struct {
	verifications     map[string]*HumanVerification
	verificationMutex sync.RWMutex
	expirationTime    time.Duration // How long a verification remains valid
}

// NewProofOfHumanity creates a new Proof of Humanity system
func NewProofOfHumanity(expirationTime time.Duration) *ProofOfHumanity {
	return &ProofOfHumanity{
		verifications:  make(map[string]*HumanVerification),
		expirationTime: expirationTime,
	}
}

// RegisterVerification initiates a human verification process
func (poh *ProofOfHumanity) RegisterVerification(address string) (string, error) {
	poh.verificationMutex.Lock()
	defer poh.verificationMutex.Unlock()
	
	// Check if verification already exists
	if verification, exists := poh.verifications[address]; exists {
		if verification.Verified && time.Now().Unix() < verification.ExpiresAt {
			return verification.ProofToken, nil
		}
	}
	
	// Generate a proof token (in a real system, this would be part of an external verification process)
	timestamp := time.Now().Unix()
	tokenInput := fmt.Sprintf("%s:%d", address, timestamp)
	hash := sha256.Sum256([]byte(tokenInput))
	proofToken := hex.EncodeToString(hash[:])
	
	// Create new verification record
	verification := &HumanVerification{
		Address:    address,
		ProofToken: proofToken,
		Timestamp:  timestamp,
		ExpiresAt:  timestamp + int64(poh.expirationTime.Seconds()),
		Verified:   false, // Will be set to true after verification process
	}
	
	poh.verifications[address] = verification
	
	// In a real system, we would initiate an external verification process here
	// For demonstration purposes, we'll return the token directly
	return proofToken, nil
}

// CompleteVerification completes the verification process
// In a real system, this would be called by an external verification service
func (poh *ProofOfHumanity) CompleteVerification(address string, proofToken string) error {
	poh.verificationMutex.Lock()
	defer poh.verificationMutex.Unlock()
	
	verification, exists := poh.verifications[address]
	if !exists {
		return errors.New("verification not found")
	}
	
	if verification.ProofToken != proofToken {
		return errors.New("invalid proof token")
	}
	
	// Check if verification has expired
	if time.Now().Unix() > verification.ExpiresAt {
		delete(poh.verifications, address)
		return errors.New("verification expired")
	}
	
	verification.Verified = true
	return nil
}

// IsHumanVerified checks if an address has been verified as human
func (poh *ProofOfHumanity) IsHumanVerified(address string) bool {
	poh.verificationMutex.RLock()
	defer poh.verificationMutex.RUnlock()
	
	verification, exists := poh.verifications[address]
	if !exists {
		return false
	}
	
	// Check if verification is still valid
	if !verification.Verified || time.Now().Unix() > verification.ExpiresAt {
		return false
	}
	
	return true
}

// GetProofToken gets the proof token for a verified address
func (poh *ProofOfHumanity) GetProofToken(address string) (string, error) {
	poh.verificationMutex.RLock()
	defer poh.verificationMutex.RUnlock()
	
	verification, exists := poh.verifications[address]
	if !exists {
		return "", errors.New("verification not found")
	}
	
	if !verification.Verified {
		return "", errors.New("address not verified")
	}
	
	if time.Now().Unix() > verification.ExpiresAt {
		return "", errors.New("verification expired")
	}
	
	return verification.ProofToken, nil
}

// CleanupExpiredVerifications removes expired verifications
func (poh *ProofOfHumanity) CleanupExpiredVerifications() {
	poh.verificationMutex.Lock()
	defer poh.verificationMutex.Unlock()
	
	now := time.Now().Unix()
	for address, verification := range poh.verifications {
		if now > verification.ExpiresAt {
			delete(poh.verifications, address)
		}
	}
}

// StartCleanupRoutine starts a routine to periodically clean up expired verifications
func (poh *ProofOfHumanity) StartCleanupRoutine(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			poh.CleanupExpiredVerifications()
		}
	}()
} 