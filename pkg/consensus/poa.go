package consensus

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"sync"
	"time"

	"confirmix/pkg/blockchain"
)

// PoAConsensus implements a Proof of Authority consensus mechanism
type PoAConsensus struct {
	blockchain      *blockchain.Blockchain
	privateKey      *ecdsa.PrivateKey
	address         string
	validatorList   []string
	validatorIndex  int
	validatorMutex  sync.Mutex
	blockTime       time.Duration // Time between blocks
	isValidator     bool
	humanProof      string
	blockMutex      sync.Mutex
	stopMining      chan struct{}
	miningActive    bool
}

// NewPoAConsensus creates a new Proof of Authority consensus engine
func NewPoAConsensus(bc *blockchain.Blockchain, privateKey *ecdsa.PrivateKey, address string, blockTime time.Duration, humanProof string) *PoAConsensus {
	return &PoAConsensus{
		blockchain:     bc,
		privateKey:     privateKey,
		address:        address,
		validatorList:  []string{},
		validatorIndex: 0,
		blockTime:      blockTime,
		isValidator:    false,
		humanProof:     humanProof,
		stopMining:     make(chan struct{}),
		miningActive:   false,
	}
}

// RegisterAsValidator registers this node as a validator
func (poa *PoAConsensus) RegisterAsValidator() error {
	if poa.humanProof == "" {
		return errors.New("human proof is required to register as validator")
	}
	
	err := poa.blockchain.AddValidator(poa.address, poa.humanProof)
	if err != nil {
		return err
	}
	
	poa.isValidator = true
	return nil
}

// UpdateValidatorList updates the list of authorized validators
func (poa *PoAConsensus) UpdateValidatorList(validators []string) {
	poa.validatorMutex.Lock()
	defer poa.validatorMutex.Unlock()
	
	poa.validatorList = validators
}

// getCurrentValidator gets the current validator who should create a block
func (poa *PoAConsensus) getCurrentValidator() string {
	poa.validatorMutex.Lock()
	defer poa.validatorMutex.Unlock()
	
	if len(poa.validatorList) == 0 {
		return ""
	}
	
	// Simple round-robin selection
	validator := poa.validatorList[poa.validatorIndex]
	poa.validatorIndex = (poa.validatorIndex + 1) % len(poa.validatorList)
	return validator
}

// StartMining starts the block production process
func (poa *PoAConsensus) StartMining() error {
	poa.blockMutex.Lock()
	if poa.miningActive {
		poa.blockMutex.Unlock()
		return errors.New("mining already active")
	}
	poa.miningActive = true
	poa.blockMutex.Unlock()

	go poa.miningLoop()
	return nil
}

// StopMining stops the block production process
func (poa *PoAConsensus) StopMining() {
	poa.blockMutex.Lock()
	if poa.miningActive {
		poa.stopMining <- struct{}{}
		poa.miningActive = false
	}
	poa.blockMutex.Unlock()
}

// miningLoop is the main loop for block production
func (poa *PoAConsensus) miningLoop() {
	ticker := time.NewTicker(poa.blockTime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !poa.isValidator {
				continue
			}
			
			// Check if it's this validator's turn
			currentValidator := poa.getCurrentValidator()
			if currentValidator != poa.address {
				continue
			}
			
			// Create a new block
			poa.createNewBlock()
			
		case <-poa.stopMining:
			return
		}
	}
}

// createNewBlock creates and adds a new block to the blockchain
func (poa *PoAConsensus) createNewBlock() error {
	// Get pending transactions
	transactions := poa.blockchain.GetPendingTransactions() // Get all pending transactions
	
	// Only create a block if there are pending transactions
	if len(transactions) == 0 {
		return nil
	}

	// Get latest block
	latestBlock := poa.blockchain.GetLatestBlock()
	
	// Create new block
	newBlock := blockchain.NewBlock(
		latestBlock.Index+1,
		transactions,
		latestBlock.Hash,
		poa.address,
		poa.humanProof,
	)
	
	// Sign the block
	signature, err := poa.signBlock(newBlock)
	if err != nil {
		return err
	}
	newBlock.Signature = signature
	
	// Add block to blockchain
	return poa.blockchain.AddBlock(newBlock)
}

// signBlock signs a block using the validator's private key
func (poa *PoAConsensus) signBlock(block *blockchain.Block) ([]byte, error) {
	// Hash the block
	blockHash := block.CalculateHash()
	
	// Convert hash to bytes
	hashBytes := []byte(blockHash)
	
	// Create a hash of the hash for signing
	h := sha256.New()
	h.Write(hashBytes)
	digest := h.Sum(nil)
	
	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, poa.privateKey, digest)
	if err != nil {
		return nil, err
	}
	
	// Combine r and s into signature
	signature := append(r.Bytes(), s.Bytes()...)
	return signature, nil
}

// VerifyBlock verifies that a block is valid according to PoA rules
func (poa *PoAConsensus) VerifyBlock(block *blockchain.Block) error {
	// Verify that the validator is authorized
	if !poa.blockchain.IsValidator(block.Validator) {
		return errors.New("unauthorized validator")
	}
	
	// Verify human proof
	expectedProof := poa.blockchain.GetHumanProof(block.Validator)
	if block.HumanProof != expectedProof {
		return errors.New("invalid human proof")
	}

	// Get latest block
	latestBlock := poa.blockchain.GetLatestBlock()
	
	// Verify block is linked properly
	if block.PrevHash != latestBlock.Hash {
		return errors.New("invalid previous hash")
	}
	
	// Verify block index
	if block.Index != latestBlock.Index+1 {
		return errors.New("invalid block index")
	}
	
	// Verify block hash
	if block.Hash != block.CalculateHash() {
		return errors.New("invalid block hash")
	}
	
	// In a real implementation, we would also verify the signature here
	// For simplicity, we'll skip the actual signature verification
	
	return nil
} 