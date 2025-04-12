package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"confirmix/pkg/blockchain"
	"confirmix/pkg/consensus"
)

func main() {
	// Create a new blockchain
	bc := blockchain.NewBlockchain()
	fmt.Println("Created new blockchain")

	// Create private keys for users
	validatorKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	userKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Create addresses from public keys
	validatorPubKey := validatorKey.PublicKey
	validatorPubKeyBytes := elliptic.Marshal(validatorPubKey.Curve, validatorPubKey.X, validatorPubKey.Y)
	validatorAddress := fmt.Sprintf("%x", validatorPubKeyBytes[:10])

	userPubKey := userKey.PublicKey
	userPubKeyBytes := elliptic.Marshal(userPubKey.Curve, userPubKey.X, userPubKey.Y)
	userAddress := fmt.Sprintf("%x", userPubKeyBytes[:10])

	fmt.Printf("Validator address: %s\n", validatorAddress)
	fmt.Printf("User address: %s\n", userAddress)

	// Create custom configuration for hybrid consensus with external PoH
	config := consensus.DefaultHybridConsensusConfig()
	config.UseExternalPoh = true
	config.UsePoHSimulator = true
	config.PoHSimulatorPort = 8081
	config.BlockTime = 5 * time.Second

	// Create consensus engine for validator
	validatorConsensus := consensus.NewHybridConsensusWithConfig(
		bc,
		validatorKey,
		validatorAddress,
		config,
	)

	// Start the external PoH verification process
	proofToken, err := validatorConsensus.InitiateHumanVerification()
	if err != nil {
		log.Fatalf("Failed to initiate human verification: %v", err)
	}

	// Get the verification URL
	verificationURL, err := validatorConsensus.GetVerificationURL()
	if err != nil {
		log.Fatalf("Failed to get verification URL: %v", err)
	}

	fmt.Println("\n===== EXTERNAL POH VERIFICATION =====")
	fmt.Printf("Proof Token: %s\n", proofToken)
	fmt.Printf("Verification URL: %s\n", verificationURL)
	fmt.Println("In a real implementation, you would visit this URL to complete verification.")
	fmt.Println("For this example, we'll simulate automatic verification.")
	fmt.Println("=================================\n")

	// Simulate completing the verification
	err = validatorConsensus.CompleteHumanVerification(proofToken)
	if err != nil {
		log.Fatalf("Failed to complete human verification: %v", err)
	}

	fmt.Println("Human verification completed via external PoH service")

	// Check if human verified
	isVerified := validatorConsensus.IsHumanVerified()
	fmt.Printf("Is human verified: %v\n", isVerified)

	// Register as validator
	err = validatorConsensus.RegisterAsValidator()
	if err != nil {
		log.Fatalf("Failed to register as validator: %v", err)
	}

	fmt.Println("Registered as validator")

	// Update validator list
	validatorConsensus.UpdateValidatorList([]string{validatorAddress})

	// Create a transaction
	tx := blockchain.NewTransaction(
		userAddress,
		validatorAddress,
		10.0,
		[]byte("Example transaction with external PoH"),
	)

	// Add transaction to blockchain
	err = bc.AddTransaction(tx)
	if err != nil {
		log.Fatalf("Failed to add transaction: %v", err)
	}

	fmt.Println("Transaction added to blockchain")

	// Start mining
	err = validatorConsensus.StartMining()
	if err != nil {
		log.Fatalf("Failed to start mining: %v", err)
	}

	fmt.Println("Mining started...")
	fmt.Println("Press Ctrl+C to stop")

	// Setup signal handler for clean shutdown
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)

	// Wait for interrupt signal
	<-interruptChan

	// Stop mining
	validatorConsensus.StopMining()

	// Print blockchain state
	fmt.Printf("\nBlockchain height: %d\n", bc.GetChainHeight())
	latestBlock := bc.GetLatestBlock()
	fmt.Printf("Latest block hash: %s\n", latestBlock.Hash)
	fmt.Printf("Latest block validator: %s\n", latestBlock.Validator)
	fmt.Printf("Latest block human proof: %s\n", latestBlock.HumanProof)
} 