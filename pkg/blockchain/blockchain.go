package blockchain

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"crypto/sha256"
	"encoding/hex"
)

// Blockchain represents the blockchain data structure
type Blockchain struct {
	Blocks           []*Block
	CurrentDifficult int
	TotalMinted      *big.Int
	WalletPrivate    *rsa.PrivateKey
	WalletPublic     *rsa.PublicKey
	accounts         map[string]*big.Int
	PendingTXs       map[string]*Transaction
	chain_data       string
	validators       map[string]bool
	humanProofs      map[string]string
	lockedBalances   map[string]*big.Int
	mutex            sync.RWMutex
	mu               sync.RWMutex
	pendingTxs       []*Transaction
	txPool           map[string]*Transaction
	contractManager  *ContractManager
	keyPairs         map[string]*KeyPair
	mutex_           sync.RWMutex
	multiSigWallets  map[string]*MultiSigWallet
	Admins           []string
}

// NewBlockchain creates a new blockchain instance
func NewBlockchain() (*Blockchain, error) {
	// Initialize variables
	var genesisReward *big.Int
	var rewardPerOwner *big.Int
	var ok bool

	bc := &Blockchain{
		Blocks:            make([]*Block, 0),
		pendingTxs:        make([]*Transaction, 0),
		accounts:          make(map[string]*big.Int),
		keyPairs:          make(map[string]*KeyPair),
		validators:        make(map[string]bool),
		multiSigWallets:   make(map[string]*MultiSigWallet),
		PendingTXs:        make(map[string]*Transaction),
		txPool:           make(map[string]*Transaction),
		contractManager:  NewContractManager(),
		humanProofs:      make(map[string]string),
		lockedBalances:   make(map[string]*big.Int),
		TotalMinted:      big.NewInt(0),
		CurrentDifficult: 1,
	}

	// Try to load existing blockchain state
	err := bc.LoadFromDisk()
	if err == nil {
		log.Printf("Successfully loaded existing blockchain state")
		return bc, nil
	}

	// If no existing state found, create new blockchain
	log.Printf("No existing blockchain state found, creating new blockchain")

	// Create genesis block
	genesisBlock := &Block{
		Index:        0,
		Timestamp:    time.Now().Unix(),
		Transactions: []*Transaction{},
		PrevHash:     "0",
		Validator:    "genesis",
		HumanProof:   "genesis_proof",
		Reward:       0,
	}

	// Calculate genesis block hash
	genesisBlock.Hash = genesisBlock.CalculateHash()

	// Add genesis block to chain
	bc.Blocks = append(bc.Blocks, genesisBlock)

	// Create genesis admin account (symbolic address)
	adminAddress := "0x0000000000000000000000000000000000000000admin"

	// Create three new wallets for multisig owners
	owner1KeyPair, err := NewKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to create owner1 wallet: %v", err)
	}

	owner2KeyPair, err := NewKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to create owner2 wallet: %v", err)
	}

	owner3KeyPair, err := NewKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to create owner3 wallet: %v", err)
	}

	// Save owner key pairs to files
	if err := owner1KeyPair.SaveToFile(owner1KeyPair.GetAddress()); err != nil {
		log.Printf("Warning: Failed to save owner1 key pair: %v", err)
	}
	if err := owner2KeyPair.SaveToFile(owner2KeyPair.GetAddress()); err != nil {
		log.Printf("Warning: Failed to save owner2 key pair: %v", err)
	}
	if err := owner3KeyPair.SaveToFile(owner3KeyPair.GetAddress()); err != nil {
		log.Printf("Warning: Failed to save owner3 key pair: %v", err)
	}

	// Create genesis multisig wallet with only the three new owners
	genesisOwners := []string{
		owner1KeyPair.GetAddress(), // Owner 1
		owner2KeyPair.GetAddress(), // Owner 2
		owner3KeyPair.GetAddress(), // Owner 3
	}
	requiredSigs := 2 // 2/3 threshold for multisig operations

	genesisMultiSigWallet, err := NewMultiSigWallet(
		adminAddress,
		genesisOwners,
		requiredSigs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create genesis multisig wallet: %v", err)
	}

	// Add Genesis MultiSig wallet to blockchain
	bc.multiSigWallets[genesisMultiSigWallet.Address] = genesisMultiSigWallet

	// Register genesis multisig wallet as validator
	bc.validators[genesisMultiSigWallet.Address] = true
	bc.humanProofs[genesisMultiSigWallet.Address] = "genesis"

	// Add genesis multisig wallet as first admin
	bc.Admins = append(bc.Admins, genesisMultiSigWallet.Address)

	// Calculate genesis block reward (50 ConX tokens with 18 decimals)
	genesisReward = new(big.Int)
	_, ok = genesisReward.SetString("50000000000000000000", 10) // 50 tokens with 18 decimals
	if !ok {
		return nil, fmt.Errorf("failed to set genesis reward")
	}

	// Calculate reward per owner (total reward divided by 3)
	rewardPerOwner = new(big.Int).Div(genesisReward, big.NewInt(3))

	// Distribute rewards to each owner
	for _, owner := range genesisOwners {
		// Create account for owner if it doesn't exist
		if _, exists := bc.accounts[owner]; !exists {
			bc.accounts[owner] = big.NewInt(0)
		}
		
		// Add reward to owner's balance
		bc.accounts[owner] = new(big.Int).Add(bc.accounts[owner], rewardPerOwner)
		log.Printf("Distributed %s tokens to owner %s", rewardPerOwner.String(), owner)
	}

	// Update total minted tokens
	bc.TotalMinted = new(big.Int).Add(bc.TotalMinted, genesisReward)
	log.Printf("Total minted tokens after genesis block: %s", bc.TotalMinted.String())

	// Save multisig wallet info to file
	multisigInfo := struct {
		Address      string   `json:"address"`
		Owners       []string `json:"owners"`
		RequiredSigs int      `json:"required_sigs"`
		AdminWallet  string   `json:"admin_wallet"`
		CreatedAt    int64    `json:"created_at"`
		Description  string   `json:"description"`
	}{
		Address:      genesisMultiSigWallet.Address,
		Owners:       genesisOwners,
		RequiredSigs: requiredSigs,
		AdminWallet:  adminAddress,
		CreatedAt:    time.Now().Unix(),
		Description:  "Genesis multisig wallet with 3 owners and 2/3 threshold",
	}

	multisigData, err := json.MarshalIndent(multisigInfo, "", "  ")
	if err != nil {
		log.Printf("Warning: Failed to marshal multisig info: %v", err)
	} else {
		if err := os.WriteFile("data/multisig.json", multisigData, 0644); err != nil {
			log.Printf("Warning: Failed to save multisig info: %v", err)
		}
	}

	// Create admin multisig wallet
	adminWallet, err := NewMultiSigWallet(adminAddress, []string{adminAddress}, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin multisig wallet: %v", err)
	}
	bc.multiSigWallets[adminAddress] = adminWallet
	
	// Add admin wallet as a validator
	bc.validators[adminAddress] = true
	bc.humanProofs[adminAddress] = "admin"
	
	// Create genesis block
	genesisBlock = &Block{
		Index:        0,
		Timestamp:    time.Now().Unix(),
		Transactions: []*Transaction{},
		PrevHash:     "0",
		Validator:    "genesis",
		HumanProof:   "genesis_proof",
		Reward:       0,
	}

	// Calculate genesis block hash
	genesisBlock.Hash = genesisBlock.CalculateHash()

	// Add genesis block to chain
	bc.Blocks = append(bc.Blocks, genesisBlock)

	// Calculate genesis block reward (50 ConX tokens with 18 decimals)
	genesisReward = new(big.Int)
	_, ok = genesisReward.SetString("50000000000000000000", 10) // 50 tokens with 18 decimals
	if !ok {
		return nil, fmt.Errorf("failed to set genesis reward")
	}

	// Calculate reward per owner (total reward divided by 3)
	rewardPerOwner = new(big.Int).Div(genesisReward, big.NewInt(3))

	// Distribute rewards to each owner
	for _, owner := range genesisOwners {
		// Create account for owner if it doesn't exist
		if _, exists := bc.accounts[owner]; !exists {
			bc.accounts[owner] = big.NewInt(0)
		}
		
		// Add reward to owner's balance
		bc.accounts[owner] = new(big.Int).Add(bc.accounts[owner], rewardPerOwner)
		log.Printf("Distributed %s tokens to owner %s", rewardPerOwner.String(), owner)
	}

	// Update total minted tokens
	bc.TotalMinted = new(big.Int).Add(bc.TotalMinted, genesisReward)
	log.Printf("Total minted tokens after genesis block: %s", bc.TotalMinted.String())

	// Save multisig wallet info to file
	multisigInfo = struct {
		Address      string   `json:"address"`
		Owners       []string `json:"owners"`
		RequiredSigs int      `json:"required_sigs"`
		AdminWallet  string   `json:"admin_wallet"`
		CreatedAt    int64    `json:"created_at"`
		Description  string   `json:"description"`
	}{
		Address:      genesisMultiSigWallet.Address,
		Owners:       genesisOwners,
		RequiredSigs: requiredSigs,
		AdminWallet:  adminAddress,
		CreatedAt:    time.Now().Unix(),
		Description:  "Genesis multisig wallet with 3 owners and 2/3 threshold",
	}

	multisigData, err = json.MarshalIndent(multisigInfo, "", "  ")
	if err != nil {
		log.Printf("Warning: Failed to marshal multisig info: %v", err)
	} else {
		if err := os.WriteFile("data/multisig.json", multisigData, 0644); err != nil {
			log.Printf("Warning: Failed to save multisig info: %v", err)
		}
	}

	// Save initial state
	if err := bc.SaveToDisk(); err != nil {
		return nil, fmt.Errorf("failed to save initial state: %v", err)
	}

	log.Printf("Genesis block created with multisig wallet address: %s", genesisMultiSigWallet.Address)
	log.Printf("Admin wallet address (symbolic): %s", adminAddress)
	log.Printf("Genesis MultiSig wallet initialized with %d owners", len(genesisOwners))
	log.Printf("Multisig wallet info saved to data/multisig.json")
	log.Printf("Required signatures for multisig operations: %d", requiredSigs)
	log.Printf("Owner addresses and their key pairs:")
	log.Printf("  Owner 1: %s (saved to data/key_%s.json)", owner1KeyPair.GetAddress(), owner1KeyPair.GetAddress())
	log.Printf("  Owner 2: %s (saved to data/key_%s.json)", owner2KeyPair.GetAddress(), owner2KeyPair.GetAddress())
	log.Printf("  Owner 3: %s (saved to data/key_%s.json)", owner3KeyPair.GetAddress(), owner3KeyPair.GetAddress())

	return bc, nil
}

