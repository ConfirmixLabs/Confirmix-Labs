package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"confirmix/pkg/api"
	"confirmix/pkg/blockchain"
	"confirmix/pkg/consensus"
)

func main() {
	// Initialize blockchain
	bc, err := blockchain.NewBlockchain()
	if err != nil {
		log.Fatalf("Failed to initialize blockchain: %v", err)
	}

	// Initialize validator manager with genesis admin
	initialAdmins := []string{bc.Admins[0]} // Use the first admin from blockchain
	vm := consensus.NewValidatorManager(bc, initialAdmins, consensus.ModeAdminOnly)

	// Create a new hybrid consensus engine with 5-second block time
	blockTime := 5 * time.Second
	ce := consensus.NewHybridConsensus(bc, nil, "", blockTime) // We'll use nil for private key and empty string for address for now

	// Create a governance system (can be nil if not used)
	var gov *consensus.Governance = nil

	// Initialize web server
	server := api.NewWebServer(bc, ce, vm, gov, 8080)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down...")
	if err := server.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
} 