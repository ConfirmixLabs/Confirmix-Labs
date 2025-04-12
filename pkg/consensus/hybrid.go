package consensus

import (
	"crypto/ecdsa"
	"errors"
	"time"

	"confirmix/pkg/blockchain"
)

// HybridConsensus combines Proof of Authority and Proof of Humanity
type HybridConsensus struct {
	poaConsensus      *PoAConsensus
	pohVerifier       *ProofOfHumanity
	externalPohVerifier *ExternalPoHVerifier
	useExternalPoh    bool
	blockchain        *blockchain.Blockchain
	address           string
	isValidator       bool
}

// HybridConsensusConfig represents configuration for the hybrid consensus
type HybridConsensusConfig struct {
	UseExternalPoh      bool
	ExternalPohBaseURL  string
	ExternalPohAPIKey   string
	UsePoHSimulator     bool
	PoHSimulatorPort    int
	BlockTime           time.Duration
}

// DefaultHybridConsensusConfig returns the default configuration
func DefaultHybridConsensusConfig() *HybridConsensusConfig {
	return &HybridConsensusConfig{
		UseExternalPoh:   false,
		UsePoHSimulator:  true,
		PoHSimulatorPort: 8080,
		BlockTime:        15 * time.Second,
	}
}

// NewHybridConsensus creates a new hybrid consensus engine
func NewHybridConsensus(bc *blockchain.Blockchain, privateKey *ecdsa.PrivateKey, address string, blockTime time.Duration) *HybridConsensus {
	config := DefaultHybridConsensusConfig()
	config.BlockTime = blockTime
	return NewHybridConsensusWithConfig(bc, privateKey, address, config)
}

// NewHybridConsensusWithConfig creates a new hybrid consensus engine with custom configuration
func NewHybridConsensusWithConfig(bc *blockchain.Blockchain, privateKey *ecdsa.PrivateKey, address string, config *HybridConsensusConfig) *HybridConsensus {
	// Create PoH verifier with 30-day verification expiration
	pohVerifier := NewProofOfHumanity(30 * 24 * time.Hour)
	
	// Start the cleanup routine to remove expired verifications every hour
	pohVerifier.StartCleanupRoutine(time.Hour)
	
	// Create external PoH verifier if enabled
	var externalPohVerifier *ExternalPoHVerifier
	if config.UseExternalPoh {
		externalPohVerifier = NewExternalPoHVerifier(
			config.ExternalPohBaseURL,
			config.ExternalPohAPIKey,
			config.UsePoHSimulator,
		)
		
		// Start PoH simulator server if enabled
		if config.UsePoHSimulator && externalPohVerifier.localSimulator != nil {
			externalPohVerifier.localSimulator.StartSimulatorServer(config.PoHSimulatorPort)
		}
	}
	
	return &HybridConsensus{
		poaConsensus:       NewPoAConsensus(bc, privateKey, address, config.BlockTime, ""),
		pohVerifier:        pohVerifier,
		externalPohVerifier: externalPohVerifier,
		useExternalPoh:     config.UseExternalPoh,
		blockchain:         bc,
		address:            address,
		isValidator:        false,
	}
}

// RegisterAsValidator registers this node as a validator after human verification
func (hc *HybridConsensus) RegisterAsValidator() error {
	// First check if we have human verification
	verified := false
	var proofToken string
	
	if hc.useExternalPoh && hc.externalPohVerifier != nil {
		// Check with external verifier
		isVerified, err := hc.externalPohVerifier.VerifyHumanity(hc.address, hc.poaConsensus.humanProof)
		if err != nil {
			return err
		}
		
		if isVerified {
			verified = true
			proofToken = hc.poaConsensus.humanProof
		}
	} else {
		// Use internal verifier
		if hc.pohVerifier.IsHumanVerified(hc.address) {
			verified = true
			
			// Get proof token
			var err error
			proofToken, err = hc.pohVerifier.GetProofToken(hc.address)
			if err != nil {
				return err
			}
		}
	}
	
	if !verified {
		return errors.New("human verification required to become validator")
	}
	
	// Update PoA consensus with the human proof
	hc.poaConsensus.humanProof = proofToken
	
	// Register as validator
	err := hc.poaConsensus.RegisterAsValidator()
	if err != nil {
		return err
	}
	
	hc.isValidator = true
	return nil
}

