package blockchain

import (
	"math/big"
)

// TokenSystemAdapter adapts the Blockchain to implement the TokenSystem interface
// required by the Governance system
type TokenSystemAdapter struct {
	Blockchain *Blockchain
}

// GetBalance returns the balance of an address
func (ta *TokenSystemAdapter) GetBalance(address string) (*big.Int, error) {
	return ta.Blockchain.GetBalance(address)
}

// TransferFrom transfers tokens from one address to another
func (ta *TokenSystemAdapter) TransferFrom(from, to string, amount *big.Int) error {
	return ta.Blockchain.TransferFrom(from, to, amount)
}

// Lock locks tokens for governance or staking
func (ta *TokenSystemAdapter) Lock(address string, amount *big.Int) error {
	return ta.Blockchain.Lock(address, amount)
}

// Unlock unlocks tokens that were previously locked
func (ta *TokenSystemAdapter) Unlock(address string, amount *big.Int) error {
	return ta.Blockchain.Unlock(address, amount)
} 