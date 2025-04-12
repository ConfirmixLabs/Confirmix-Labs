package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/confirmix/pkg/blockchain"
	"github.com/confirmix/pkg/validator"
)

type ValidatorConfig struct {
	Address    string
	PrivateKey string
	MultisigAddr string
}

func main() {
	// Parse command line flags
	config := ValidatorConfig{}
	flag.StringVar(&config.Address, "address", "", "Validator address")
	flag.StringVar(&config.PrivateKey, "privateKey", "", "Validator private key")
	flag.StringVar(&config.MultisigAddr, "multisig", "774eb077-14a3-4bdd-a2a5-41ace2b0e155", "Multisig wallet address")
	flag.Parse()

	// Validate input
	if config.Address == "" || config.PrivateKey == "" {
		log.Fatal("Both address and private key are required")
	}

	// Import private key
	keyPair, err := blockchain.ImportPrivateKey(config.PrivateKey)
	if err != nil {
		log.Fatalf("Failed to import private key: %v", err)
	}

	// Verify address matches key pair
	if err := blockchain.VerifyAddress(config.Address, keyPair.PublicKey); err != nil {
		log.Fatalf("Address verification failed: %v", err)
	}

	// Start validator
	v := validator.NewValidator(config.Address, keyPair, config.MultisigAddr)
	if err := v.Start(); err != nil {
		log.Fatalf("Failed to start validator: %v", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down validator...")
	v.Stop()
} 