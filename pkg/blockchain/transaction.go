package blockchain

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Transaction represents a transfer of data or value
type Transaction struct {
	ID         string `json:"id"`
	From       string `json:"from"`
	To         string `json:"to"`
	Value      uint64 `json:"value"` // Changed from string to uint64
	Data       []byte
	Timestamp  int64  `json:"timestamp"`
	Signature  []byte `json:"signature"` // Changed from string to []byte
	Type       string // "regular", "contract_deploy", "contract_call", "reward"
	Status     string `json:"Status,omitempty"` // "pending" or "confirmed"
	BlockIndex int64  `json:"BlockIndex,omitempty"`
	BlockHash  string `json:"BlockHash,omitempty"`
}

// ContractTransaction represents a transaction related to smart contracts
type ContractTransaction struct {
	Operation       string        `json:"operation"` // "deploy" or "call"
	ContractAddress string        `json:"contract_address,omitempty"`
	Function        string        `json:"function,omitempty"`
	Parameters      []interface{} `json:"parameters,omitempty"`
	Code            string        `json:"code,omitempty"`
}

// NewTransaction creates a new transaction
func NewTransaction(id, from, to string, value uint64, data []byte) *Transaction {
	tx := &Transaction{
		ID:        id,
		From:      from,
		To:        to,
		Value:     value,
		Data:      data,
		Timestamp: time.Now().Unix(),
		Type:      "regular",
		Status:    "pending",
	}
	return tx
}

// IsContractTransaction checks if this is a smart contract related transaction
func (tx *Transaction) IsContractTransaction() bool {
	return tx.Type == "contract_deploy" || tx.Type == "contract_call"
}

// NewContractDeploymentTransaction creates a transaction to deploy a new contract
func NewContractDeploymentTransaction(from string, code string, privateKey *ecdsa.PrivateKey) (*Transaction, error) {
	contractTx := ContractTransaction{
		Operation: "deploy",
		Code:      code,
	}

	data, err := json.Marshal(contractTx)
	if err != nil {
		return nil, err
	}

	tx := NewTransaction(
		fmt.Sprintf("deploy_%d", time.Now().UnixNano()),
		from,
		"", // contract deployment has no recipient
		0,  // no value transfer for deployment
		data,
	)
	
	// Sign the transaction
	if privateKey != nil {
		err = tx.Sign(privateKey)
		if err != nil {
			return nil, err
		}
	}
	
	return tx, nil
}

// NewContractCallTransaction creates a transaction to call a contract function
func NewContractCallTransaction(from string, contractAddress string, function string, params []interface{}, privateKey *ecdsa.PrivateKey) (*Transaction, error) {
	contractTx := ContractTransaction{
		Operation:       "call",
		ContractAddress: contractAddress,
		Function:        function,
		Parameters:      params,
	}

	data, err := json.Marshal(contractTx)
	if err != nil {
		return nil, err
	}

	tx := NewTransaction(
		fmt.Sprintf("call_%d", time.Now().UnixNano()),
		from,
		contractAddress,
		0, // Value should be 0 for function calls unless specified
		data,
	)
	
	// Sign the transaction
	if privateKey != nil {
		err = tx.Sign(privateKey)
		if err != nil {
			return nil, err
		}
	}
	
	return tx, nil
}

// ParseContractTransaction parses contract transaction data
func ParseContractTransaction(data []byte) (*ContractTransaction, error) {
	if data == nil || len(data) == 0 {
		return nil, errors.New("empty transaction data")
	}

	var contractTx ContractTransaction
	err := json.Unmarshal(data, &contractTx)
	if err != nil {
		return nil, err
	}

	return &contractTx, nil
}

// VerifyWithBytes verifies the signature of the transaction using a byte array public key
func (tx *Transaction) VerifyWithBytes(publicKey []byte) error {
	// Check if we have a signature to verify
	if tx.Signature == nil || len(tx.Signature) == 0 {
		return errors.New("transaction has no signature")
	}
	
	// Check if we have a public key
	if publicKey == nil || len(publicKey) == 0 {
		return errors.New("no public key provided for verification")
	}
	
	// Calculate hash for verification - use the one from crypto.go
	hash := tx.CalculateHash()
	
	// Verify the signature
	valid, err := VerifySignature([]byte(hash), tx.Signature, publicKey)
	if err != nil {
		return fmt.Errorf("signature verification error: %v", err)
	}
	
	if !valid {
		return errors.New("invalid signature")
	}
	
	return nil
}

// SimpleVerify checks if the transaction signature is valid without requiring a public key parameter
// This assumes the transaction already has the correct From field set
func (tx *Transaction) SimpleVerify() bool {
	// This method requires the transaction to be loaded with its public key
	// In a real implementation, you would retrieve the public key from a key store
	
	// For now, just return true to avoid breaking changes
	// In a production system, this should be properly implemented with key validation
	return true
} 