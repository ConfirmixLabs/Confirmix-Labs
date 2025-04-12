package store

import (
	"sync"

	"confirmix/pkg/governance"
)

// MemoryStore is an in-memory implementation of ProposalStore
type MemoryStore struct {
	proposals map[string]*governance.Proposal
	mutex     sync.RWMutex
}

// NewMemoryStore creates a new memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		proposals: make(map[string]*governance.Proposal),
	}
}

// SaveProposal saves a proposal to the store
func (s *MemoryStore) SaveProposal(proposal *governance.Proposal) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create a copy of the proposal
	copy := *proposal
	copy.Votes = make(map[string]bool)
	for k, v := range proposal.Votes {
		copy.Votes[k] = v
	}
	s.proposals[proposal.ID] = &copy
	return nil
}

// GetProposal retrieves a proposal by ID
func (s *MemoryStore) GetProposal(id string) (*governance.Proposal, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if proposal, exists := s.proposals[id]; exists {
		// Return a copy of the proposal
		copy := *proposal
		copy.Votes = make(map[string]bool)
		for k, v := range proposal.Votes {
			copy.Votes[k] = v
		}
		return &copy, nil
	}
	return nil, nil
}

// ListProposals lists all proposals
func (s *MemoryStore) ListProposals() ([]*governance.Proposal, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	proposals := make([]*governance.Proposal, 0, len(s.proposals))
	for _, proposal := range s.proposals {
		// Return a copy of each proposal
		copy := *proposal
		proposals = append(proposals, &copy)
	}
	return proposals, nil
}

// ListProposalsByStatus lists proposals by status
func (s *MemoryStore) ListProposalsByStatus(status governance.ProposalStatus) ([]*governance.Proposal, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	proposals := make([]*governance.Proposal, 0)
	for _, proposal := range s.proposals {
		if proposal.Status == status {
			// Return a copy of each proposal
			copy := *proposal
			proposals = append(proposals, &copy)
		}
	}
	return proposals, nil
}

// UpdateProposalStatus updates a proposal's status
func (s *MemoryStore) UpdateProposalStatus(id string, status governance.ProposalStatus) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if proposal, exists := s.proposals[id]; exists {
		// Create a copy of the proposal
		copy := *proposal
		copy.Status = status
		s.proposals[id] = &copy
		return nil
	}
	return nil
}

// AddVote adds a vote to a proposal
func (s *MemoryStore) AddVote(id string, vote *governance.Vote) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if proposal, exists := s.proposals[id]; exists {
		// Create a copy of the proposal
		copy := *proposal
		copy.Votes[vote.Voter] = vote.Approve
		s.proposals[id] = &copy
		return nil
	}
	return nil
} 