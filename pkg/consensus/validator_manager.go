package consensus

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
	
	"confirmix/pkg/blockchain"
	"confirmix/pkg/types"
)

// ValidatorStatus represents the status of a validator
type ValidatorStatus string

const (
	StatusPending   ValidatorStatus = "pending"   // Awaiting approval
	StatusApproved  ValidatorStatus = "approved"  // Approved by admin or governance
	StatusRejected  ValidatorStatus = "rejected"  // Rejected
	StatusSuspended ValidatorStatus = "suspended" // Temporarily suspended
)

// ValidatorInfo contains validator information
type ValidatorInfo struct {
	Address     string          // Blockchain address
	HumanProof  string          // Human proof token
	Status      ValidatorStatus // Current status
	JoinedAt    time.Time       // When they joined the validator set
	ApprovedBy  string          // Who approved the validator (address or "governance")
	PerformanceScore float64    // 0-100 score based on performance metrics
	TotalBlocks uint64          // Total blocks produced
	LastActive  time.Time       // Last activity timestamp
}

// ValidationMode defines how validators are approved
type ValidationMode int

const (
	ModeAdminOnly ValidationMode = iota // Only administrators can approve
	ModeHybrid                          // Admin or governance can approve
	ModeGovernance                      // Only governance (voting) can approve
	ModeAutomatic                       // Automatic approval based on criteria
)

// ProposalType represents the type of proposal
type ProposalType string

const (
	ProposalTypeAddValidator    ProposalType = "ADD_VALIDATOR"
	ProposalTypeRemoveValidator ProposalType = "REMOVE_VALIDATOR"
	ProposalTypeChangeParams    ProposalType = "CHANGE_PARAMS"
)

// ProposalStatus represents the status of a proposal
type ProposalStatus string

const (
	ProposalStatusPending   ProposalStatus = "PENDING"
	ProposalStatusApproved  ProposalStatus = "APPROVED"
	ProposalStatusRejected  ProposalStatus = "REJECTED"
	ProposalStatusExecuted  ProposalStatus = "EXECUTED"
	ProposalStatusCancelled ProposalStatus = "CANCELLED"
	ProposalStatusAll      ProposalStatus = "ALL"
)

// Proposal represents a governance proposal
type Proposal struct {
	ID          string         `json:"id"`
	Type        ProposalType   `json:"type"`
	Creator     string         `json:"creator"`
	Status      ProposalStatus `json:"status"`
	CreateTime  int64          `json:"createTime"`
	UpdateTime  int64          `json:"updateTime"`
	Data        interface{}    `json:"data"`
	Votes       map[string]bool `json:"votes"`
	Description string         `json:"description"`
}

// Governance represents the governance system
type Governance struct {
	proposals       map[string]*Proposal
	validators     map[string]bool
	requiredVotes  int
	mutex          sync.RWMutex
}

// NewGovernance creates a new governance system
func NewGovernance(validators map[string]bool, requiredVotes int) *Governance {
	return &Governance{
		proposals:      make(map[string]*Proposal),
		validators:     validators,
		requiredVotes:  requiredVotes,
	}
}

// ValidatorManager handles validator registration and approval
type ValidatorManager struct {
	blockchain       *blockchain.Blockchain
	validators       map[string]*ValidatorInfo
	adminAddresses   map[string]bool
	mutex            sync.RWMutex
	mode             ValidationMode
	pohVerifier      *ProofOfHumanity
	externalVerifier *ExternalPoHVerifier
	useExternalPoh   bool
	admins           map[string]bool
}

// NewValidatorManager creates a new validator manager
func NewValidatorManager(bc *blockchain.Blockchain, initialAdmins []string, mode ValidationMode) *ValidatorManager {
	adminMap := make(map[string]bool)
	for _, admin := range initialAdmins {
		adminMap[admin] = true
	}
	
	vm := &ValidatorManager{
		blockchain:     bc,
		validators:     make(map[string]*ValidatorInfo),
		adminAddresses: adminMap,
		mode:           mode,
		pohVerifier:    NewProofOfHumanity(30 * 24 * time.Hour), // 30 days expiration
		admins:         make(map[string]bool),
	}
	
	// Initialize with existing validators from blockchain
	validators := bc.GetValidators()
	for _, validator := range validators {
		vm.validators[validator.Address] = &ValidatorInfo{
			Address:     validator.Address,
			HumanProof:  validator.HumanProof,
			Status:      StatusApproved,
			JoinedAt:    time.Now(), // We don't know the actual time
			ApprovedBy:  "system_initialization",
			PerformanceScore: 100.0, // Initial perfect score
			LastActive:  time.Now(),
		}
	}
	
	return vm
}

