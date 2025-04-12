package validator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

// ValidatorStatus represents the status of a validator
type ValidatorStatus int

const (
	Active ValidatorStatus = iota
	Inactive
	Jailed
)

// Block represents a blockchain block
type Block struct {
	Index        int64         `json:"Index"`
	Hash         string        `json:"Hash"`
	PrevHash     string        `json:"PrevHash"`
	Timestamp    int64         `json:"Timestamp"`
	Transactions []Transaction `json:"Transactions"`
	Validator    string        `json:"Validator"`
	HumanProof   string        `json:"HumanProof"`
	Signature    string        `json:"Signature,omitempty"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID        string  `json:"ID"`
	From      string  `json:"From"`
	To        string  `json:"To"`
	Value     float64 `json:"Value"`
	Data      string  `json:"Data,omitempty"`
	Timestamp int64   `json:"Timestamp"`
	Signature string  `json:"Signature,omitempty"`
}

// Validator represents a validator in the system
type Validator struct {
	Name        string
	Address     string
	VotingPower int64
	Status      ValidatorStatus
	JailedUntil time.Time
}

// ValidatorRuntime represents a validator with runtime fields
type ValidatorRuntime struct {
	*Validator
	timeout    time.Duration
	isRunning  bool
	stopChan   chan struct{}
	wg         sync.WaitGroup
	interval   time.Duration
	client     *http.Client
	apiBaseURL string
	manager    *Manager
}

// NewValidator creates a new validator
func NewValidator(name string, power int64, timeout time.Duration, apiBaseURL string, manager *Manager) *ValidatorRuntime {
	return &ValidatorRuntime{
		Validator: &Validator{
			Name:        name,
			Address:     name, // Using name as address for simplicity
			VotingPower: power,
			Status:      Active,
		},
		timeout:    timeout,
		isRunning:  false,
		stopChan:   make(chan struct{}),
		interval:   1 * time.Second,
		client:     &http.Client{Timeout: 10 * time.Second},
		apiBaseURL: apiBaseURL,
		manager:    manager,
	}
}

// Start begins the validation process
func (v *ValidatorRuntime) Start() error {
	if v.isRunning {
		return fmt.Errorf("validator is already running")
	}

	log.Printf("Starting validator with address: %s", v.Address)
	log.Printf("Connected to API at: %s", v.apiBaseURL)
	v.isRunning = true
	v.wg.Add(1)

	go v.validationLoop()

	return nil
}

// Stop halts the validation process
func (v *ValidatorRuntime) Stop() {
	if !v.isRunning {
		return
	}

	log.Println("Stopping validator...")
	close(v.stopChan)
	v.wg.Wait()
	v.isRunning = false
	log.Println("Validator stopped")
}

// validationLoop runs continuously to validate pending transactions
func (v *ValidatorRuntime) validationLoop() {
	defer v.wg.Done()

	ticker := time.NewTicker(v.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			v.validatePendingTransactions()
		case <-v.stopChan:
			return
		}
	}
}

// validatePendingTransactions processes pending transactions and creates a new block
func (v *ValidatorRuntime) validatePendingTransactions() {
	// Get pending transactions
	pendingTxs, err := v.getPendingTransactions()
	if err != nil {
		log.Printf("Error fetching pending transactions: %v", err)
		return
	}

	if len(pendingTxs) == 0 {
		return // Nothing to validate
	}

	log.Printf("Found %d pending transactions to validate", len(pendingTxs))
	
	// İşlemlerin Transaction.ID değerlerini saklayalım
	txIDs := make([]string, len(pendingTxs))
	for i, tx := range pendingTxs {
		txIDs[i] = tx.ID
	}

	// Mine a new block
	err = v.mineBlock()
	if err != nil {
		log.Printf("Error mining block: %v", err)
		return
	}

	log.Printf("Successfully validated transactions and created a new block")
}

// getPendingTransactions retrieves pending transactions from the API
func (v *ValidatorRuntime) getPendingTransactions() ([]Transaction, error) {
	resp, err := v.client.Get(fmt.Sprintf("%s/transactions/pending", v.apiBaseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending transactions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var transactions []Transaction
	if err := json.NewDecoder(resp.Body).Decode(&transactions); err != nil {
		return nil, fmt.Errorf("failed to decode transactions response: %w", err)
	}

	return transactions, nil
}

// mineBlock requests the API to mine a new block
func (v *ValidatorRuntime) mineBlock() error {
	// Create request body
	reqBody, err := json.Marshal(map[string]string{
		"validator": v.Address,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Send request to mine endpoint
	resp, err := v.client.Post(
		fmt.Sprintf("%s/mine", v.apiBaseURL),
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return fmt.Errorf("failed to send mine request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Try to read error message
		var errMsg struct {
			Error string `json:"error"`
		}
		
		respBody, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Mining response body: %s", string(respBody))
		
		if err := json.Unmarshal(respBody, &errMsg); err == nil && errMsg.Error != "" {
			return fmt.Errorf("mining failed: %s", errMsg.Error)
		}
		return fmt.Errorf("mining failed with status: %d", resp.StatusCode)
	}

	return nil
}

// ValidateProposal validates a proposal
func (v *ValidatorRuntime) ValidateProposal(proposal *Proposal) error {
	if proposal == nil {
		return errors.New("proposal is nil")
	}

	if proposal.Type == "" {
		return errors.New("proposal type is empty")
	}

	if proposal.Title == "" {
		return errors.New("proposal title is empty")
	}

	if proposal.Description == "" {
		return errors.New("proposal description is empty")
	}

	return nil
}

// ValidateVote validates a vote
func (v *ValidatorRuntime) ValidateVote(proposal *Proposal, vote *Vote) error {
	if proposal == nil {
		return errors.New("proposal is nil")
	}

	if vote == nil {
		return errors.New("vote is nil")
	}

	if vote.Voter == "" {
		return errors.New("voter address is empty")
	}

	if vote.Proposal == "" {
		return errors.New("proposal ID is empty")
	}

	return nil
}

// GetVotingPower returns the voting power of a validator
func (v *ValidatorRuntime) GetVotingPower(address string) (int64, error) {
	if address == v.Address {
		return v.VotingPower, nil
	}
	
	// Check if address is in the manager
	power, err := v.manager.GetValidatorPower(address)
	if err == nil {
		return power, nil
	}
	
	return 0, fmt.Errorf("address %s is not a validator", address)
}

// IsValidator checks if an address is a validator
func (v *ValidatorRuntime) IsValidator(address string) bool {
	if address == v.Address {
		return true
	}
	return v.manager.IsValidator(address)
} 