// GetBlockchainDataPath returns the path where blockchain data is stored
func GetBlockchainDataPath() string {
	// Create data directory in project root
	dataDir := "data"
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		log.Printf("Failed to create data directory: %v", err)
	}
	
	return dataDir
}

// SaveToDisk persists the blockchain state to disk
func (bc *Blockchain) SaveToDisk() error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	dataDir := GetBlockchainDataPath()
	
	// Save blocks
	blocksFile := filepath.Join(dataDir, "blocks.json")
	blocksData, err := json.MarshalIndent(bc.Blocks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal blocks: %v", err)
	}
	
	if err := ioutil.WriteFile(blocksFile, blocksData, 0644); err != nil {
		return fmt.Errorf("failed to write blocks file: %v", err)
	}
	
	// Save validators
	validatorsMap := make(map[string]string)
	for addr := range bc.validators {
		validatorsMap[addr] = bc.humanProofs[addr]
	}
	
	validatorsFile := filepath.Join(dataDir, "validators.json")
	validatorsData, err := json.MarshalIndent(validatorsMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal validators: %v", err)
	}
	
	if err := ioutil.WriteFile(validatorsFile, validatorsData, 0644); err != nil {
		return fmt.Errorf("failed to write validators file: %v", err)
	}
	
	// Save accounts
	accountsMap := make(map[string]string)
	for addr, balance := range bc.accounts {
		accountsMap[addr] = balance.String()
	}
	
	accountsFile := filepath.Join(dataDir, "accounts.json")
	accountsData, err := json.MarshalIndent(accountsMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal accounts: %v", err)
	}
	
	if err := ioutil.WriteFile(accountsFile, accountsData, 0644); err != nil {
		return fmt.Errorf("failed to write accounts file: %v", err)
	}

	// Save multi-signature wallets
	multiSigFile := filepath.Join(dataDir, "multisig.json")
	multiSigData, err := json.MarshalIndent(bc.multiSigWallets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal multi-signature wallets: %v", err)
	}
	
	if err := ioutil.WriteFile(multiSigFile, multiSigData, 0644); err != nil {
		return fmt.Errorf("failed to write multi-signature wallets file: %v", err)
	}
	
	log.Printf("Blockchain state saved to disk: %s", dataDir)
	return nil
}

