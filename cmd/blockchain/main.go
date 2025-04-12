package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
	"crypto/sha256"

	"confirmix/pkg/blockchain"
	"confirmix/pkg/consensus"
	"confirmix/pkg/network"
	"confirmix/pkg/api"
)

// NodeConfig represents the node configuration
type NodeConfig struct {
	Address           string   `json:"address"`
	Port              int      `json:"port"`
	PrivateKeyPEM     string   `json:"private_key_pem"`
	IsValidator       bool     `json:"is_validator"`
	HumanProof        string   `json:"human_proof"`
	PeerAddresses     []string `json:"peer_addresses"`
	GovernanceEnabled bool     `json:"governance_enabled"` // Whether to enable governance features
	ValidatorMode     string   `json:"validator_mode"`     // Validator approval mode: admin, hybrid, governance, automatic
	AdminAddress      string   `json:"admin_address"`      // Admin address for validator approvals (in admin mode)
	ValidatorAddress  string   `json:"validator_address"`    // Validator address for registration
}

func main() {
	log.Printf("Starting blockchain node...")

	// Define command line flags
	nodeCmd := flag.NewFlagSet("node", flag.ExitOnError)
	log.Printf("Setting up command line flags...")
	validatorFlag := nodeCmd.Bool("validator", false, "Run as a validator")
	addressFlag := nodeCmd.String("address", "127.0.0.1", "Node address")
	portFlag := nodeCmd.Int("port", 8000, "Node port")
	configFlag := nodeCmd.String("config", "", "Configuration file path")
	peersFlag := nodeCmd.String("peers", "", "Comma-separated list of peer addresses")
	pohVerifyFlag := nodeCmd.Bool("poh-verify", false, "Enable PoH verification")
	governanceFlag := nodeCmd.Bool("governance", false, "Enable governance features")
	validatorModeFlag := nodeCmd.String("validator-mode", "admin", "Validator approval mode: admin, hybrid, governance, automatic")
	adminAddressFlag := nodeCmd.String("admin", "", "Admin address for validator approvals (in admin mode)")

	log.Printf("Parsing command line arguments...")
	// Parse command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Expected 'node' subcommand")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "node":
		log.Printf("Parsing node command arguments...")
		nodeCmd.Parse(os.Args[2:])
	default:
		fmt.Println("Expected 'node' subcommand")
		os.Exit(1)
	}

	log.Printf("Loading configuration...")
	// Load or create configuration
	config := &NodeConfig{
		Address:           *addressFlag,
		Port:              *portFlag,
		IsValidator:       *validatorFlag,
		PeerAddresses:     []string{},
		GovernanceEnabled: *governanceFlag,
		ValidatorMode:     *validatorModeFlag,
		AdminAddress:      *adminAddressFlag,
		ValidatorAddress:  "",
	}

	if *configFlag != "" {
		log.Printf("Loading configuration from file: %s", *configFlag)
		// Load configuration from file
		configData, err := ioutil.ReadFile(*configFlag)
		if err == nil {
			err = json.Unmarshal(configData, config)
			if err != nil {
				log.Fatalf("Failed to parse config file: %v", err)
			}
		}
	}

	log.Printf("Parsing peer addresses...")
	// Parse peer addresses
	if *peersFlag != "" {
		config.PeerAddresses = strings.Split(*peersFlag, ",")
	}

	log.Printf("Loading or creating private key...")
	// Create or load private key
	privateKey, err := loadOrCreatePrivateKey(config)
	if err != nil {
		log.Fatalf("Failed to load or create private key: %v", err)
	}

	log.Printf("Creating node address...")
	// Create node address from public key
	publicKey := privateKey.PublicKey
	publicKeyBytes := elliptic.Marshal(publicKey.Curve, publicKey.X, publicKey.Y)
	// Use full public key hash for address
	hash := sha256.Sum256(publicKeyBytes)
	nodeAddress := fmt.Sprintf("%x", hash)

	// Set fixed admin address
	config.AdminAddress = "0x0000000000000000000000000000000000000000admin"

	log.Printf("Creating blockchain...")
	// Create blockchain
	bc, err := blockchain.NewBlockchain()
	if err != nil {
		log.Fatalf("Failed to create blockchain: %v", err)
	}

	log.Printf("Setting up validator management...")
	// Set up validator management
	var validationMode consensus.ValidationMode
	switch strings.ToLower(config.ValidatorMode) {
	case "admin":
		validationMode = consensus.ModeAdminOnly
	case "hybrid":
		validationMode = consensus.ModeHybrid
	case "governance":
		validationMode = consensus.ModeGovernance
	case "automatic":
		validationMode = consensus.ModeAutomatic
	default:
		validationMode = consensus.ModeAdminOnly
		log.Printf("Warning: Unknown validator mode '%s', defaulting to 'admin'", config.ValidatorMode)
	}

	// Initialize ValidatorManager with empty admin list (genesis will be added later)
	validatorManager := consensus.NewValidatorManager(bc, []string{}, validationMode)
	
	// Add initial admin if specified
	if config.AdminAddress != "" {
		// Check if this is a first run with no existing admins
		admins := validatorManager.GetAdmins()
		if len(admins) == 0 {
			// First initialization with no existing admins
			if err := validatorManager.InitializeFirstAdmin(config.AdminAddress); err != nil {
				log.Printf("Failed to initialize admin address: %v", err)
			} else {
				log.Printf("Initial admin initialized: %s", config.AdminAddress)
			}
		} else {
			log.Printf("Admin address specified but admins already exist. Use admin functionality to add new admins.")
			log.Printf("Existing admins: %v", admins)
		}
	}

	// Register validator if specified
	if config.ValidatorAddress != "" {
		// Check if this is a first run with no existing validators
		validators := validatorManager.GetValidators()
		if len(validators) == 0 {
			// First initialization with no existing validators
			if err := validatorManager.RegisterValidator(config.ValidatorAddress, "Initial validator"); err != nil {
				log.Printf("Failed to register validator: %v", err)
			} else {
				log.Printf("Initial validator registered: %s", config.ValidatorAddress)
			}
		} else {
			log.Printf("Validator address specified but validators already exist. Use validator functionality to add new validators.")
			log.Printf("Existing validators: %v", validators)
		}
	}

	// Initialize Governance system if enabled
	var governanceSystem *consensus.Governance
	if config.GovernanceEnabled {
		// Get current validators
		validators := validatorManager.GetValidators()
		validatorMap := make(map[string]bool)
		for _, v := range validators {
			validatorMap[v.Address] = true
		}
		
		// Initialize governance with current validators and required votes
		requiredVotes := len(validators)/2 + 1 // Simple majority
		governanceSystem = consensus.NewGovernance(validatorMap, requiredVotes)
		log.Printf("Governance system initialized with %d validators, requiring %d votes", len(validators), requiredVotes)
	}

	// Create consensus engine
	hybridConsensus := consensus.NewHybridConsensus(bc, privateKey, nodeAddress, 15*time.Second)

	// Create P2P network node
	p2pNode := network.NewP2PNode(config.Address, config.Port, bc)

	// Initialize node
	initializeNode(config, hybridConsensus, p2pNode, *pohVerifyFlag, validatorManager)

	// Start P2P node
	err = p2pNode.Start()
	if err != nil {
		log.Fatalf("Failed to start P2P node: %v", err)
	}
	defer p2pNode.Stop()

	// Save configuration
	saveConfig(config)

	// Handle node startup based on configuration
	if config.IsValidator {
		err = hybridConsensus.StartMining()
		if err != nil {
			log.Printf("Failed to start mining: %v", err)
			if *pohVerifyFlag {
				// Initiate human verification if needed
				initiateHumanVerification(hybridConsensus)
			}
		}
	}

	// Connect to known peers
	for _, peerAddr := range config.PeerAddresses {
		err = p2pNode.ConnectToPeer(peerAddr)
		if err != nil {
			log.Printf("Failed to connect to peer %s: %v", peerAddr, err)
		}
	}
	
	// Start API server if enabled
	apiPort := 8080 // Default API port
	webServer := api.NewWebServer(bc, hybridConsensus, validatorManager, governanceSystem, apiPort)
	go func() {
		if err := webServer.Start(); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()
	log.Printf("API server started on port %d", apiPort)

	// Wait for interrupt signal
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)
	<-interruptChan

	// Cleanup
	hybridConsensus.StopMining()
	fmt.Println("Blockchain node stopped")
}