// InitiateHumanVerification starts the human verification process
func (hc *HybridConsensus) InitiateHumanVerification() (string, error) {
	if hc.useExternalPoh && hc.externalPohVerifier != nil {
		return hc.externalPohVerifier.InitiateVerification(hc.address)
	}
	
	// In a real system, this would initiate an external verification process
	return hc.pohVerifier.RegisterVerification(hc.address)
}

// CompleteHumanVerification completes the human verification process
func (hc *HybridConsensus) CompleteHumanVerification(proofToken string) error {
	if hc.useExternalPoh && hc.externalPohVerifier != nil {
		verified, err := hc.externalPohVerifier.VerifyHumanity(hc.address, proofToken)
		if err != nil {
			return err
		}
		
		if !verified {
			return errors.New("humanity verification failed")
		}
		
		// Store the proof token for later use
		hc.poaConsensus.humanProof = proofToken
		return nil
	}
	
	return hc.pohVerifier.CompleteVerification(hc.address, proofToken)
}

// IsHumanVerified checks if the node has been verified as human
func (hc *HybridConsensus) IsHumanVerified() bool {
	if hc.useExternalPoh && hc.externalPohVerifier != nil {
		if hc.poaConsensus.humanProof == "" {
			return false
		}
		
		verified, err := hc.externalPohVerifier.VerifyHumanity(hc.address, hc.poaConsensus.humanProof)
		if err != nil {
			return false
		}
		
		return verified
	}
	
	return hc.pohVerifier.IsHumanVerified(hc.address)
}

// GetVerificationURL returns the URL for human verification
func (hc *HybridConsensus) GetVerificationURL() (string, error) {
	if !hc.useExternalPoh || hc.externalPohVerifier == nil {
		return "", errors.New("external PoH verifier not enabled")
	}
	
	if hc.poaConsensus.humanProof == "" {
		return "", errors.New("verification not initiated")
	}
	
	return hc.externalPohVerifier.GetVerificationURL(hc.address, hc.poaConsensus.humanProof), nil
}

// StartMining starts the block production process
func (hc *HybridConsensus) StartMining() error {
	// Check if node is a validator
	if !hc.isValidator {
		// Try to register as validator if human verified
		if hc.IsHumanVerified() {
			err := hc.RegisterAsValidator()
			if err != nil {
				return err
			}
		} else {
			return errors.New("not a validator - human verification required")
		}
	}
	
	// Start the PoA consensus mining process
	return hc.poaConsensus.StartMining()
}

// StopMining stops the block production process
func (hc *HybridConsensus) StopMining() {
	hc.poaConsensus.StopMining()
}

// UpdateValidatorList updates the list of authorized validators
func (hc *HybridConsensus) UpdateValidatorList(validators []string) {
	hc.poaConsensus.UpdateValidatorList(validators)
}

// VerifyBlock verifies that a block is valid according to the hybrid rules
func (hc *HybridConsensus) VerifyBlock(block *blockchain.Block) error {
	// Check PoA rules
	err := hc.poaConsensus.VerifyBlock(block)
	if err != nil {
		return err
	}
	
	// Check that the HumanProof in the block is valid
	if !hc.validateHumanProof(block.Validator, block.HumanProof) {
		return errors.New("invalid human proof in block")
	}
	
	return nil
}

// validateHumanProof checks if a human proof is valid for a validator
// For demonstration, we're simply checking that it matches what's stored in the blockchain
// In a real system, this would validate with the PoH registry
func (hc *HybridConsensus) validateHumanProof(validator string, proof string) bool {
	if hc.useExternalPoh && hc.externalPohVerifier != nil {
		verified, err := hc.externalPohVerifier.VerifyHumanity(validator, proof)
		if err != nil {
			return false
		}
		
		return verified
	}
	
	expectedProof := hc.blockchain.GetHumanProof(validator)
	return proof == expectedProof
}

// GetNodeAddress returns the address of this node
func (hc *HybridConsensus) GetNodeAddress() string {
	return hc.address
} 