// LoadFromDisk loads the blockchain state from disk
func (bc *Blockchain) LoadFromDisk() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	dataDir := GetBlockchainDataPath()
	
	// Load blocks
	blocksFile := filepath.Join(dataDir, "blocks.json")
	if _, err := os.Stat(blocksFile); os.IsNotExist(err) {
		log.Println("No existing blockchain data found")
		return errors.New("no existing blockchain data found")
	}
	
	blocksData, err := ioutil.ReadFile(blocksFile)
	if err != nil {
		log.Printf("Failed to read blocks file: %v", err)
		return err
	}
	
	var blocks []*Block
	if err := json.Unmarshal(blocksData, &blocks); err != nil {
		log.Printf("Failed to unmarshal blocks: %v", err)
		return err
	}
	
	bc.Blocks = blocks
	
	// Load validators
	validatorsFile := filepath.Join(dataDir, "validators.json")
	if _, err := os.Stat(validatorsFile); !os.IsNotExist(err) {
		validatorsData, err := ioutil.ReadFile(validatorsFile)
		if err == nil {
			var validatorsMap map[string]string
			if err := json.Unmarshal(validatorsData, &validatorsMap); err == nil {
				bc.validators = make(map[string]bool)
				bc.humanProofs = make(map[string]string)
				
				for addr, proof := range validatorsMap {
					bc.validators[addr] = true
					bc.humanProofs[addr] = proof
				}
			}
		}
	}
	
	// Load accounts
	accountsFile := filepath.Join(dataDir, "accounts.json")
	accountsData, err := ioutil.ReadFile(accountsFile)
	if err != nil {
		log.Printf("Failed to read accounts file: %v", err)
		return err
	}
	
	var accountsMap map[string]string
	if err := json.Unmarshal(accountsData, &accountsMap); err != nil {
		log.Printf("Failed to unmarshal accounts: %v", err)
		return err
	}
	
	bc.accounts = make(map[string]*big.Int)
	for addr, balanceStr := range accountsMap {
		balance := new(big.Int)
		success := false
		if balanceStr != "" {
			_, success = balance.SetString(balanceStr, 10)
		}
		
		if !success {
			log.Printf("Invalid balance format for %s: %s, setting to 0", addr, balanceStr)
			balance = big.NewInt(0)
		}
		
		bc.accounts[addr] = balance
		log.Printf("Loaded account %s with balance %s", addr, balance.String())
	}

	// Load multi-signature wallets
	multiSigFile := filepath.Join(dataDir, "multisig.json")
	if _, err := os.Stat(multiSigFile); !os.IsNotExist(err) {
		multiSigData, err := ioutil.ReadFile(multiSigFile)
		if err == nil {
			var multiSigInfo struct {
				Address      string   `json:"address"`
				Owners       []string `json:"owners"`
				RequiredSigs int      `json:"required_sigs"`
				AdminWallet  string   `json:"admin_wallet"`
				CreatedAt    int64    `json:"created_at"`
				Description  string   `json:"description"`
			}
			if err := json.Unmarshal(multiSigData, &multiSigInfo); err == nil {
				// Create multisig wallet
				wallet, err := NewMultiSigWallet(
					multiSigInfo.AdminWallet,
					multiSigInfo.Owners,
					multiSigInfo.RequiredSigs,
				)
				if err == nil {
					bc.multiSigWallets[wallet.Address] = wallet
				}
			}
		}
	}
	
	log.Printf("Blockchain state loaded from disk: %s", dataDir)
	log.Printf("Loaded %d blocks, %d pending transactions, %d accounts, %d multi-signature wallets", 
		len(bc.Blocks), len(bc.txPool), len(bc.accounts), len(bc.multiSigWallets))
	
	return nil
}

// AddValidator adds a new authorized validator to the blockchain
func (bc *Blockchain) AddValidator(address string, humanProof string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Check if human proof is valid (in a real implementation, this would verify with PoH)
	if humanProof == "" {
		return errors.New("human proof is required for validators")
	}
	
	// Generate a new key pair for the validator
	keyPair, err := NewKeyPair()
	if err != nil {
		return err
	}
	
	bc.validators[address] = true
	bc.humanProofs[address] = humanProof
	bc.keyPairs[address] = keyPair
	return nil
}

// IsValidator checks if an address is an authorized validator
func (bc *Blockchain) IsValidator(address string) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.validators[address]
}

// GetHumanProof gets the human proof for a validator
func (bc *Blockchain) GetHumanProof(address string) string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.humanProofs[address]
}

// AddTransaction adds a new transaction to the pending transactions pool
func (bc *Blockchain) AddTransaction(tx *Transaction) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Validate transaction
	if tx == nil {
		return errors.New("transaction is nil")
	}

	// Check if transaction already exists
	if _, exists := bc.txPool[tx.ID]; exists {
		return errors.New("transaction already exists")
	}

	// Add to pending transactions
	bc.txPool[tx.ID] = tx
	bc.pendingTxs = append(bc.pendingTxs, tx)
	return nil
}

// GetPendingTransactions returns the list of pending transactions
func (bc *Blockchain) GetPendingTransactions() []*Transaction {
	bc.mu.RLock()
	
	transactionCount := len(bc.pendingTxs)
	
	// Hızlı bir kopya oluştur ve kilidi bırak
	result := make([]*Transaction, transactionCount)
	for i := 0; i < transactionCount && i < len(bc.pendingTxs); i++ {
		result[i] = bc.pendingTxs[i]
	}
	
	bc.mu.RUnlock()
	
	return result
}

