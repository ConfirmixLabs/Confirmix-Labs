package validator

import (
	"testing"
	"time"
)

func TestValidatorCreation(t *testing.T) {
	manager := NewValidatorManager()

	// Test creating a new validator
	validator := manager.CreateValidator("test-validator", 100)
	if validator == nil {
		t.Fatal("Expected validator to be created, got nil")
	}

	if validator.Name != "test-validator" {
		t.Errorf("Expected validator name to be 'test-validator', got '%s'", validator.Name)
	}

	if validator.VotingPower != 100 {
		t.Errorf("Expected voting power to be 100, got %d", validator.VotingPower)
	}

	if validator.Status != Active {
		t.Errorf("Expected validator status to be Active, got %v", validator.Status)
	}
}

func TestValidatorManagement(t *testing.T) {
	manager := NewValidatorManager()

	// Create a validator
	validator := manager.CreateValidator("test-validator", 100)

	// Test getting validator
	retrievedValidator := manager.GetValidator(validator.Address)
	if retrievedValidator == nil {
		t.Fatal("Expected to retrieve validator, got nil")
	}

	if retrievedValidator.Address != validator.Address {
		t.Errorf("Expected validator address to match, got different addresses")
	}

	// Test updating validator
	manager.UpdateValidator(validator.Address, func(v *Validator) {
		v.VotingPower = 200
	})

	updatedValidator := manager.GetValidator(validator.Address)
	if updatedValidator.VotingPower != 200 {
		t.Errorf("Expected voting power to be updated to 200, got %d", updatedValidator.VotingPower)
	}

	// Test removing validator
	manager.RemoveValidator(validator.Address)
	removedValidator := manager.GetValidator(validator.Address)
	if removedValidator != nil {
		t.Error("Expected validator to be removed, but it still exists")
	}
}

func TestValidatorStatus(t *testing.T) {
	manager := NewValidatorManager()
	validator := manager.CreateValidator("test-validator", 100)

	// Test setting status to inactive
	manager.UpdateValidator(validator.Address, func(v *Validator) {
		v.Status = Inactive
	})

	updatedValidator := manager.GetValidator(validator.Address)
	if updatedValidator.Status != Inactive {
		t.Errorf("Expected validator status to be Inactive, got %v", updatedValidator.Status)
	}

	// Test setting status to jailed
	manager.UpdateValidator(validator.Address, func(v *Validator) {
		v.Status = Jailed
		v.JailedUntil = time.Now().Add(24 * time.Hour)
	})

	updatedValidator = manager.GetValidator(validator.Address)
	if updatedValidator.Status != Jailed {
		t.Errorf("Expected validator status to be Jailed, got %v", updatedValidator.Status)
	}
} 