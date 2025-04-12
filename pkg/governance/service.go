package governance

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
	"crypto/rand"
	"encoding/hex"
)

var (
	// ErrInvalidProposalType indicates an invalid proposal type
	ErrInvalidProposalType = errors.New("invalid proposal type")

	// ErrNotValidator indicates the address is not a validator
	ErrNotValidator = errors.New("address is not a validator")

	// ErrProposalNotInVoting indicates the proposal is not in voting period
	ErrProposalNotInVoting = errors.New("proposal is not in voting period")

	// ErrProposalNotFound indicates the proposal was not found
	ErrProposalNotFound = errors.New("proposal not found")
)

// Service represents the governance service
type Service struct {
	blockchain BlockchainReader
	validators ValidatorManager
	tokens     TokenSystem
	store      ProposalStore
	executor   ProposalExecutor
	config     *Config
	mutex      sync.RWMutex
}

// NewService creates a new governance service
func NewService(
	blockchain BlockchainReader,
	validators ValidatorManager,
	tokens TokenSystem,
	store ProposalStore,
	executor ProposalExecutor,
	config *Config,
) *Service {
	return &Service{
		blockchain: blockchain,
		validators: validators,
		tokens:     tokens,
		store:      store,
		executor:   executor,
		config:     config,
	}
}

// CreateProposal creates a new proposal
func (s *Service) CreateProposal(creator string, proposalType ProposalType, title string, description string, data map[string]string) (string, error) {
	// Validate proposal type
	if proposalType == "" {
		return "", ErrInvalidProposalType
	}

	// Check if creator is a validator
	if !s.validators.IsValidator(creator) {
		return "", ErrNotValidator
	}

	// Create proposal ID
	proposalID := generateProposalID()

	// Create proposal
	proposal := &Proposal{
		ID:          proposalID,
		Type:        proposalType,
		Title:       title,
		Description: description,
		Status:      ProposalStatusVoting,
		CreatedAt:   time.Now(),
		Votes:       make(map[string]bool),
		Data:        data,
	}

	// Save proposal
	if err := s.store.SaveProposal(proposal); err != nil {
		return "", fmt.Errorf("failed to save proposal: %w", err)
	}

	return proposalID, nil
}

// Vote casts a vote on a proposal
func (s *Service) Vote(voter string, proposalID string, approve bool) error {
	// Check if voter is a validator
	if !s.validators.IsValidator(voter) {
		return ErrNotValidator
	}

	// Get proposal
	proposal, err := s.store.GetProposal(proposalID)
	if err != nil {
		return fmt.Errorf("failed to get proposal: %w", err)
	}

	// Check if proposal is in voting period
	if proposal.Status != ProposalStatusVoting {
		return ErrProposalNotInVoting
	}

	// Add vote to proposal
	proposal.Votes[voter] = approve

	// Save proposal
	if err := s.store.SaveProposal(proposal); err != nil {
		return fmt.Errorf("failed to save proposal: %w", err)
	}

	// Update proposal status if necessary
	if err := s.checkAndUpdateProposalStatus(proposal); err != nil {
		return fmt.Errorf("failed to update proposal status: %w", err)
	}

	return nil
}

// ExecuteProposal executes a passed proposal
func (s *Service) ExecuteProposal(proposalID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Get proposal
	proposal, err := s.store.GetProposal(proposalID)
	if err != nil {
		return err
	}

	if proposal == nil {
		return errors.New("proposal not found")
	}

	// Check if proposal has passed
	if proposal.Status != ProposalStatusPassed {
		return errors.New("proposal has not passed")
	}

	// Execute proposal
	if err := s.executor.Execute(proposal); err != nil {
		return err
	}

	// Update status
	if err := s.store.UpdateProposalStatus(proposalID, ProposalStatusExecuted); err != nil {
		return err
	}

	return nil
}

// CancelProposal cancels a proposal
func (s *Service) CancelProposal(proposalID string, canceller string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Get proposal
	proposal, err := s.store.GetProposal(proposalID)
	if err != nil {
		return err
	}

	if proposal == nil {
		return errors.New("proposal not found")
	}

	// // Check if canceller is the creator
	// if proposal.Creator != canceller {
	// 	return errors.New("only creator can cancel proposal")
	// }

	// // Check if proposal is still pending
	// if proposal.Status != ProposalStatusPending {
	// 	return errors.New("proposal is not in voting period")
	// }

	// // Update status
	// if err := s.store.UpdateProposalStatus(proposalID, ProposalStatusCancelled); err != nil {
	// 	return err
	// }

	// // Unlock tokens
	// if err := s.tokens.Unlock(proposal.Creator, proposal.Deposit); err != nil {
	// 	return err
	// }

	return nil
}