// AddBlock adds a new block to the blockchain
func (bc *Blockchain) AddBlock(block *Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Verify block index
	if uint64(len(bc.Blocks)) != block.Index {
		return fmt.Errorf("invalid block index: expected %d, got %d", len(bc.Blocks), block.Index)
	}
	
	// Verify previous hash
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	if prevBlock.Hash != block.PrevHash {
		return fmt.Errorf("invalid previous hash: expected %s, got %s", prevBlock.Hash, block.PrevHash)
	}
	
	// Check if validator is authorized by admin multisig wallet
	adminWallet, err := bc.GetMultiSigWallet("0x0000000000000000000000000000000000000000admin")
	if err != nil {
		return fmt.Errorf("admin multisig wallet not found: %v", err)
	}
	
	// Check if validator is an owner of admin multisig wallet
	isAuthorized := false
	for _, owner := range adminWallet.Owners {
		if owner == block.Validator {
			isAuthorized = true
			break
		}
	}
	
	if !isAuthorized {
		return fmt.Errorf("validator %s is not authorized by admin multisig wallet", block.Validator)
	}
	
	// Get the block's signatures
	signatures := block.GetSignatures()
	requiredSignatures := adminWallet.RequiredSigs
	validSignatures := 0
	var signers []string
	
	// Count valid signatures from admin wallet owners
	for _, owner := range adminWallet.Owners {
		if _, exists := signatures[owner]; exists {
			validSignatures++
			signers = append(signers, owner)
		}
	}
	
	if validSignatures < requiredSignatures {
		return fmt.Errorf("insufficient signatures from admin multisig wallet owners: got %d, need %d", validSignatures, requiredSignatures)
	}
	
	// Set multisig validation information
	block.IsMultisig = true
	block.Signers = signers
	block.RequiredSigners = requiredSignatures
	block.Signatures = signatures
	
	// Add the block
	bc.Blocks = append(bc.Blocks, block)
	
	// Process all transactions
	var errMsgs []string
	
	// Create a mining reward transaction for the validator
	rewardAmount := bc.GetRewardAmount()
	if rewardAmount.Cmp(big.NewInt(0)) > 0 {
		// Convert big.Int to uint64 for the transaction
		rewardUint64 := uint64(0)
		if rewardAmount.IsUint64() {
			rewardUint64 = rewardAmount.Uint64()
		} else {
			// If reward is too large for uint64, cap it
			log.Printf("Warning: Reward amount is too large for uint64, capping it")
			rewardUint64 = ^uint64(0) // Maximum uint64 value
		}
		
		// If validator is a multisig wallet, distribute reward equally among owners
		if block.IsMultisig {
			// Calculate reward per owner (total reward divided by number of owners)
			rewardPerOwner := new(big.Int).Div(rewardAmount, big.NewInt(int64(len(signers))))
			rewardPerOwnerUint64 := rewardPerOwner.Uint64()
			
			// Create reward transactions for each owner
			for _, owner := range signers {
				rewardTx := &Transaction{
					ID:        fmt.Sprintf("reward_%d_%s", block.Index, owner),
					From:      "confirmix_genesis_address",
					To:        owner,
					Value:     rewardPerOwnerUint64,
					Timestamp: block.Timestamp,
					Type:      "reward",
				}
				
				// Add the reward transaction to the block
				block.Transactions = append(block.Transactions, rewardTx)
				
				// Update balances for the reward transaction
				if err := bc.UpdateBalances(rewardTx); err != nil {
					errMsgs = append(errMsgs, fmt.Sprintf("failed to process reward transaction for %s: %v", owner, err))
				}
			}
		} else {
			// For non-multisig validators, give full reward to the validator
			rewardTx := &Transaction{
				ID:        fmt.Sprintf("reward_%d", block.Index),
				From:      "confirmix_genesis_address",
				To:        block.Validator,
				Value:     rewardUint64,
				Timestamp: block.Timestamp,
				Type:      "reward",
			}
			
			// Add the reward transaction to the block
			block.Transactions = append(block.Transactions, rewardTx)
			
			// Update balances for the reward transaction
			if err := bc.UpdateBalances(rewardTx); err != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("failed to process reward transaction: %v", err))
			}
		}
	}
	
	// Process all user transactions
	for _, tx := range block.Transactions {
		// Skip the reward transaction as it was already processed
		if tx.Type == "reward" {
			continue
		}
		
		// Update balances
		if err := bc.UpdateBalances(tx); err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("failed to process transaction %s: %v", tx.ID, err))
			continue
		}
		
		// Process contract transaction if applicable
		if tx.IsContractTransaction() {
			if err := bc.processContractTransaction(tx); err != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("failed to process contract transaction %s: %v", tx.ID, err))
			}
		}
	}
	
	// Clean transaction pool
	bc.cleanTransactionPool(block.Transactions)
	
	// Save blockchain state
	if err := bc.SaveToDisk(); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("failed to save blockchain state: %v", err))
	}
	
	if len(errMsgs) > 0 {
		return fmt.Errorf("block added with errors: %s", strings.Join(errMsgs, "; "))
	}
	
	return nil
}

// verifyBlockSignature verifies the signature of a block
func (bc *Blockchain) verifyBlockSignature(block *Block) error {
	// Get the public key for the block validator
	keyPair, exists := bc.keyPairs[block.Validator]
	if !exists {
		return errors.New("validator's public key not found")
	}
	
	return block.Verify(keyPair.PublicKey)
}

// processContractTransaction processes a contract transaction
func (bc *Blockchain) processContractTransaction(tx *Transaction) error {
	// Parse contract transaction data
	contractTx, err := ParseContractTransaction(tx.Data)
	if err != nil {
		return err
	}
	
	// Process based on contract operation
	switch contractTx.Operation {
	case "deploy":
		// Deploy a new contract
		_, err := bc.contractManager.DeployContract(contractTx.Code, tx.From)
		return err
		
	case "call":
		// Call a contract function
		_, err := bc.contractManager.CallContract(
			contractTx.ContractAddress,
			contractTx.Function,
			contractTx.Parameters,
			tx.From,
		)
		return err
		
	default:
		return errors.New("unknown contract operation")
	}
}

// cleanTransactionPool removes transactions that were included in a block
func (bc *Blockchain) cleanTransactionPool(txs []*Transaction) {
	for _, tx := range txs {
		delete(bc.txPool, tx.ID)
		
		// Also remove from pending transactions
		for i, pendingTx := range bc.pendingTxs {
			if pendingTx.ID == tx.ID {
				bc.pendingTxs = append(bc.pendingTxs[:i], bc.pendingTxs[i+1:]...)
				break
			}
		}
	}
}

// GetChainHeight returns the current height (length) of the blockchain
func (bc *Blockchain) GetChainHeight() uint64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return uint64(len(bc.Blocks) - 1)
}

// GetLatestBlock returns the latest block in the blockchain
func (bc *Blockchain) GetLatestBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.Blocks[len(bc.Blocks)-1]
}

// GetBlock returns a block by its hash
func (bc *Blockchain) GetBlock(hash string) (*Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	for _, block := range bc.Blocks {
		if block.Hash == hash {
			return block, nil
		}
	}
	
	return nil, errors.New("block not found")
}

// GetBlockByIndex returns a block by its index
func (bc *Blockchain) GetBlockByIndex(index uint64) (*Block, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	if index >= uint64(len(bc.Blocks)) {
		return nil, errors.New("block index out of range")
	}
	
	return bc.Blocks[index], nil
}

// GetContractManager returns the contract manager
func (bc *Blockchain) GetContractManager() *ContractManager {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.contractManager
}

// GetTransaction returns a transaction by ID
func (bc *Blockchain) GetTransaction(id string) (*Transaction, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	tx, exists := bc.txPool[id]
	return tx, exists
}

// GetKeyPair returns the key pair for an address
func (bc *Blockchain) GetKeyPair(address string) (*KeyPair, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	keyPair, exists := bc.keyPairs[address]
	return keyPair, exists
}

// AddKeyPair adds a key pair for an address to the blockchain
func (bc *Blockchain) AddKeyPair(address string, keyPair *KeyPair) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.keyPairs[address] = keyPair
	
	// Save the blockchain state after adding a key pair
	go bc.SaveToDisk()
}

