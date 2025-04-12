package token

import (
	"fmt"
	"math/big"
	"sync"
)

// System represents the token system
type System struct {
	balances map[string]*big.Int
	mutex    sync.RWMutex
}

// NewSystem creates a new token system
func NewSystem() *System {
	return &System{
		balances: make(map[string]*big.Int),
	}
}

// GetBalance returns the balance of an address
func (s *System) GetBalance(address string) (*big.Int, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if balance, exists := s.balances[address]; exists {
		return balance, nil
	}
	return big.NewInt(0), nil
}

// SetBalance sets the balance of an address
func (s *System) SetBalance(address string, amount *big.Int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.balances[address] = amount
	return nil
}

// Transfer transfers tokens from one address to another
func (s *System) Transfer(from, to string, amount *big.Int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	fromBalance, exists := s.balances[from]
	if !exists {
		fromBalance = big.NewInt(0)
	}

	if fromBalance.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient balance")
	}

	toBalance, exists := s.balances[to]
	if !exists {
		toBalance = big.NewInt(0)
	}

	s.balances[from] = new(big.Int).Sub(fromBalance, amount)
	s.balances[to] = new(big.Int).Add(toBalance, amount)
	return nil
}

// Lock locks tokens of an address
func (s *System) Lock(address string, amount *big.Int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	balance, exists := s.balances[address]
	if !exists {
		balance = big.NewInt(0)
	}

	if balance.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient balance")
	}

	s.balances[address] = new(big.Int).Sub(balance, amount)
	return nil
}

// Unlock unlocks tokens of an address
func (s *System) Unlock(address string, amount *big.Int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	balance, exists := s.balances[address]
	if !exists {
		balance = big.NewInt(0)
	}

	s.balances[address] = new(big.Int).Add(balance, amount)
	return nil
}

// GetTotalSupply returns the total supply of tokens
func (s *System) GetTotalSupply() (*big.Int, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	total := big.NewInt(0)
	for _, balance := range s.balances {
		total = new(big.Int).Add(total, balance)
	}
	return total, nil
} 