package governance_test

import (
	"math/big"
	"testing"

	"confirmix/pkg/governance"
	"confirmix/pkg/governance/executor"
	"confirmix/pkg/governance/store"
	"confirmix/pkg/token"
	"confirmix/pkg/validator"
	"github.com/stretchr/testify/assert"
)

// MockBlockchainReader implements BlockchainReader interface
type MockBlockchainReader struct{}

func (m *MockBlockchainReader) GetBlockHeight() int64 {
	return 1
}

func (m *MockBlockchainReader) GetBlockHash(height int64) string {
	return "mock_hash"
}

// newTokenSystem creates a new token system with initial balances
func newTokenSystem(t *testing.T) *token.System {
	tokenSystem := token.NewSystem()
	
	// Set initial balances
	deposit := big.NewInt(10000) // 10000 tokens
	err := tokenSystem.SetBalance("validator1", deposit)
	assert.NoError(t, err)
	
	err = tokenSystem.SetBalance("validator2", deposit)
	assert.NoError(t, err)
	
	err = tokenSystem.SetBalance("user1", deposit)
	assert.NoError(t, err)
	
	return tokenSystem
}

// newService creates a new governance service with mock dependencies
func newService(t *testing.T) *governance.Service {
	// Create dependencies
	blockchain := &MockBlockchainReader{}
	validators := validator.NewManager()
	tokens := newTokenSystem(t)
	store := store.NewMemoryStore()
	executor := executor.NewExecutor(validators)

	// Create validators
	validator1 := validators.CreateValidator("validator1", 100)
	assert.NotNil(t, validator1)
	validator2 := validators.CreateValidator("validator2", 100)
	assert.NotNil(t, validator2)

	config := governance.DefaultConfig()

	// Create service
	service := governance.NewService(
		blockchain,
		validators,
		tokens,
		store,
		executor,
		config,
	)

	return service
}

func TestGovernance(t *testing.T) {
	service := newService(t)

	t.Run("Create Proposal", func(t *testing.T) {
		// Create a proposal
		proposalID, err := service.CreateProposal(
			"validator1",
			governance.ProposalTypeAddValidator,
			"Add New Validator",
			"Add a new validator to the network",
			map[string]string{
				"address": "validator3",
				"power":   "100",
			},
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, proposalID)

		// Get proposal
		proposal, err := service.GetProposal(proposalID)
		assert.NoError(t, err)
		assert.Equal(t, governance.ProposalStatusVoting, proposal.Status)
	})

	t.Run("Vote on Proposal", func(t *testing.T) {
		// Create a proposal
		proposalID, err := service.CreateProposal(
			"validator1",
			governance.ProposalTypeText,
			"Test Proposal",
			"This is a test proposal",
			map[string]string{},
		)
		assert.NoError(t, err)

		// Vote on the proposal
		err = service.Vote("validator1", proposalID, true)
		assert.NoError(t, err)

		err = service.Vote("validator2", proposalID, true)
		assert.NoError(t, err)

		// Get proposal and check status
		proposal, err := service.GetProposal(proposalID)
		assert.NoError(t, err)
		assert.Equal(t, governance.ProposalStatusPassed, proposal.Status)
	})

	t.Run("Proposal Rejection", func(t *testing.T) {
		// Create a proposal
		proposalID, err := service.CreateProposal(
			"validator1",
			governance.ProposalTypeText,
			"Test Proposal",
			"This is a test proposal",
			map[string]string{},
		)
		assert.NoError(t, err)

		// Vote against the proposal
		err = service.Vote("validator1", proposalID, false)
		assert.NoError(t, err)

		err = service.Vote("validator2", proposalID, false)
		assert.NoError(t, err)

		// Get proposal and check status
		proposal, err := service.GetProposal(proposalID)
		assert.NoError(t, err)
		assert.Equal(t, governance.ProposalStatusVoting, proposal.Status) // Should still be voting, as it didn't pass
	})
} 