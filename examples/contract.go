package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"confirmix/pkg/blockchain"
	"confirmix/pkg/consensus"
)

// Simple token contract code
const tokenContractCode = `
// ERC-20 style token contract
contract Token {
    string public name = "HybridToken";
    string public symbol = "HBT";
    uint8 public decimals = 18;
    uint256 public totalSupply = 0;
    mapping(address => uint256) public balanceOf;
    
    // Constructor
    constructor() {
        // Creator gets initial supply
        mint(msg.sender, 1000000);
    }
    
    // Mint new tokens (only owner)
    function mint(address to, uint256 amount) public {
        require(msg.sender == owner, "Only owner can mint");
        balanceOf[to] += amount;
        totalSupply += amount;
    }
    
    // Transfer tokens
    function transfer(address to, uint256 amount) public {
        require(balanceOf[msg.sender] >= amount, "Insufficient balance");
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
    }
    
    // Get balance
    function balanceOf(address account) public view returns (uint256) {
        return balanceOf[account];
    }
}
`

func main() {
	// Create a new blockchain
	bc := blockchain.NewBlockchain()
	fmt.Println("Created new blockchain")
	fmt.Printf("Genesis block hash: %s\n", bc.GetLatestBlock().Hash)

	// Create private keys for users
	ownerKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	validatorKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	user1Key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	user2Key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Create addresses from public keys
	ownerPubKey := ownerKey.PublicKey
	ownerPubKeyBytes := elliptic.Marshal(ownerPubKey.Curve, ownerPubKey.X, ownerPubKey.Y)
	ownerAddress := fmt.Sprintf("%x", ownerPubKeyBytes[:10])

	validatorPubKey := validatorKey.PublicKey
	validatorPubKeyBytes := elliptic.Marshal(validatorPubKey.Curve, validatorPubKey.X, validatorPubKey.Y)
	validatorAddress := fmt.Sprintf("%x", validatorPubKeyBytes[:10])

	user1PubKey := user1Key.PublicKey
	user1PubKeyBytes := elliptic.Marshal(user1PubKey.Curve, user1PubKey.X, user1PubKey.Y)
	user1Address := fmt.Sprintf("%x", user1PubKeyBytes[:10])

	user2PubKey := user2Key.PublicKey
	user2PubKeyBytes := elliptic.Marshal(user2PubKey.Curve, user2PubKey.X, user2PubKey.Y)
	user2Address := fmt.Sprintf("%x", user2PubKeyBytes[:10])

	fmt.Printf("Owner address: %s\n", ownerAddress)
	fmt.Printf("Validator address: %s\n", validatorAddress)
	fmt.Printf("User1 address: %s\n", user1Address)
	fmt.Printf("User2 address: %s\n", user2Address)

	// Create consensus engine for validator
	validatorConsensus := consensus.NewHybridConsensus(bc, validatorKey, validatorAddress, 5*time.Second)

	// Setup validator
	proofToken, err := validatorConsensus.InitiateHumanVerification()
	if err != nil {
		log.Fatalf("Failed to initiate human verification: %v", err)
	}

	err = validatorConsensus.CompleteHumanVerification(proofToken)
	if err != nil {
		log.Fatalf("Failed to complete human verification: %v", err)
	}

	err = validatorConsensus.RegisterAsValidator()
	if err != nil {
		log.Fatalf("Failed to register validator: %v", err)
	}

	validatorConsensus.UpdateValidatorList([]string{validatorAddress})
	fmt.Println("Validator registered and ready")

	// Create a contract deployment transaction
	deployTx, err := blockchain.NewContractDeploymentTransaction(ownerAddress, tokenContractCode)
	if err != nil {
		log.Fatalf("Failed to create deployment transaction: %v", err)
	}

	// Add transaction to blockchain
	err = bc.AddTransaction(deployTx)
	if err != nil {
		log.Fatalf("Failed to add deployment transaction: %v", err)
	}

	fmt.Println("Contract deployment transaction added")

	// Start mining
	err = validatorConsensus.StartMining()
	if err != nil {
		log.Fatalf("Failed to start mining: %v", err)
	}

	// Wait for block to be mined
	time.Sleep(10 * time.Second)
	validatorConsensus.StopMining()

	// Get contract address (in a real system this would be returned by deployment)
	contractManager := bc.GetContractManager()
	contracts := contractManager.GetAllContracts()
	if len(contracts) == 0 {
		log.Fatalf("No contracts deployed")
	}

	contractAddress := contracts[0].Address
	fmt.Printf("Contract deployed at address: %s\n", contractAddress)

	// Get owner balance
	balanceCheckTx, err := blockchain.NewContractCallTransaction(
		ownerAddress,
		contractAddress,
		"balanceOf",
		[]interface{}{ownerAddress},
	)
	if err != nil {
		log.Fatalf("Failed to create balance check transaction: %v", err)
	}

	// Add transaction to blockchain
	err = bc.AddTransaction(balanceCheckTx)
	if err != nil {
		log.Fatalf("Failed to add balance check transaction: %v", err)
	}

	// Create a transfer transaction
	transferTx, err := blockchain.NewContractCallTransaction(
		ownerAddress,
		contractAddress,
		"transfer",
		[]interface{}{user1Address, 100.0},
	)
	if err != nil {
		log.Fatalf("Failed to create transfer transaction: %v", err)
	}

	// Add transaction to blockchain
	err = bc.AddTransaction(transferTx)
	if err != nil {
		log.Fatalf("Failed to add transfer transaction: %v", err)
	}

	fmt.Println("Transfer transaction added")

	// Start mining again to process these transactions
	err = validatorConsensus.StartMining()
	if err != nil {
		log.Fatalf("Failed to start mining: %v", err)
	}

	// Wait for block to be mined
	time.Sleep(10 * time.Second)
	validatorConsensus.StopMining()

	// Check user1 balance
	user1BalanceCheckTx, err := blockchain.NewContractCallTransaction(
		user1Address,
		contractAddress,
		"balanceOf",
		[]interface{}{user1Address},
	)
	if err != nil {
		log.Fatalf("Failed to create user1 balance check transaction: %v", err)
	}

	// Add transaction to blockchain
	err = bc.AddTransaction(user1BalanceCheckTx)
	if err != nil {
		log.Fatalf("Failed to add user1 balance check transaction: %v", err)
	}

	// Start mining one more time
	err = validatorConsensus.StartMining()
	if err != nil {
		log.Fatalf("Failed to start mining: %v", err)
	}

	// Wait for block to be mined
	time.Sleep(10 * time.Second)
	validatorConsensus.StopMining()

	// Print blockchain state
	fmt.Printf("Blockchain height: %d\n", bc.GetChainHeight())
	latestBlock := bc.GetLatestBlock()
	fmt.Printf("Latest block hash: %s\n", latestBlock.Hash)

	// Try to get contract directly
	contract, err := contractManager.GetContract(contractAddress)
	if err != nil {
		log.Printf("Warning: Failed to get contract: %v", err)
	} else {
		fmt.Println("Contract State:")
		for key, value := range contract.State {
			fmt.Printf("%s: %v\n", key, value)
		}
	}
} 