// SetupExternalPoH sets up external proof of humanity verification
func (vm *ValidatorManager) SetupExternalPoH(baseURL, apiKey string, useSimulator bool) {
	vm.externalVerifier = NewExternalPoHVerifier(baseURL, apiKey, useSimulator)
	vm.useExternalPoh = true
}

// IsAdmin checks if an address is an admin
func (vm *ValidatorManager) IsAdmin(address string) bool {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()
	return vm.adminAddresses[address]
}

// AddAdmin adds a new admin address if called by an existing admin
func (vm *ValidatorManager) AddAdmin(newAdminAddress string, callerAddress string) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()
	
	// Check if caller is an admin
	if !vm.IsAdmin(callerAddress) {
		return errors.New("only existing admins can add new admins")
	}
	
	// Add the new admin
	vm.adminAddresses[newAdminAddress] = true
	log.Printf("New admin added: %s (by %s)", newAdminAddress, callerAddress)
	return nil
}

// InitializeFirstAdmin sets the first admin of the system
// This should only be called during initial setup
func (vm *ValidatorManager) InitializeFirstAdmin(adminAddress string) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()
	
	// Check if admins already exist
	if len(vm.adminAddresses) > 0 {
		return errors.New("admin(s) already initialized")
	}
	
	// Add the initial admin
	vm.adminAddresses[adminAddress] = true
	log.Printf("Initial admin initialized: %s", adminAddress)
	return nil
}

// RemoveAdmin removes an admin address if called by another admin
func (vm *ValidatorManager) RemoveAdmin(adminToRemove string, callerAddress string) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()
	
	// Check if caller is an admin
	if !vm.IsAdmin(callerAddress) {
		return errors.New("only existing admins can remove admins")
	}
	
	// Check if admin exists
	if _, exists := vm.adminAddresses[adminToRemove]; !exists {
		return errors.New("admin address does not exist")
	}
	
	// Prevent removing the last admin
	if len(vm.adminAddresses) <= 1 {
		return errors.New("cannot remove the last admin")
	}
	
	// Remove the admin
	delete(vm.adminAddresses, adminToRemove)
	log.Printf("Admin removed: %s (by %s)", adminToRemove, callerAddress)
	return nil
}

// GetAdmins returns a list of all admin addresses
func (vm *ValidatorManager) GetAdmins() []string {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()
	
	admins := make([]string, 0, len(vm.adminAddresses))
	for address := range vm.adminAddresses {
		admins = append(admins, address)
	}
	return admins
}

// RegisterValidator initiates validator registration process
func (vm *ValidatorManager) RegisterValidator(address, humanProof string) error {
	// Verify that the address has a valid human proof
	verified := false
	
	if vm.useExternalPoh && vm.externalVerifier != nil {
		// Use external verifier
		var err error
		verified, err = vm.externalVerifier.VerifyHumanity(address, humanProof)
		if err != nil {
			return fmt.Errorf("external human verification failed: %v", err)
		}
	} else {
		// Use internal verifier
		verified = vm.pohVerifier.IsHumanVerified(address)
		if !verified {
			// Try to complete verification
			err := vm.pohVerifier.CompleteVerification(address, humanProof)
			if err != nil {
				return fmt.Errorf("internal human verification failed: %v", err)
			}
			verified = true
		}
	}
	
	if !verified {
		return errors.New("address is not verified as human")
	}
	
	vm.mutex.Lock()
	defer vm.mutex.Unlock()
	
	// Check if already registered
	if _, exists := vm.validators[address]; exists {
		return errors.New("validator already registered")
	}
	
	// Create new validator with pending status
	validator := &ValidatorInfo{
		Address:     address,
		HumanProof:  humanProof,
		Status:      StatusPending,
		JoinedAt:    time.Time{}, // Not set until approved
		PerformanceScore: 0,      // No score until approved
		LastActive:  time.Now(),
	}
	
	// If automatic mode, approve immediately
	if vm.mode == ModeAutomatic {
		validator.Status = StatusApproved
		validator.JoinedAt = time.Now()
		validator.ApprovedBy = "automatic"
		validator.PerformanceScore = 100.0
		
		// Register with blockchain
		if err := vm.blockchain.RegisterValidator(address, humanProof); err != nil {
			return fmt.Errorf("blockchain registration failed: %v", err)
		}
	}
	
	vm.validators[address] = validator
	log.Printf("Validator registered: %s (status: %s)", address, validator.Status)
	return nil
}