// loadOrCreatePrivateKey loads an existing private key or creates a new one
func loadOrCreatePrivateKey(config *NodeConfig) (*ecdsa.PrivateKey, error) {
	if config.PrivateKeyPEM != "" {
		// Load existing private key
		block, _ := pem.Decode([]byte(config.PrivateKeyPEM))
		if block == nil {
			return nil, fmt.Errorf("failed to decode PEM block containing private key")
		}

		privateKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		return privateKey, nil
	}

	// Create new private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// Encode private key to PEM
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	config.PrivateKeyPEM = string(privateKeyPEM)
	return privateKey, nil
}

// saveConfig saves the node configuration to a file
func saveConfig(config *NodeConfig) {
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal config: %v", err)
		return
	}

	// Create data directory if it doesn't exist
	dataDir := "data"
	os.MkdirAll(dataDir, 0755)

	configFile := filepath.Join(dataDir, "config.json")
	err = ioutil.WriteFile(configFile, configData, 0644)
	if err != nil {
		log.Printf("Failed to save config: %v", err)
	}
}

// initializeNode initializes the node based on configuration
func initializeNode(config *NodeConfig, hybridConsensus *consensus.HybridConsensus, p2pNode *network.P2PNode, pohVerify bool, validatorManager *consensus.ValidatorManager) {
	// Get genesis address from blockchain
	genesisAddress := hybridConsensus.GetNodeAddress()
	
	// Initialize genesis address as admin
	if err := validatorManager.InitializeFirstAdmin(genesisAddress); err != nil {
		log.Printf("Failed to initialize genesis address as admin: %v", err)
	} else {
		log.Printf("Genesis address initialized as admin: %s", genesisAddress)
	}
	
	if config.IsValidator {
		if config.HumanProof != "" && pohVerify {
			// Attempt to register as validator with existing proof
			err := validatorManager.RegisterValidator(
				hybridConsensus.GetNodeAddress(),
				"self-registration",
			)
			if err != nil {
				log.Printf("Failed to register as validator: %v", err)
			}
			
			// Also try with consensus engine
			err = hybridConsensus.CompleteHumanVerification(config.HumanProof)
			if err != nil {
				log.Printf("Failed to verify with existing human proof: %v", err)
				initiateHumanVerification(hybridConsensus)
			}
		} else if pohVerify {
			initiateHumanVerification(hybridConsensus)
		}
	}
}

// initiateHumanVerification initiates the human verification process
func initiateHumanVerification(hybridConsensus *consensus.HybridConsensus) {
	proofToken, err := hybridConsensus.InitiateHumanVerification()
	if err != nil {
		log.Printf("Failed to initiate human verification: %v", err)
		return
	}

	fmt.Println("Human verification required to become a validator")
	fmt.Printf("Your proof token is: %s\n", proofToken)
	fmt.Println("Please complete the verification process at [verification URL]")
	fmt.Println("After verification, restart the node with --config flag")
} 