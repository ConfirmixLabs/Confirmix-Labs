package validator

import (
	"fmt"
	"math/big"
	"time"

	"confirmix/pkg/governance"
)

// DefaultValidator implements ProposalValidator interface
type DefaultValidator struct {
	blockchain governance.BlockchainReader
	validators governance.ValidatorManager
	tokens    governance.TokenSystem
}

// NewDefaultValidator creates a new default proposal validator
func NewDefaultValidator(
	bc governance.BlockchainReader,
	vm governance.ValidatorManager,
	ts governance.TokenSystem,
) *DefaultValidator {
	return &DefaultValidator{
		blockchain: bc,
		validators: vm,
		tokens:    ts,
	}
}

// ValidateProposal validates a proposal
func (v *DefaultValidator) ValidateProposal(proposal *governance.Proposal) error {
	// Basic validation
	if proposal.Title == "" {
		return fmt.Errorf("proposal title is required")
	}

	if proposal.Description == "" {
		return fmt.Errorf("proposal description is required")
	}

	if proposal.Creator == "" {
		return fmt.Errorf("proposal creator is required")
	}

	if proposal.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("proposal expiration time must be in the future")
	}

	// Type-specific validation
	switch proposal.Type {
	case governance.ProposalTypeAddValidator:
		return v.validateAddValidatorProposal(proposal)
	case governance.ProposalTypeRemoveValidator:
		return v.validateRemoveValidatorProposal(proposal)
	case governance.ProposalTypeChangeParameter:
		return v.validateChangeParameterProposal(proposal)
	case governance.ProposalTypeUpgradeSoftware:
		return v.validateUpgradeProposal(proposal)
	case governance.ProposalTypeTransferFunds:
		return v.validateTransferProposal(proposal)
	default:
		return fmt.Errorf("unknown proposal type: %s", proposal.Type)
	}
}

// ValidateVote validates a vote on a proposal
func (v *DefaultValidator) ValidateVote(proposal *governance.Proposal, vote *governance.Vote) error {
	// Basic validation
	if vote.Voter == "" {
		return fmt.Errorf("voter address is required")
	}

	// Get voting power
	var votingPower *big.Int
	if v.validators.IsValidator(vote.Voter) {
		// If validator, get token balance and multiply by 2
		balance, err := v.tokens.GetBalance(vote.Voter)
		if err != nil {
			return fmt.Errorf("failed to get voter balance: %v", err)
		}
		votingPower = new(big.Int).Mul(balance, big.NewInt(2))
	} else {
		// If not a validator, just get token balance
		balance, err := v.tokens.GetBalance(vote.Voter)
		if err != nil {
			return fmt.Errorf("failed to get voter balance: %v", err)
		}
		if balance.Cmp(big.NewInt(0)) <= 0 {
			return fmt.Errorf("voter has no voting power")
		}
		votingPower = balance
	}

	vote.VotingPower = votingPower
	return nil
}

// validateAddValidatorProposal validates add validator proposal
func (v *DefaultValidator) validateAddValidatorProposal(proposal *governance.Proposal) error {
	address, ok := proposal.Data["address"]
	if !ok {
		return fmt.Errorf("validator address is required")
	}

	if v.validators.IsValidator(address) {
		return fmt.Errorf("address %s is already a validator", address)
	}

	return nil
}

// validateRemoveValidatorProposal validates remove validator proposal
func (v *DefaultValidator) validateRemoveValidatorProposal(proposal *governance.Proposal) error {
	address, ok := proposal.Data["address"]
	if !ok {
		return fmt.Errorf("validator address is required")
	}

	if !v.validators.IsValidator(address) {
		return fmt.Errorf("address %s is not a validator", address)
	}

	return nil
}

// validateChangeParameterProposal validates parameter change proposal
func (v *DefaultValidator) validateChangeParameterProposal(proposal *governance.Proposal) error {
	paramName, ok := proposal.Data["parameter"]
	if !ok {
		return fmt.Errorf("parameter name is required")
	}

	_, ok = proposal.Data["value"]
	if !ok {
		return fmt.Errorf("parameter value is required")
	}

	// Validate specific parameters
	switch paramName {
	case "voting_period":
		// Validate voting period
		return nil
	case "execution_delay":
		// Validate execution delay
		return nil
	case "quorum_percentage":
		// Validate quorum percentage
		return nil
	case "approval_threshold":
		// Validate approval threshold
		return nil
	default:
		return fmt.Errorf("unknown parameter: %s", paramName)
	}
}

// validateUpgradeProposal validates software upgrade proposal
func (v *DefaultValidator) validateUpgradeProposal(proposal *governance.Proposal) error {
	_, ok := proposal.Data["version"]
	if !ok {
		return fmt.Errorf("upgrade version is required")
	}

	_, ok = proposal.Data["height"]
	if !ok {
		return fmt.Errorf("upgrade height is required")
	}

	// Additional upgrade-specific validation
	return nil
}

// validateTransferProposal validates fund transfer proposal
func (v *DefaultValidator) validateTransferProposal(proposal *governance.Proposal) error {
	_, ok := proposal.Data["to"]
	if !ok {
		return fmt.Errorf("recipient address is required")
	}

	_, ok = proposal.Data["amount"]
	if !ok {
		return fmt.Errorf("transfer amount is required")
	}

	// Additional transfer-specific validation
	return nil
} 