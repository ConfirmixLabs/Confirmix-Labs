package governance

import (
	"math/big"
	"time"
)

// ProposalStatus represents the status of a proposal
type ProposalStatus int

const (
	ProposalStatusVoting ProposalStatus = iota
	ProposalStatusPassed
	ProposalStatusRejected
	ProposalStatusExecuted
)

// ProposalType represents the type of a proposal
type ProposalType string

const (
	ProposalTypeAddValidator    ProposalType = "add_validator"
	ProposalTypeRemoveValidator ProposalType = "remove_validator"
	ProposalTypeChangeParam     ProposalType = "change_param"
	ProposalTypeText           ProposalType = "text"
)

// Vote represents a vote on a proposal
type Vote struct {
	Voter    string
	Proposal string
	Approve  bool
	Time     time.Time
}

// Proposal represents a governance proposal
type Proposal struct {
	ID          string
	Type        ProposalType
	Title       string
	Description string
	Status      ProposalStatus
	CreatedAt   time.Time
	Votes       map[string]bool
	Data        map[string]string
}

// Config represents the governance configuration
type Config struct {
	MinProposalDeposit    *big.Int `json:"min_proposal_deposit"`
	VotingPeriod         time.Duration `json:"voting_period"`
	QuorumPercentage     int64    `json:"quorum_percentage"`
	ApprovalThreshold    int64    `json:"approval_threshold"`
	ExecutionDelay       time.Duration `json:"execution_delay"`
	DefaultGovernance    bool     `json:"default_governance"`
	AdminOverride        bool     `json:"admin_override"`
}

// DefaultConfig returns the default governance configuration
func DefaultConfig() *Config {
	return &Config{
		MinProposalDeposit:    big.NewInt(1000),
		VotingPeriod:         7 * 24 * time.Hour, // 1 week
		QuorumPercentage:     30,
		ApprovalThreshold:    51,
		ExecutionDelay:       24 * time.Hour, // 1 day
		DefaultGovernance:    false,
		AdminOverride:        true,
	}
} 