package governance

import (
	"math/big"
	"time"

	"confirmix/pkg/validator"
)

// BlockchainReader defines methods for reading blockchain state
type BlockchainReader interface {
	GetBlockHeight() int64
	GetBlockHash(height int64) string
}

// ValidatorManager defines the interface for validator management
type ValidatorManager interface {
	// CreateValidator creates a new validator
	CreateValidator(name string, power int64) *validator.Validator

	// GetValidator returns a validator by address
	GetValidator(address string) *validator.Validator

	// UpdateValidator updates a validator using the provided update function
	UpdateValidator(address string, updateFn func(*validator.Validator))

	// RemoveValidator removes a validator
	RemoveValidator(address string)

	// IsValidator checks if an address is a validator
	IsValidator(address string) bool

	// GetValidators returns all validators
	GetValidators() []string

	// GetTotalPower returns the total voting power
	GetTotalPower() int64

	// GetValidatorPower returns the voting power of a validator
	GetValidatorPower(address string) (int64, error)
}

// TokenSystem defines methods for managing tokens
type TokenSystem interface {
	GetBalance(address string) (*big.Int, error)
	SetBalance(address string, amount *big.Int) error
	Transfer(from, to string, amount *big.Int) error
	Lock(address string, amount *big.Int) error
	Unlock(address string, amount *big.Int) error
	GetTotalSupply() (*big.Int, error)
}

// ProposalStore defines methods for storing proposals
type ProposalStore interface {
	SaveProposal(proposal *Proposal) error
	GetProposal(id string) (*Proposal, error)
	ListProposals() ([]*Proposal, error)
	ListProposalsByStatus(status ProposalStatus) ([]*Proposal, error)
	UpdateProposalStatus(id string, status ProposalStatus) error
	AddVote(id string, vote *Vote) error
}

// ProposalExecutor defines methods for executing proposals
type ProposalExecutor interface {
	Execute(proposal *Proposal) error
	Validate(proposal *Proposal) error
	GetRequiredDeposit(proposalType string) (*big.Int, error)
	GetExecutionDelay(proposalType string) (time.Duration, error)
}

// ProposalValidator defines methods for validating proposals and votes
type ProposalValidator interface {
	ValidateProposal(proposal *Proposal) error
	ValidateVote(proposal *Proposal, vote *Vote) error
	GetVotingPower(address string) (int64, error)
	IsValidator(address string) bool
} 