// GetAllAddresses returns all addresses with key pairs in the blockchain
func (bc *Blockchain) GetAllAddresses() []string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	addresses := make([]string, 0, len(bc.keyPairs))
	for addr := range bc.keyPairs {
		addresses = append(addresses, addr)
	}
	return addresses
}

// CreateAccount creates a new account with initial balance
func (bc *Blockchain) CreateAccount(address string, initialBalance *big.Int) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	if _, exists := bc.accounts[address]; exists {
		return errors.New("account already exists")
	}
	
	bc.accounts[address] = initialBalance
	return nil
}

// GetBalance gets the balance of an account
func (bc *Blockchain) GetBalance(address string) (*big.Int, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	balance, exists := bc.accounts[address]
	if !exists {
		return big.NewInt(0), nil // Return 0 if account doesn't exist
	}
	
	return balance, nil
}

// FormatBalance formats a balance with the correct number of decimals
func (bc *Blockchain) FormatBalance(balance *big.Int) string {
	// Convert balance to string
	balanceStr := balance.String()
	
	// If balance is 0, return "0"
	if balanceStr == "0" {
		return "0"
	}
	
	// Add leading zeros if needed
	for len(balanceStr) < 18 {
		balanceStr = "0" + balanceStr
	}
	
	// Insert decimal point
	decimalPoint := len(balanceStr) - 18
	if decimalPoint > 0 {
		balanceStr = balanceStr[:decimalPoint] + "." + balanceStr[decimalPoint:]
	} else {
		balanceStr = "0." + balanceStr
	}
	
	// Remove trailing zeros after decimal point
	balanceStr = strings.TrimRight(balanceStr, "0")
	balanceStr = strings.TrimRight(balanceStr, ".")
	
	return balanceStr
}

// GetFormattedBalance gets the formatted balance of an account
func (bc *Blockchain) GetFormattedBalance(address string) (string, error) {
	balance, err := bc.GetBalance(address)
	if err != nil {
		return "", err
	}
	
	return bc.FormatBalance(balance), nil
}

// UpdateBalances updates account balances based on a transaction
func (bc *Blockchain) UpdateBalances(tx *Transaction) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	// Convert the uint64 value to big.Int
	txValue := new(big.Int).SetUint64(tx.Value)
	
	// Reward transaction handling
	if tx.Type == "reward" {
		// Add rewards to validator account
		currentBalance, exists := bc.accounts[tx.To]
		if !exists {
			currentBalance = big.NewInt(0)
		}
		bc.accounts[tx.To] = new(big.Int).Add(currentBalance, txValue)
		return nil
	}
	
	// Regular transaction handling
	if tx.From == tx.To {
		return errors.New("sender and recipient cannot be the same")
	}
	
	fromBalance, exists := bc.accounts[tx.From]
	if !exists {
		return errors.New("sender account does not exist")
	}
	
	// Check if sender has enough funds
	if fromBalance.Cmp(txValue) < 0 {
		return errors.New("insufficient funds")
	}
	
	// Update sender's balance
	bc.accounts[tx.From] = new(big.Int).Sub(fromBalance, txValue)
	
	// Update recipient's balance
	toBalance, exists := bc.accounts[tx.To]
	if !exists {
		toBalance = big.NewInt(0)
	}
	bc.accounts[tx.To] = new(big.Int).Add(toBalance, txValue)
	
	return nil
}

// ValidatorInfo represents information about a validator
type ValidatorInfo struct {
	Address    string `json:"address"`
	HumanProof string `json:"humanProof"`
}

// GetValidators returns the list of registered validators
func (bc *Blockchain) GetValidators() []ValidatorInfo {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	validators := make([]ValidatorInfo, 0, len(bc.validators))
	for addr := range bc.validators {
		validators = append(validators, ValidatorInfo{
			Address:    addr,
			HumanProof: bc.humanProofs[addr],
		})
	}
	return validators
}

// RemoveTransaction removes a transaction from the pool by ID
func (bc *Blockchain) RemoveTransaction(txID string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Check if transaction exists in the pool
	if _, exists := bc.txPool[txID]; !exists {
		return fmt.Errorf("transaction %s not found in pool", txID)
	}
	
	// Remove from transaction pool
	delete(bc.txPool, txID)
	
	// Also remove from pending transactions
	for i, tx := range bc.pendingTxs {
		if tx.ID == txID {
			// Remove by replacing with the last element and then truncating
			bc.pendingTxs[i] = bc.pendingTxs[len(bc.pendingTxs)-1]
			bc.pendingTxs = bc.pendingTxs[:len(bc.pendingTxs)-1]
			break
		}
	}
	
	return nil
}

// GetRewardAmount returns the amount of ConX tokens to be rewarded for mining a block
// This implements a halving schedule for rewards
func (bc *Blockchain) GetRewardAmount() *big.Int {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	
	// Base reward: 50 ConX tokens with 18 decimals
	baseReward := new(big.Int)
	baseReward.SetString("50000000000000000000", 10) // 50 tokens with 18 decimals
	
	// Determine the reward epoch (halving every 210,000 blocks, similar to Bitcoin)
	blockHeight := uint64(len(bc.Blocks))
	halvingInterval := uint64(210000)
	epoch := blockHeight / halvingInterval
	
	// Calculate the reward based on the epoch (halving)
	if epoch > 0 {
		// Calculate 2^epoch for the divisor
		divisor := big.NewInt(1)
		for i := uint64(0); i < epoch; i++ {
			divisor.Mul(divisor, big.NewInt(2))
		}
		
		// Apply the divisor
		reward := new(big.Int).Div(baseReward, divisor)
		log.Printf("Current block height: %d, Epoch: %d, Reward: %s", blockHeight, epoch, reward.String())
		log.Printf("Total minted tokens: %s", bc.TotalMinted.String())
		return reward
	}
	
	log.Printf("Current block height: %d, Epoch: %d, Reward: %s", blockHeight, epoch, baseReward.String())
	log.Printf("Total minted tokens: %s", bc.TotalMinted.String())
	return baseReward
}