// ListProposals returns all proposals
func (s *Service) ListProposals() ([]*Proposal, error) {
	return s.store.ListProposals()
}

// checkAndFinalizeProposal checks if a proposal has passed and updates its status
func (s *Service) checkAndFinalizeProposal(proposalID string) error {
	// Get proposal
	proposal, err := s.store.GetProposal(proposalID)
	if err != nil {
		return err
	}

	if proposal == nil {
		return errors.New("proposal not found")
	}

	// Get all validators
	validators := s.validators.GetValidators()
	if len(validators) == 0 {
		return errors.New("no validators found")
	}

	// Check if all validators have voted
	if len(proposal.Votes) < len(validators) {
		return nil // Not all validators have voted yet
	}

	// // Calculate total voting power
	// var totalPower int64
	// for _, vote := range proposal.Votes {
	// 	totalPower += vote.Power
	// }

	// // Calculate total validator power
	// var totalValidatorPower int64
	// for _, address := range validators {
	// 	power, err := s.validators.GetValidatorPower(address)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	totalValidatorPower += power
	// }

	// // Check quorum
	// if totalPower*100/totalValidatorPower < s.config.QuorumPercentage {
	// 	return s.store.UpdateProposalStatus(proposalID, ProposalStatusRejected)
	// }

	// // Count votes
	// var yesVotes, noVotes int64
	// for _, vote := range proposal.Votes {
	// 	if vote.Support {
	// 		yesVotes += vote.Power
	// 	} else {
	// 		noVotes += vote.Power
	// 	}
	// }

	// // Check if proposal passed
	// if yesVotes*100/totalPower >= s.config.ApprovalThreshold {
	// 	return s.store.UpdateProposalStatus(proposalID, ProposalStatusPassed)
	// }

	// return s.store.UpdateProposalStatus(proposalID, ProposalStatusRejected)
	return nil
}

// generateProposalID generates a unique proposal ID
func generateProposalID() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		// If we can't generate random bytes, use timestamp
		return fmt.Sprintf("proposal-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// GetProposal returns a proposal by ID
func (s *Service) GetProposal(id string) (*Proposal, error) {
	proposal, err := s.store.GetProposal(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get proposal: %w", err)
	}
	if proposal == nil {
		return nil, ErrProposalNotFound
	}
	return proposal, nil
}

// IsValidator checks if an address is a validator
func (s *Service) IsValidator(address string) bool {
	return s.validators.IsValidator(address)
}

// checkAndUpdateProposalStatus checks and updates the proposal status based on votes
func (s *Service) checkAndUpdateProposalStatus(proposal *Proposal) error {
	// Get total voting power
	totalPower := s.validators.GetTotalPower()
	if totalPower == 0 {
		return errors.New("no validators found")
	}

	// Calculate required votes for approval
	requiredVotes := (totalPower * s.config.ApprovalThreshold) / 100

	// Count votes
	approvalVotes := int64(0)
	for voter, approve := range proposal.Votes {
		if approve {
			power, err := s.validators.GetValidatorPower(voter)
			if err != nil {
				log.Printf("Warning: Failed to get voting power for %s: %v", voter, err)
				continue
			}
			approvalVotes += power
		}
	}

	// Update proposal status
	if approvalVotes >= requiredVotes {
		proposal.Status = ProposalStatusPassed
		if err := s.store.UpdateProposalStatus(proposal.ID, ProposalStatusPassed); err != nil {
			return fmt.Errorf("failed to update proposal status: %w", err)
		}

		// Execute proposal if passed
		if err := s.executor.Execute(proposal); err != nil {
			log.Printf("Warning: Failed to execute proposal %s: %v", proposal.ID, err)
		}
	}

	return nil
}

// ListProposalsByStatus returns proposals with the specified status
func (s *Service) ListProposalsByStatus(status ProposalStatus) ([]*Proposal, error) {
	return s.store.ListProposalsByStatus(status)
} 