// ApproveValidator approves a validator registration request
func (vm *ValidatorManager) ApproveValidator(adminAddress, validatorAddress string) error {
	// Check if the admin is authorized
	if !vm.IsAdmin(adminAddress) {
		return fmt.Errorf("unauthorized: address %s is not an admin", adminAddress)
	}

	// Check if validator exists
	validator, exists := vm.validators[validatorAddress]
	if !exists {
		return fmt.Errorf("validator %s not found", validatorAddress)
	}

	// Check if validator is already approved
	if validator.Status == StatusApproved {
		return fmt.Errorf("validator %s is already approved", validatorAddress)
	}

	// Update validator status
	validator.Status = StatusApproved
	validator.ApprovedBy = adminAddress
	validator.JoinedAt = time.Now()

	// Save to blockchain
	if err := vm.blockchain.SaveToDisk(); err != nil {
		return fmt.Errorf("failed to save validator status: %v", err)
	}

	return nil
}

// SuspendValidator suspends an approved validator
func (vm *ValidatorManager) SuspendValidator(requesterAddress, validatorAddress, reason string) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()
	
	// Check requester permissions based on mode
	if vm.mode == ModeAdminOnly || vm.mode == ModeHybrid {
		if !vm.adminAddresses[requesterAddress] {
			return errors.New("only admins can suspend validators in this mode")
		}
	} else if vm.mode == ModeGovernance {
		return errors.New("in governance mode, validators must be suspended through governance votes")
	}
	
	// Check if validator exists and is approved
	validator, exists := vm.validators[validatorAddress]
	if !exists {
		return errors.New("validator not found")
	}
	
	if validator.Status != StatusApproved {
		return fmt.Errorf("validator is not active (current status: %s)", validator.Status)
	}
	
	// Update validator status
	validator.Status = StatusSuspended
	
	// Remove from blockchain validator set
	if err := vm.blockchain.RemoveValidator(validatorAddress); err != nil {
		validator.Status = StatusApproved // Revert on error
		return fmt.Errorf("failed to remove validator from blockchain: %v", err)
	}
	
	log.Printf("Validator suspended: %s (by %s) - Reason: %s", validatorAddress, requesterAddress, reason)
	return nil
}

// GetValidators returns all validators with optional filtering
func (vm *ValidatorManager) GetValidators(statusFilter ...ValidatorStatus) []*ValidatorInfo {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()
	
	var validators []*ValidatorInfo
	
	if len(statusFilter) == 0 {
		// Return all validators
		validators = make([]*ValidatorInfo, 0, len(vm.validators))
		for _, validator := range vm.validators {
			validators = append(validators, validator)
		}
	} else {
		// Filter by status
		validators = make([]*ValidatorInfo, 0)
		statusMap := make(map[ValidatorStatus]bool)
		for _, status := range statusFilter {
			statusMap[status] = true
		}
		
		for _, validator := range vm.validators {
			if statusMap[validator.Status] {
				validators = append(validators, validator)
			}
		}
	}
	
	return validators
}

// IsValidator checks if an address is an approved validator
func (vm *ValidatorManager) IsValidator(address string) bool {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()
	
	validator, exists := vm.validators[address]
	if !exists {
		return false
	}
	
	return validator.Status == StatusApproved
}

// UpdateValidatorMode changes the validator approval mode
func (vm *ValidatorManager) UpdateValidatorMode(requesterAddress string, newMode ValidationMode) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()
	
	// Only admins can change the mode
	if !vm.adminAddresses[requesterAddress] {
		return errors.New("only admins can change the validation mode")
	}
	
	vm.mode = newMode
	log.Printf("Validation mode updated to: %d (by %s)", newMode, requesterAddress)
	return nil
}