// MineBlock creates a new block with pending transactions
func (bc *Blockchain) MineBlock(validatorAddress string) (*Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Check if the validator is authorized
	if !bc.IsValidator(validatorAddress) {
		return nil, errors.New("unauthorized validator")
	}

	// Get pending transactions
	pendingTxs := bc.GetPendingTransactions()
	if len(pendingTxs) == 0 {
		return nil, errors.New("no pending transactions")
	}

	// Create new block
	prevBlock := bc.GetLatestBlock()
	block := &Block{
		Index:        uint64(len(bc.Blocks)),
		Timestamp:    time.Now().Unix(),
		Transactions: pendingTxs,
		PrevHash:     prevBlock.Hash,
		Validator:    validatorAddress,
		Reward:       bc.GetRewardAmount().Uint64(), // Convert big.Int to uint64
	}

	// Calculate block hash
	block.Hash = block.CalculateHash()

	// Sign block with validator's private key
	keyPair, exists := bc.GetKeyPair(validatorAddress)
	if !exists {
		return nil, errors.New("validator key pair not found")
	}

	if err := block.Sign(keyPair.PrivateKey); err != nil {
		return nil, fmt.Errorf("failed to sign block: %v", err)
	}

	// Add block to chain
	if err := bc.AddBlock(block); err != nil {
		return nil, err
	}

	// Clean transaction pool
	bc.cleanTransactionPool(pendingTxs)

	return block, nil
}

// RegisterValidator registers an address as a validator
func (bc *Blockchain) RegisterValidator(address string, humanProof string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Check if already a validator
	if _, exists := bc.validators[address]; exists {
		return errors.New("address is already a validator")
	}
	
	// Add to validators map
	bc.validators[address] = true
	
	// Store human proof
	bc.humanProofs[address] = humanProof
	
	log.Printf("Validator registered: %s with human proof: %s", address, humanProof)
	
	// Save changes to disk
	go bc.SaveToDisk()
	
	return nil
}

// RemoveValidator removes an address from the validator set
func (bc *Blockchain) RemoveValidator(address string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Check if the address is a validator
	if _, exists := bc.validators[address]; !exists {
		return errors.New("address is not a validator")
	}
	
	// Remove from validators map
	delete(bc.validators, address)
	
	// We keep the human proof in case they are re-added later
	
	log.Printf("Validator removed: %s", address)
	
	// Save changes to disk
	go bc.SaveToDisk()
	
	return nil
}

// Lock locks tokens for governance or staking
func (bc *Blockchain) Lock(address string, amount *big.Int) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	// Check if the address has sufficient balance
	balance, exists := bc.accounts[address]
	if !exists {
		return fmt.Errorf("address %s not found", address)
	}
	
	// Check if balance is sufficient
	if balance.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient balance: have %s, trying to lock %s", 
			balance.String(), amount.String())
	}
	
	// Initialize locked balance if it doesn't exist
	if _, exists := bc.lockedBalances[address]; !exists {
		bc.lockedBalances[address] = big.NewInt(0)
	}
	
	// Update balances
	bc.accounts[address] = new(big.Int).Sub(balance, amount)
	bc.lockedBalances[address] = new(big.Int).Add(bc.lockedBalances[address], amount)
	
	// Save the updated state
	return bc.SaveToDisk()
}

// Unlock unlocks tokens that were previously locked
func (bc *Blockchain) Unlock(address string, amount *big.Int) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	// Check if the address has locked tokens
	lockedBalance, exists := bc.lockedBalances[address]
	if !exists || lockedBalance.Cmp(big.NewInt(0)) == 0 {
		return fmt.Errorf("address %s has no locked tokens", address)
	}
	
	// Check if locked balance is sufficient
	if lockedBalance.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient locked balance: have %s locked, trying to unlock %s", 
			lockedBalance.String(), amount.String())
	}
	
	// Initialize account if it doesn't exist
	if _, exists := bc.accounts[address]; !exists {
		bc.accounts[address] = big.NewInt(0)
	}
	
	// Update balances
	bc.lockedBalances[address] = new(big.Int).Sub(lockedBalance, amount)
	bc.accounts[address] = new(big.Int).Add(bc.accounts[address], amount)
	
	// Save the updated state
	return bc.SaveToDisk()
}

// GetLockedBalance returns the locked balance for an address
func (bc *Blockchain) GetLockedBalance(address string) (*big.Int, error) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()
	
	lockedBalance, exists := bc.lockedBalances[address]
	if !exists {
		return big.NewInt(0), nil
	}
	
	return new(big.Int).Set(lockedBalance), nil
}

// TransferFrom transfers tokens from one address to another
// Used for governance operations like treasury transfers
func (bc *Blockchain) TransferFrom(from, to string, amount *big.Int) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	
	// Check if the source address exists and has sufficient balance
	fromBalance, exists := bc.accounts[from]
	if !exists {
		return fmt.Errorf("source address %s not found", from)
	}
	
	// Check if balance is sufficient
	if fromBalance.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient balance: have %s, trying to transfer %s", 
			fromBalance.String(), amount.String())
	}
	
	// Initialize target account if it doesn't exist
	if _, exists := bc.accounts[to]; !exists {
		bc.accounts[to] = big.NewInt(0)
	}
	
	// Update balances
	bc.accounts[from] = new(big.Int).Sub(fromBalance, amount)
	bc.accounts[to] = new(big.Int).Add(bc.accounts[to], amount)
	
	// Save the updated state
	return bc.SaveToDisk()
}

// initialize initializes a new blockchain
func (bc *Blockchain) initialize() {
	bc.Blocks = []*Block{}
	bc.CurrentDifficult = 1
	bc.TotalMinted = big.NewInt(0)
	
	// Initialize maps
	bc.accounts = make(map[string]*big.Int)
	bc.PendingTXs = make(map[string]*Transaction)
	bc.pendingTxs = make([]*Transaction, 0)
	bc.txPool = make(map[string]*Transaction)
	bc.validators = make(map[string]bool)
	bc.humanProofs = make(map[string]string)
	bc.lockedBalances = make(map[string]*big.Int)
	bc.contractManager = NewContractManager()
	bc.keyPairs = make(map[string]*KeyPair)
	
	// Initialize total supply
	totalSupply := new(big.Int)
	totalSupply.SetString("100000000000000000000", 10) // 100 million tokens with 18 decimals
	
	// Add genesis block
	bc.AddGenesisBlock(totalSupply)
}

