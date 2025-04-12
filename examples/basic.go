package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"confirmix/pkg/blockchain"
	"confirmix/pkg/consensus"
)

func main() {
	// Create a new blockchain
	bc := blockchain.NewBlockchain()
	fmt.Println("Created new blockchain")
	fmt.Printf("Genesis block hash: %s\n", bc.GetLatestBlock().Hash)

	// Create private keys for validators
	validator1Key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	validator2Key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Create addresses from public keys
	publicKey1 := validator1Key.PublicKey
	publicKeyBytes1 := elliptic.Marshal(publicKey1.Curve, publicKey1.X, publicKey1.Y)
	validator1Address := fmt.Sprintf("%x", publicKeyBytes1[:10])

	publicKey2 := validator2Key.PublicKey
	publicKeyBytes2 := elliptic.Marshal(publicKey2.Curve, publicKey2.X, publicKey2.Y)
	validator2Address := fmt.Sprintf("%x", publicKeyBytes2[:10])

	fmt.Printf("Validator 1 address: %s\n", validator1Address)
	fmt.Printf("Validator 2 address: %s\n", validator2Address)

	// Create consensus engine for validator 1
	consensus1 := consensus.NewHybridConsensus(bc, validator1Key, validator1Address, 5*time.Second)

	// Create consensus engine for validator 2
	consensus2 := consensus.NewHybridConsensus(bc, validator2Key, validator2Address, 5*time.Second)

	// Initiate human verification for validators
	proofToken1, err := consensus1.InitiateHumanVerification()
	if err != nil {
		log.Fatalf("Failed to initiate human verification for validator 1: %v", err)
	}

	proofToken2, err := consensus2.InitiateHumanVerification()
	if err != nil {
		log.Fatalf("Failed to initiate human verification for validator 2: %v", err)
	}

	fmt.Printf("Validator 1 proof token: %s\n", proofToken1)
	fmt.Printf("Validator 2 proof token: %s\n", proofToken2)

	// Complete human verification (simulated)
	err = consensus1.CompleteHumanVerification(proofToken1)
	if err != nil {
		log.Fatalf("Failed to complete human verification for validator 1: %v", err)
	}

	err = consensus2.CompleteHumanVerification(proofToken2)
	if err != nil {
		log.Fatalf("Failed to complete human verification for validator 2: %v", err)
	}

	fmt.Println("Human verification completed for both validators")

	// Register as validators
	err = consensus1.RegisterAsValidator()
	if err != nil {
		log.Fatalf("Failed to register validator 1: %v", err)
	}

	err = consensus2.RegisterAsValidator()
	if err != nil {
		log.Fatalf("Failed to register validator 2: %v", err)
	}

	fmt.Println("Both validators registered")

	// Update validator list
	consensus1.UpdateValidatorList([]string{validator1Address, validator2Address})
	consensus2.UpdateValidatorList([]string{validator1Address, validator2Address})

	// Create a transaction
	tx := &blockchain.Transaction{
		ID:        "tx1",
		From:      "user1",
		To:        "user2",
		Value:     10.0,
		Data:      []byte("Hello, blockchain!"),
		Timestamp: time.Now().Unix(),
	}

	// Add transaction to blockchain
	err = bc.AddTransaction(tx)
	if err != nil {
		log.Fatalf("Failed to add transaction: %v", err)
	}

	fmt.Println("Transaction added to blockchain")

	// Start mining on validator 1
	err = consensus1.StartMining()
	if err != nil {
		log.Fatalf("Failed to start mining: %v", err)
	}

	// Wait for a block to be mined
	fmt.Println("Mining in progress...")
	time.Sleep(10 * time.Second)

	// Stop mining
	consensus1.StopMining()

	// Print blockchain state
	fmt.Printf("Blockchain height: %d\n", bc.GetChainHeight())
	latestBlock := bc.GetLatestBlock()
	fmt.Printf("Latest block hash: %s\n", latestBlock.Hash)
	fmt.Printf("Latest block validator: %s\n", latestBlock.Validator)
	fmt.Printf("Latest block human proof: %s\n", latestBlock.HumanProof)
	fmt.Printf("Number of transactions in latest block: %d\n", len(latestBlock.Transactions))
} 