package blockchain

import (
	"fmt"
	"math/big"
	"sync"
	"time"
)

// MultiSigWallet represents a multi-signature wallet
type MultiSigWallet struct {
	Address         string
	Owners          []string
	RequiredSigs    int
	PendingTxs      map[string]*MultiSigTransaction
	mutex           sync.RWMutex
}

// MultiSigTransaction represents a transaction that requires multiple signatures
type MultiSigTransaction struct {
	ID          string
	From        string
	To          string
	Value       *big.Int
	Data        []byte
	Type        string
	Signatures  map[string]string
	Status      string
	CreatedAt   int64
}

// NewMultiSigWallet creates a new multi-signature wallet
func NewMultiSigWallet(address string, owners []string, requiredSigs int) (*MultiSigWallet, error) {
	if len(owners) < requiredSigs {
		return nil, fmt.Errorf("number of owners (%d) must be greater than or equal to required signatures (%d)", 
			len(owners), requiredSigs)
	}

	return &MultiSigWallet{
		Address:      address,
		Owners:       owners,
		RequiredSigs: requiredSigs,
		PendingTxs:   make(map[string]*MultiSigTransaction),
	}, nil
}

// CreateTransaction creates a new multi-signature transaction
func (w *MultiSigWallet) CreateTransaction(from, to string, value string, data []byte, txType string) (*MultiSigTransaction, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Verify sender is an owner
	isOwner := false
	for _, owner := range w.Owners {
		if owner == from {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return nil, fmt.Errorf("sender %s is not an owner of this wallet", from)
	}

	// Convert value string to big.Int
	valueBig := new(big.Int)
	if _, success := valueBig.SetString(value, 10); !success {
		return nil, fmt.Errorf("invalid value format: %s", value)
	}

	tx := &MultiSigTransaction{
		ID:         fmt.Sprintf("multisig_%d", time.Now().UnixNano()),
		From:       from,
		To:         to,
		Value:      valueBig,
		Data:       data,
		Type:       txType,
		Signatures: make(map[string]string),
		Status:     "pending",
		CreatedAt:  time.Now().Unix(),
	}

	w.PendingTxs[tx.ID] = tx
	return tx, nil
}

// SignTransaction adds a signature to a pending transaction
func (w *MultiSigWallet) SignTransaction(txID string, signer string, signature string) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Verify signer is an owner
	isOwner := false
	for _, owner := range w.Owners {
		if owner == signer {
			isOwner = true
			break
		}
	}
	if !isOwner {
		return fmt.Errorf("signer %s is not an owner of this wallet", signer)
	}

	tx, exists := w.PendingTxs[txID]
	if !exists {
		return fmt.Errorf("transaction %s not found", txID)
	}

	// Check if already signed by this owner
	if _, exists := tx.Signatures[signer]; exists {
		return fmt.Errorf("transaction already signed by %s", signer)
	}

	tx.Signatures[signer] = signature
	return nil
}

// GetTransactionStatus returns the current status of a transaction
func (w *MultiSigWallet) GetTransactionStatus(txID string) (string, error) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	tx, exists := w.PendingTxs[txID]
	if !exists {
		return "", fmt.Errorf("transaction %s not found", txID)
	}

	return tx.Status, nil
}

// GetPendingTransactions returns all pending transactions
func (w *MultiSigWallet) GetPendingTransactions() []*MultiSigTransaction {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	txs := make([]*MultiSigTransaction, 0, len(w.PendingTxs))
	for _, tx := range w.PendingTxs {
		txs = append(txs, tx)
	}
	return txs
}

// ExecuteTransaction executes a transaction that has enough signatures
func (w *MultiSigWallet) ExecuteTransaction(txID string) (*Transaction, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	tx, exists := w.PendingTxs[txID]
	if !exists {
		return nil, fmt.Errorf("transaction %s not found", txID)
	}

	// Check if we have enough signatures
	if len(tx.Signatures) < w.RequiredSigs {
		return nil, fmt.Errorf("not enough signatures: got %d, need %d", 
			len(tx.Signatures), w.RequiredSigs)
	}

	// Create a regular transaction
	regularTx := &Transaction{
		ID:        tx.ID,
		From:      tx.From,
		To:        tx.To,
		Value:     tx.Value.Uint64(),
		Data:      tx.Data,
		Type:      tx.Type,
		Timestamp: tx.CreatedAt,
		Status:    "pending",
	}

	// Remove from pending transactions
	delete(w.PendingTxs, txID)
	return regularTx, nil
}

// RejectTransaction rejects a pending transaction
func (w *MultiSigWallet) RejectTransaction(txID string) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if _, exists := w.PendingTxs[txID]; !exists {
		return fmt.Errorf("transaction %s not found", txID)
	}

	delete(w.PendingTxs, txID)
	return nil
}

// GetOwners returns the list of wallet owners
func (w *MultiSigWallet) GetOwners() []string {
	return w.Owners
}

// GetRequiredSignatures returns the number of required signatures
func (w *MultiSigWallet) GetRequiredSignatures() int {
	return w.RequiredSigs
} 