// AddGenesisBlock adds the genesis block to the blockchain with initial supply
func (bc *Blockchain) AddGenesisBlock(totalSupply *big.Int) {
	// Step 1: Create Admin Wallet (Genesis Validator) - Symbolic address only
	adminAddress := "0x0000000000000000000000000000000000000000admin" // Genesis admin address

	// Step 2: Create Three New Wallets for Multisig Owners
	owner1KeyPair, err := NewKeyPair()
	if err != nil {
		log.Fatalf("Failed to create owner1 wallet: %v", err)
	}

	owner2KeyPair, err := NewKeyPair()
	if err != nil {
		log.Fatalf("Failed to create owner2 wallet: %v", err)
	}

	owner3KeyPair, err := NewKeyPair()
	if err != nil {
		log.Fatalf("Failed to create owner3 wallet: %v", err)
	}

	// Save owner key pairs to files
	if err := owner1KeyPair.SaveToFile(owner1KeyPair.GetAddress()); err != nil {
		log.Printf("Warning: Failed to save owner1 key pair: %v", err)
	}
	if err := owner2KeyPair.SaveToFile(owner2KeyPair.GetAddress()); err != nil {
		log.Printf("Warning: Failed to save owner2 key pair: %v", err)
	}
	if err := owner3KeyPair.SaveToFile(owner3KeyPair.GetAddress()); err != nil {
		log.Printf("Warning: Failed to save owner3 key pair: %v", err)
	}

	// Step 3: Create Multisig Wallet Structure with only the three new owners
	genesisOwners := []string{
		owner1KeyPair.GetAddress(), // Owner 1
		owner2KeyPair.GetAddress(), // Owner 2
		owner3KeyPair.GetAddress(), // Owner 3
	}
	requiredSigs := 2 // 2/3 threshold for multisig operations

	// Create Genesis MultiSig wallet
	genesisMultiSigWallet, err := NewMultiSigWallet(
		adminAddress,
		genesisOwners,
		requiredSigs,
	)
	if err != nil {
		log.Fatalf("Failed to create genesis multisig wallet: %v", err)
	}

	// Add Genesis MultiSig wallet to blockchain
	bc.multiSigWallets[genesisMultiSigWallet.Address] = genesisMultiSigWallet

	// Register genesis multisig wallet as validator
	bc.validators[genesisMultiSigWallet.Address] = true
	bc.humanProofs[genesisMultiSigWallet.Address] = "genesis"

	// Add genesis multisig wallet as first admin
	bc.Admins = append(bc.Admins, genesisMultiSigWallet.Address)

	// Calculate reward per owner (total supply divided by 3)
	rewardPerOwner := new(big.Int).Div(totalSupply, big.NewInt(3))

	// Distribute rewards to each owner
	for _, owner := range genesisOwners {
		// Create account for owner if it doesn't exist
		if _, exists := bc.accounts[owner]; !exists {
			bc.accounts[owner] = big.NewInt(0)
		}
		
		// Add reward to owner's balance
		bc.accounts[owner] = new(big.Int).Add(bc.accounts[owner], rewardPerOwner)
		log.Printf("Distributed %s tokens to owner %s", rewardPerOwner.String(), owner)
	}

	// Save multisig wallet info to file
	multisigInfo := struct {
		Address      string   `json:"address"`
		Owners       []string `json:"owners"`
		RequiredSigs int      `json:"required_sigs"`
		AdminWallet  string   `json:"admin_wallet"`
		CreatedAt    int64    `json:"created_at"`
		Description  string   `json:"description"`
	}{
		Address:      genesisMultiSigWallet.Address,
		Owners:       genesisOwners,
		RequiredSigs: requiredSigs,
		AdminWallet:  adminAddress,
		CreatedAt:    time.Now().Unix(),
		Description:  "Genesis multisig wallet with 3 owners and 2/3 threshold",
	}

	multisigData, err := json.MarshalIndent(multisigInfo, "", "  ")
	if err != nil {
		log.Printf("Warning: Failed to marshal multisig info: %v", err)
	} else {
		if err := os.WriteFile("data/multisig.json", multisigData, 0644); err != nil {
			log.Printf("Warning: Failed to save multisig info: %v", err)
		}
	}

	// Step 10: Log Initialization Details
	log.Printf("Genesis block created with multisig wallet address: %s", genesisMultiSigWallet.Address)
	log.Printf("Admin wallet address (symbolic): %s", adminAddress)
	log.Printf("Genesis MultiSig wallet initialized with %d owners", len(genesisOwners))
	log.Printf("Genesis wallet initialized with total supply of %s tokens", totalSupply.String())
	log.Printf("Reward per owner: %s tokens", rewardPerOwner.String())
	log.Printf("Multisig wallet info saved to data/multisig.json")
	log.Printf("Required signatures for multisig operations: %d", requiredSigs)
	log.Printf("Owner addresses and their key pairs:")
	log.Printf("  Owner 1: %s (saved to data/key_%s.json)", owner1KeyPair.GetAddress(), owner1KeyPair.GetAddress())
	log.Printf("  Owner 2: %s (saved to data/key_%s.json)", owner2KeyPair.GetAddress(), owner2KeyPair.GetAddress())
	log.Printf("  Owner 3: %s (saved to data/key_%s.json)", owner3KeyPair.GetAddress(), owner3KeyPair.GetAddress())

	// Step 11: Save Final Blockchain State
	bc.SaveToDisk()
}

// CreateMultiSigWallet creates a new multi-signature wallet
func (bc *Blockchain) CreateMultiSigWallet(address string, owners []string, requiredSigs int) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Check if wallet already exists
	if _, exists := bc.multiSigWallets[address]; exists {
		return errors.New("multi-signature wallet already exists")
	}

	// Create new wallet
	wallet, err := NewMultiSigWallet(address, owners, requiredSigs)
	if err != nil {
		return err
	}

	bc.multiSigWallets[address] = wallet
	return nil
}

// GetMultiSigWallet returns a multisig wallet by its address
func (bc *Blockchain) GetMultiSigWallet(address string) (*MultiSigWallet, error) {
	wallet, exists := bc.multiSigWallets[address]
	if !exists {
		return nil, fmt.Errorf("multisig wallet not found: %s", address)
	}
	return wallet, nil
}

// CreateMultiSigTransaction creates a new multi-signature transaction
func (bc *Blockchain) CreateMultiSigTransaction(walletAddress, from, to string, value string, data []byte, txType string) (*MultiSigTransaction, error) {
	wallet, err := bc.GetMultiSigWallet(walletAddress)
	if err != nil {
		return nil, err
	}

	return wallet.CreateTransaction(from, to, value, data, txType)
}

// SignMultiSigTransaction signs a multi-signature transaction
func (bc *Blockchain) SignMultiSigTransaction(walletAddress, txID, signer string, signature string) error {
	wallet, err := bc.GetMultiSigWallet(walletAddress)
	if err != nil {
		return err
	}

	return wallet.SignTransaction(txID, signer, signature)
}

// ExecuteMultiSigTransaction executes a multi-signature transaction that has enough signatures
func (bc *Blockchain) ExecuteMultiSigTransaction(walletAddress, txID string) error {
	wallet, err := bc.GetMultiSigWallet(walletAddress)
	if err != nil {
		return err
	}

	// Get the transaction
	tx, err := wallet.ExecuteTransaction(txID)
	if err != nil {
		return err
	}

	// Add to pending transactions
	return bc.AddTransaction(tx)
}

