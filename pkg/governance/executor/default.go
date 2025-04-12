package executor

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"confirmix/pkg/governance"
	"confirmix/pkg/validator"
)

// Executor implements the ProposalExecutor interface
type Executor struct {
	validators *validator.Manager
}

// NewExecutor creates a new executor
func NewExecutor(validators *validator.Manager) *Executor {
	return &Executor{
		validators: validators,
	}
}

// Execute executes a proposal
func (e *Executor) Execute(proposal *governance.Proposal) error {
	if proposal == nil {
		return errors.New("proposal is nil")
	}

	switch proposal.Type {
	case governance.ProposalTypeAddValidator:
		return e.executeAddValidator(proposal)
	case governance.ProposalTypeRemoveValidator:
		return e.executeRemoveValidator(proposal)
	case governance.ProposalTypeChangeParam:
		return e.executeChangeParam(proposal)
	// case governance.ProposalTypeUpgradeSoftware:
	// 	return e.executeUpgradeSoftware(proposal)
	// case governance.ProposalTypeTransferFunds:
	// 	return e.executeTransferFunds(proposal)
	default:
		return fmt.Errorf("unknown proposal type: %s", proposal.Type)
	}
}

// Validate validates a proposal
func (e *Executor) Validate(proposal *governance.Proposal) error {
	if proposal == nil {
		return errors.New("proposal is nil")
	}

	if proposal.Title == "" {
		return errors.New("proposal title is empty")
	}

	if proposal.Description == "" {
		return errors.New("proposal description is empty")
	}

	return nil
}

// GetRequiredDeposit returns the required deposit for a proposal type
func (e *Executor) GetRequiredDeposit(proposalType string) (*big.Int, error) {
	return big.NewInt(1000), nil // Fixed deposit for now
}

// GetExecutionDelay returns the execution delay for a proposal type
func (e *Executor) GetExecutionDelay(proposalType string) (time.Duration, error) {
	return 24 * time.Hour, nil // Fixed delay for now
}

// executeAddValidator executes an add validator proposal
func (e *Executor) executeAddValidator(proposal *governance.Proposal) error {
	address, ok := proposal.Data["address"]
	if !ok {
		return errors.New("validator address is required")
	}

	power, ok := proposal.Data["power"]
	if !ok {
		return errors.New("validator power is required")
	}

	powerInt, err := strconv.ParseInt(power, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid power value: %v", err)
	}

	validator := e.validators.CreateValidator(address, powerInt)
	if validator == nil {
		return fmt.Errorf("failed to create validator")
	}
	return nil
}

// executeRemoveValidator executes a remove validator proposal
func (e *Executor) executeRemoveValidator(proposal *governance.Proposal) error {
	address, ok := proposal.Data["address"]
	if !ok {
		return errors.New("validator address is required")
	}

	e.validators.RemoveValidator(address)
	return nil
}

// executeChangeParam executes a change parameters proposal
func (e *Executor) executeChangeParam(proposal *governance.Proposal) error {
	// Implement parameter change logic here
	return nil
}

// // executeUpgradeSoftware executes an upgrade software proposal
// func (e *Executor) executeUpgradeSoftware(proposal *governance.Proposal) error {
// 	// Implement software upgrade logic here
// 	return nil
// }

// // executeTransferFunds executes a transfer funds proposal
// func (e *Executor) executeTransferFunds(proposal *governance.Proposal) error {
// 	// Implement fund transfer logic here
// 	return nil
// } 