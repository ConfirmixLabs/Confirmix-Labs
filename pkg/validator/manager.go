package validator

import (
	"fmt"
	"sync"
	"time"
)

// Manager manages validators and their voting powers
type Manager struct {
	validators map[string]*ValidatorRuntime
	mutex      sync.RWMutex
}

// NewManager creates a new validator manager
func NewManager() *Manager {
	return &Manager{
		validators: make(map[string]*ValidatorRuntime),
	}
}

// CreateValidator creates a new validator
func (m *Manager) CreateValidator(name string, power int64) *Validator {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	validator := NewValidator(name, power, 1*time.Second, "http://localhost:8080", m)
	m.validators[validator.Address] = validator
	return validator.Validator
}

// GetValidator returns a validator by address
func (m *Manager) GetValidator(address string) *Validator {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if validator := m.validators[address]; validator != nil {
		return validator.Validator
	}
	return nil
}

// UpdateValidator updates a validator using the provided update function
func (m *Manager) UpdateValidator(address string, updateFn func(*Validator)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if validator, exists := m.validators[address]; exists {
		updateFn(validator.Validator)
	}
}

// RemoveValidator removes a validator
func (m *Manager) RemoveValidator(address string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.validators, address)
}

// GetValidatorPower returns the voting power of a validator
func (m *Manager) GetValidatorPower(address string) (int64, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if validator, exists := m.validators[address]; exists {
		return validator.VotingPower, nil
	}
	return 0, fmt.Errorf("validator not found")
}

// IsValidator checks if an address is a validator
func (m *Manager) IsValidator(address string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	_, exists := m.validators[address]
	return exists
}

// GetTotalPower returns the total voting power of all validators
func (m *Manager) GetTotalPower() int64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var total int64
	for _, validator := range m.validators {
		total += validator.VotingPower
	}
	return total
}

// GetValidators returns all validators
func (m *Manager) GetValidators() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	validators := make([]string, 0, len(m.validators))
	for address := range m.validators {
		validators = append(validators, address)
	}
	return validators
} 