// GetMultiSigTransactionStatus returns the status of a multi-signature transaction
func (bc *Blockchain) GetMultiSigTransactionStatus(walletAddress, txID string) (string, error) {
	wallet, err := bc.GetMultiSigWallet(walletAddress)
	if err != nil {
		return "", err
	}

	return wallet.GetTransactionStatus(txID)
}

// GetMultiSigPendingTransactions returns all pending multi-signature transactions for a wallet
func (bc *Blockchain) GetMultiSigPendingTransactions(walletAddress string) ([]*MultiSigTransaction, error) {
	wallet, err := bc.GetMultiSigWallet(walletAddress)
	if err != nil {
		return nil, err
	}

	return wallet.GetPendingTransactions(), nil
}

// RevertTransaction reverts a transaction by its hash
func (bc *Blockchain) RevertTransaction(hash string) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	// Find the transaction in blocks
	for i := len(bc.Blocks) - 1; i >= 0; i-- {
		block := bc.Blocks[i]
		for _, tx := range block.Transactions {
			if tx.ID == hash {
				// Create a reversed transaction
				reversedTx := &Transaction{
					ID:        tx.ID,
					From:      tx.To,    // Swap From and To
					To:        tx.From,  // Swap From and To
					Value:     tx.Value,
					Timestamp: time.Now().Unix(),
				}

				// Update balances using the reversed transaction
				if err := bc.UpdateBalances(reversedTx); err != nil {
					return fmt.Errorf("failed to update balances: %v", err)
				}

				// Save the updated state
				bc.SaveToDisk()
				return nil
			}
		}
	}

	return fmt.Errorf("transaction not found")
}

// VerifySignature verifies a signature using a public key
func (bc *Blockchain) VerifySignature(message, signature string, publicKey *ecdsa.PublicKey) (bool, error) {
	// Decode the signature
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %v", err)
	}

	// Create a hash of the message
	hash := sha256.Sum256([]byte(message))

	// Verify the signature
	return ecdsa.VerifyASN1(publicKey, hash[:], sigBytes), nil
}

// processRewardTransaction creates and processes a reward transaction for a block
func (bc *Blockchain) processRewardTransaction(block *Block) error {
	var errMsgs []string
	
	// Calculate block reward
	rewardUint64 := bc.calculateBlockReward(block)
	
	if rewardUint64 > 0 {
		// Convert reward to big.Int for calculations
		reward := new(big.Int).SetUint64(rewardUint64)
		
		// Update total minted tokens
		bc.TotalMinted = new(big.Int).Add(bc.TotalMinted, reward)
		log.Printf("Adding reward %s to total minted tokens. New total: %s", reward.String(), bc.TotalMinted.String())
		
		// Check if validator is admin wallet or genesis block
		isAdminWallet := false
		for _, admin := range bc.Admins {
			if admin == block.Validator {
				isAdminWallet = true
				break
			}
		}
		
		if isAdminWallet || block.Index == 0 {
			// If validator is admin wallet or genesis block, distribute reward equally among the three signer wallets
			genesisWallet, err := bc.GetMultiSigWallet(block.Validator)
			if err != nil {
				return fmt.Errorf("failed to get genesis wallet: %v", err)
			}
			
			// Calculate reward per owner (total reward divided by 3)
			rewardPerOwner := new(big.Int).Div(reward, big.NewInt(3))
			log.Printf("Distributing reward %s among %d owners. Each owner gets: %s", 
				reward.String(), len(genesisWallet.Owners), rewardPerOwner.String())
			
			// Create reward transactions for each owner
			for _, owner := range genesisWallet.Owners {
				rewardTx := &Transaction{
					ID:        fmt.Sprintf("reward_%d_%s", block.Index, owner),
					From:      "confirmix_genesis_address",
					To:        owner,
					Value:     rewardPerOwner.Uint64(),
					Timestamp: block.Timestamp,
					Type:      "reward",
				}
				
				// Add the reward transaction to the block
				block.Transactions = append(block.Transactions, rewardTx)
				
				// Update balances for the reward transaction
				if err := bc.UpdateBalances(rewardTx); err != nil {
					errMsgs = append(errMsgs, fmt.Sprintf("failed to process reward transaction for %s: %v", owner, err))
				} else {
					// Log successful reward distribution
					balance, _ := bc.GetBalance(owner)
					log.Printf("Owner %s received reward %s. New balance: %s", 
						owner, rewardPerOwner.String(), balance.String())
				}
			}
		} else {
			// If validator is not admin wallet, give full reward to the validator
			rewardTx := &Transaction{
				ID:        fmt.Sprintf("reward_%d", block.Index),
				From:      "confirmix_genesis_address",
				To:        block.Validator,
				Value:     rewardUint64,
				Timestamp: block.Timestamp,
				Type:      "reward",
			}
			
			// Add the reward transaction to the block
			block.Transactions = append(block.Transactions, rewardTx)
			
			// Update balances for the reward transaction
			if err := bc.UpdateBalances(rewardTx); err != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("failed to process reward transaction: %v", err))
			} else {
				// Log successful reward distribution
				balance, _ := bc.GetBalance(block.Validator)
				log.Printf("Validator %s received reward %s. New balance: %s", 
					block.Validator, reward.String(), balance.String())
			}
		}
	}
	
	// Process all user transactions
	for _, tx := range block.Transactions {
		// Skip the reward transaction as it was already processed
		if tx.Type == "reward" {
			continue
		}
		
		// Update balances
		if err := bc.UpdateBalances(tx); err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("failed to process transaction %s: %v", tx.ID, err))
			continue
		}
		
		// Process contract transaction if applicable
		if tx.IsContractTransaction() {
			if err := bc.processContractTransaction(tx); err != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("failed to process contract transaction %s: %v", tx.ID, err))
			}
		}
	}
	
	// Clean transaction pool
	bc.cleanTransactionPool(block.Transactions)
	
	// Save blockchain state
	if err := bc.SaveToDisk(); err != nil {
		errMsgs = append(errMsgs, fmt.Sprintf("failed to save blockchain state: %v", err))
	}
	
	if len(errMsgs) > 0 {
		return fmt.Errorf("block added with errors: %s", strings.Join(errMsgs, "; "))
	}
	
	return nil
}

// calculateBlockReward calculates the reward for a block
func (bc *Blockchain) calculateBlockReward(block *Block) uint64 {
	// Base reward
	baseReward := uint64(50 * 1000000) // 50 coins

	// Halving every 210,000 blocks (similar to Bitcoin)
	halvings := block.Index / 210000
	if halvings > 0 {
		for i := uint64(0); i < halvings; i++ {
			baseReward /= 2
		}
	}

	return baseReward
}