// UpdateValidatorPerformance updates a validator's performance score
func (vm *ValidatorManager) UpdateValidatorPerformance(address string, newScore float64) {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()
	
	validator, exists := vm.validators[address]
	if !exists || validator.Status != StatusApproved {
		return
	}
	
	// Update the performance score
	validator.PerformanceScore = newScore
	validator.LastActive = time.Now()
	
	// Auto-suspend validators with very poor performance
	if newScore < 10.0 {
		validator.Status = StatusSuspended
		vm.blockchain.RemoveValidator(address) // Remove from active validator set
		log.Printf("Validator auto-suspended due to poor performance: %s (score: %.2f)", address, newScore)
	}
}

// RejectValidator rejects a pending validator
func (vm *ValidatorManager) RejectValidator(validatorAddress, requesterAddress, reason string) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()
	
	// Check requester permissions based on mode
	if vm.mode == ModeAdminOnly || vm.mode == ModeHybrid {
		if !vm.adminAddresses[requesterAddress] {
			return errors.New("only admins can reject validators in this mode")
		}
	} else if vm.mode == ModeGovernance {
		return errors.New("in governance mode, validators must be rejected through governance votes")
	}
	
	// Check if validator exists and is pending
	validator, exists := vm.validators[validatorAddress]
	if !exists {
		return errors.New("validator not found")
	}
	
	if validator.Status != StatusPending {
		return fmt.Errorf("validator is not pending (current status: %s)", validator.Status)
	}
	
	// Update validator status
	validator.Status = StatusRejected
	
	log.Printf("Validator rejected: %s (by %s) - Reason: %s", validatorAddress, requesterAddress, reason)
	return nil
}

// VerifySignature verifies the signature on a signed request
func (vm *ValidatorManager) VerifySignature(req *types.SignedRequest) (bool, error) {
	// Get the admin's key pair
	keyPair, exists := vm.blockchain.GetKeyPair(req.AdminAddress)
	if !exists {
		return false, fmt.Errorf("admin key pair not found")
	}

	// Create the message to verify
	message := fmt.Sprintf("%s:%s:%d", req.Action, req.AdminAddress, req.Timestamp)
	
	// Verify the signature
	valid, err := vm.blockchain.VerifySignature(message, req.Signature, keyPair.PublicKey)
	if err != nil {
		return false, fmt.Errorf("signature verification failed: %v", err)
	}

	return valid, nil
}

// ListProposals returns all proposals
func (g *Governance) ListProposals(status ProposalStatus) []*Proposal {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	var result []*Proposal
	for _, p := range g.proposals {
		if status == "" || p.Status == status {
			result = append(result, p)
		}
	}
	return result
}

// GetProposal returns a proposal by ID
func (g *Governance) GetProposal(id string) (*Proposal, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if p, exists := g.proposals[id]; exists {
		return p, nil
	}
	return nil, errors.New("proposal not found")
}

// CreateProposal creates a new proposal
func (g *Governance) CreateProposal(creator string, proposalType ProposalType, data interface{}, description string) (*Proposal, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.validators[creator] {
		return nil, errors.New("only validators can create proposals")
	}

	now := time.Now().Unix()
	proposal := &Proposal{
		ID:          fmt.Sprintf("proposal_%d", now),
		Type:        proposalType,
		Creator:     creator,
		Status:      ProposalStatusPending,
		CreateTime:  now,
		UpdateTime:  now,
		Data:        data,
		Votes:       make(map[string]bool),
		Description: description,
	}

	g.proposals[proposal.ID] = proposal
	return proposal, nil
}

// CastVote casts a vote on a proposal
func (g *Governance) CastVote(proposalID string, voter string, approve bool) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.validators[voter] {
		return errors.New("only validators can vote")
	}

	proposal, exists := g.proposals[proposalID]
	if !exists {
		return errors.New("proposal not found")
	}

	if proposal.Status != ProposalStatusPending {
		return errors.New("proposal is not pending")
	}

	proposal.Votes[voter] = approve
	proposal.UpdateTime = time.Now().Unix()

	// Count votes
	approvals := 0
	rejections := 0
	for _, vote := range proposal.Votes {
		if vote {
			approvals++
		} else {
			rejections++
		}
	}

	// Check if proposal can be approved or rejected
	if approvals >= g.requiredVotes {
		proposal.Status = ProposalStatusApproved
	} else if rejections >= g.requiredVotes {
		proposal.Status = ProposalStatusRejected
	}

	return nil
} 