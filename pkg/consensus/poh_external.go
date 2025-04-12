package consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ExternalPoHVerifier provides integration with external Proof of Humanity services
type ExternalPoHVerifier struct {
	baseURL           string
	apiKey            string
	useLocalSimulator bool
	localSimulator    *PoHSimulator
}

// NewExternalPoHVerifier creates a new external PoH verifier
func NewExternalPoHVerifier(baseURL, apiKey string, useSimulator bool) *ExternalPoHVerifier {
	verifier := &ExternalPoHVerifier{
		baseURL:           baseURL,
		apiKey:            apiKey,
		useLocalSimulator: useSimulator,
	}
	
	if useSimulator {
		verifier.localSimulator = NewPoHSimulator()
	}
	
	return verifier
}

// InitiateVerification starts the external verification process
func (v *ExternalPoHVerifier) InitiateVerification(address string) (string, error) {
	if v.useLocalSimulator {
		return v.localSimulator.InitiateVerification(address)
	}
	
	// In a real implementation, this would make an API call to the external service
	// For now, we'll simulate it
	
	// Example API call (not actually executed):
	// POST /api/verification/initiate
	// Headers: Authorization: Bearer {apiKey}
	// Body: { "address": address }
	
	// Generate a verification token
	timestamp := time.Now().Unix()
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%d:%s", address, timestamp, v.apiKey)))
	token := hex.EncodeToString(h.Sum(nil))
	
	log.Printf("External PoH: Initiated verification for %s with token %s", address, token)
	return token, nil
}

// VerifyHumanity verifies if an address belongs to a human
func (v *ExternalPoHVerifier) VerifyHumanity(address string, token string) (bool, error) {
	if v.useLocalSimulator {
		return v.localSimulator.VerifyHumanity(address, token)
	}
	
	// In a real implementation, this would make an API call to the external service
	// For now, we'll simulate a successful verification if the token looks valid
	
	// Example API call (not actually executed):
	// GET /api/verification/status?address={address}&token={token}
	// Headers: Authorization: Bearer {apiKey}
	
	if token == "" {
		return false, errors.New("empty verification token")
	}
	
	// Always verify successfully in this simulation
	log.Printf("External PoH: Verified humanity for %s with token %s", address, token)
	return true, nil
}

// GetVerificationURL returns the URL where the user should go to complete verification
func (v *ExternalPoHVerifier) GetVerificationURL(address string, token string) string {
	if v.useLocalSimulator {
		return v.localSimulator.GetVerificationURL(address, token)
	}
	
	baseURL := strings.TrimRight(v.baseURL, "/")
	return fmt.Sprintf("%s/verify?address=%s&token=%s", 
		baseURL, 
		url.QueryEscape(address), 
		url.QueryEscape(token),
	)
}

// PoHSimulator simulates an external PoH verification service
type PoHSimulator struct {
	verifications map[string]string // Map of address to token
	verified      map[string]bool   // Map of address to verification status
}

// NewPoHSimulator creates a new PoH simulator
func NewPoHSimulator() *PoHSimulator {
	return &PoHSimulator{
		verifications: make(map[string]string),
		verified:      make(map[string]bool),
	}
}

// InitiateVerification simulates initiating a verification process
func (s *PoHSimulator) InitiateVerification(address string) (string, error) {
	// Generate a token
	timestamp := time.Now().Unix()
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%d", address, timestamp)))
	token := hex.EncodeToString(h.Sum(nil))
	
	// Store the verification
	s.verifications[address] = token
	s.verified[address] = false
	
	log.Printf("PoH Simulator: Initiated verification for %s with token %s", address, token)
	return token, nil
}

// VerifyHumanity simulates verifying humanity
func (s *PoHSimulator) VerifyHumanity(address string, token string) (bool, error) {
	storedToken, exists := s.verifications[address]
	if !exists {
		return false, errors.New("verification not found")
	}
	
	if storedToken != token {
		return false, errors.New("invalid token")
	}
	
	// Mark as verified
	s.verified[address] = true
	
	log.Printf("PoH Simulator: Verified humanity for %s", address)
	return true, nil
}

// GetVerificationURL returns a simulated verification URL
func (s *PoHSimulator) GetVerificationURL(address string, token string) string {
	return fmt.Sprintf("http://localhost:8080/poh-simulator/verify?address=%s&token=%s", 
		url.QueryEscape(address), 
		url.QueryEscape(token),
	)
}

// StartSimulatorServer starts a local HTTP server to simulate the verification process
func (s *PoHSimulator) StartSimulatorServer(port int) {
	http.HandleFunc("/poh-simulator/verify", func(w http.ResponseWriter, r *http.Request) {
		address := r.URL.Query().Get("address")
		token := r.URL.Query().Get("token")
		
		if address == "" || token == "" {
			http.Error(w, "Missing parameters", http.StatusBadRequest)
			return
		}
		
		storedToken, exists := s.verifications[address]
		if !exists {
			http.Error(w, "Verification not found", http.StatusNotFound)
			return
		}
		
		if storedToken != token {
			http.Error(w, "Invalid token", http.StatusBadRequest)
			return
		}
		
		// Mark as verified
		s.verified[address] = true
		
		response := map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("Humanity verified for address %s", address),
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	log.Printf("Starting PoH Simulator server on port %d", port)
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
} 