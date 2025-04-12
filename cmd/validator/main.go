package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"confirmix/pkg/blockchain"
	"confirmix/pkg/validator"
	"crypto/elliptic"
)

type ValidatorInfo struct {
	Address    string `json:"address"`
	HumanProof string `json:"humanProof"`
}

type ValidatorConfig struct {
	Address      string
	PrivateKey   string
	MultisigAddr string
	APIURL       string
	Interval     int
	Power        int64
}

func main() {
	// Command line flags
	config := ValidatorConfig{}
	flag.StringVar(&config.Address, "address", "", "Signer wallet address")
	flag.StringVar(&config.PrivateKey, "privateKey", "", "Signer wallet private key")
	flag.StringVar(&config.MultisigAddr, "multisig", "0x0000000000000000000000000000000000000000admin", "Multisig wallet address")
	flag.StringVar(&config.APIURL, "api", "http://localhost:8080/api", "API base URL")
	flag.IntVar(&config.Interval, "interval", 10, "Validation interval in seconds")
	flag.Int64Var(&config.Power, "power", 100, "Validator voting power")
	flag.Parse()

	// Validate inputs
	if config.Address == "" || config.PrivateKey == "" {
		fmt.Println("Error: Address and private key are required")
		fmt.Println("Usage: validator -address=<signer_address> -privateKey=<private_key> [-multisig=<multisig_address>] [-api=<api_url>] [-interval=<seconds>] [-power=<voting_power>]")
		os.Exit(1)
	}

	// Import private key
	privKey, err := blockchain.ImportPrivateKey(config.PrivateKey)
	if err != nil {
		log.Fatalf("Failed to import private key: %v", err)
	}

	// Create key pair
	keyPair := &blockchain.KeyPair{
		PrivateKey:     privKey,
		PublicKey:      &privKey.PublicKey,
		PublicKeyBytes: elliptic.Marshal(elliptic.P256(), privKey.PublicKey.X, privKey.PublicKey.Y),
	}

	// Verify the address matches
	if keyPair.GetAddress() != config.Address {
		log.Fatalf("Address mismatch: expected %s, got %s", config.Address, keyPair.GetAddress())
	}

	// Create validator manager
	manager := validator.NewManager()

	// Create validator
	validator := validator.NewValidator(
		config.Address,
		config.Power,
		10*time.Second,
		config.APIURL,
		manager,
	)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start validation loop
	if err := validator.Start(); err != nil {
		log.Fatalf("Failed to start validator: %v", err)
	}

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down validator...")
	validator.Stop()
}

// checkValidatorStatus checks if the address is a registered validator
func checkValidatorStatus(apiURL, address string) (bool, error) {
	// Fix API URL if needed
	apiURL = strings.TrimSuffix(apiURL, "/")
	
	// Get validators list
	validatorsURL := fmt.Sprintf("%s/validators", apiURL)
	log.Printf("Requesting validators from: %s", validatorsURL)
	
	resp, err := http.Get(validatorsURL)
	if err != nil {
		return false, fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}
	
	var validators []ValidatorInfo
	if err := json.NewDecoder(resp.Body).Decode(&validators); err != nil {
		return false, fmt.Errorf("failed to decode validators: %w", err)
	}
	
	// Check if address is in validators list
	for _, v := range validators {
		if v.Address == address {
			return true, nil
		}
	}
	
	return false, nil
} 