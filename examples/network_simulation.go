package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"sync"
	"time"

	"confirmix/pkg/blockchain"
	"confirmix/pkg/consensus"
	"confirmix/pkg/network"
)

// Node represents a node in the network simulation
type Node struct {
	Address          string
	PrivateKey       *ecdsa.PrivateKey
	Blockchain       *blockchain.Blockchain
	ConsensusEngine  *consensus.HybridConsensus
	P2PNode          *network.P2PNode
	IsValidator      bool
	IsHumanVerified  bool
	Port             int
	HumanProofToken  string
}

func main() {
	fmt.Println("Starting network simulation with multiple nodes")
	
	// Create multiple nodes
	numNodes := 4
	nodes := make([]*Node, numNodes)
	
	// Initialize nodes
	for i := 0; i < numNodes; i++ {
		// Create private key
		privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		
		// Create address from public key
		publicKey := privateKey.PublicKey
		publicKeyBytes := elliptic.Marshal(publicKey.Curve, publicKey.X, publicKey.Y)
		address := fmt.Sprintf("%x", publicKeyBytes[:10])
		
		// Create blockchain
		bc := blockchain.NewBlockchain()
		
		// Determine if node is validator
		isValidator := i < 2 // Make first 2 nodes validators
		
		// Set port (start from 9000)
		port := 9000 + i
		
		// Create node
		nodes[i] = &Node{
			Address:         address,
			PrivateKey:      privateKey,
			Blockchain:      bc,
			IsValidator:     isValidator,
			IsHumanVerified: false,
			Port:            port,
		}
		
		// Create consensus engine
		nodes[i].ConsensusEngine = consensus.NewHybridConsensus(bc, privateKey, address, 5*time.Second)
		
		// Create P2P node
		nodes[i].P2PNode = network.NewP2PNode("127.0.0.1", port, bc)
		
		fmt.Printf("Created node %d: Address=%s, Port=%d, Validator=%v\n", 
			i, address, port, isValidator)
	}
	
	// Start P2P nodes
	fmt.Println("\nStarting P2P nodes...")
	for i, node := range nodes {
		err := node.P2PNode.Start()
		if err != nil {
			log.Fatalf("Failed to start P2P node %d: %v", i, err)
		}
		
		fmt.Printf("Node %d P2P service started on port %d\n", i, node.Port)
	}
	
	// Connect nodes to each other
	fmt.Println("\nConnecting nodes to each other...")
	for i, node := range nodes {
		for j, otherNode := range nodes {
			if i != j {
				peerAddr := fmt.Sprintf("127.0.0.1:%d", otherNode.Port)
				err := node.P2PNode.ConnectToPeer(peerAddr)
				if err != nil {
					log.Printf("Warning: Failed to connect node %d to node %d: %v", i, j, err)
				} else {
					fmt.Printf("Connected node %d to node %d\n", i, j)
				}
			}
		}
	}
	
	// Human verification and validator registration
	fmt.Println("\nPerforming human verification for validators...")
	var validatorAddresses []string
	
	for i, node := range nodes {
		if node.IsValidator {
			// Initiate human verification
			proofToken, err := node.ConsensusEngine.InitiateHumanVerification()
			if err != nil {
				log.Fatalf("Failed to initiate human verification for node %d: %v", i, err)
			}
			
			node.HumanProofToken = proofToken
			
			// Complete human verification
			err = node.ConsensusEngine.CompleteHumanVerification(proofToken)
			if err != nil {
				log.Fatalf("Failed to complete human verification for node %d: %v", i, err)
			}
			
			node.IsHumanVerified = true
			
			// Register as validator
			err = node.ConsensusEngine.RegisterAsValidator()
			if err != nil {
				log.Fatalf("Failed to register node %d as validator: %v", i, err)
			}
			
			validatorAddresses = append(validatorAddresses, node.Address)
			fmt.Printf("Node %d (Address=%s) verified as human and registered as validator\n", 
				i, node.Address)
		}
	}
	
	// Update validator list on all nodes
	fmt.Println("\nUpdating validator lists on all nodes...")
	for i, node := range nodes {
		node.ConsensusEngine.UpdateValidatorList(validatorAddresses)
		fmt.Printf("Updated validator list on node %d\n", i)
	}
	
	// Create transactions from non-validator nodes
	fmt.Println("\nCreating transactions from non-validator nodes...")
	for i, node := range nodes {
		if !node.IsValidator {
			// Create transaction to each validator
			for j, validator := range nodes {
				if validator.IsValidator {
					// Create transaction
					tx := blockchain.NewTransaction(
						node.Address,
						validator.Address,
						float64(i+1) * 10.0, // Different amount for each sender
						[]byte(fmt.Sprintf("Payment from node %d to validator %d", i, j)),
					)
					
					// Add transaction to local blockchain
					err := node.Blockchain.AddTransaction(tx)
					if err != nil {
						log.Printf("Warning: Failed to add transaction to node %d: %v", i, err)
						continue
					}
					
					// Broadcast transaction
					err = node.P2PNode.BroadcastTransaction(tx)
					if err != nil {
						log.Printf("Warning: Failed to broadcast transaction from node %d: %v", i, err)
						continue
					}
					
					fmt.Printf("Node %d created and broadcast transaction to validator %d, amount: %.2f\n", 
						i, j, float64(i+1) * 10.0)
				}
			}
		}
	}
	
	// Start mining on validator nodes
	fmt.Println("\nStarting mining on validator nodes...")
	var wg sync.WaitGroup
	for i, node := range nodes {
		if node.IsValidator {
			wg.Add(1)
			go func(nodeIndex int, n *Node) {
				defer wg.Done()
				
				err := n.ConsensusEngine.StartMining()
				if err != nil {
					log.Printf("Warning: Failed to start mining on node %d: %v", nodeIndex, err)
					return
				}
				
				fmt.Printf("Node %d started mining\n", nodeIndex)
				
				// Mine for some time
				time.Sleep(20 * time.Second)
				
				// Stop mining
				n.ConsensusEngine.StopMining()
				fmt.Printf("Node %d stopped mining\n", nodeIndex)
			}(i, node)
		}
	}
	
	// Wait for mining to complete
	wg.Wait()
	
	// Print final blockchain state on all nodes
	fmt.Println("\nFinal blockchain state on all nodes:")
	for i, node := range nodes {
		fmt.Printf("\nNode %d (Address=%s):\n", i, node.Address)
		fmt.Printf("  Blockchain height: %d\n", node.Blockchain.GetChainHeight())
		
		latestBlock := node.Blockchain.GetLatestBlock()
		fmt.Printf("  Latest block hash: %s\n", latestBlock.Hash)
		fmt.Printf("  Latest block validator: %s\n", latestBlock.Validator)
		fmt.Printf("  Number of transactions in latest block: %d\n", len(latestBlock.Transactions))
		
		if len(latestBlock.Transactions) > 0 {
			fmt.Println("  Transactions in latest block:")
			for j, tx := range latestBlock.Transactions {
				fmt.Printf("    %d. From=%s, To=%s, Amount=%.2f\n", 
					j+1, tx.From, tx.To, tx.Value)
			}
		}
	}
	
	// Verify blockchain consistency across nodes
	fmt.Println("\nVerifying blockchain consistency across nodes...")
	referenceHash := nodes[0].Blockchain.GetLatestBlock().Hash
	consistent := true
	
	for i := 1; i < numNodes; i++ {
		nodeHash := nodes[i].Blockchain.GetLatestBlock().Hash
		if nodeHash != referenceHash {
			fmt.Printf("Inconsistency detected: Node 0 hash = %s, Node %d hash = %s\n", 
				referenceHash, i, nodeHash)
			consistent = false
		}
	}
	
	if consistent {
		fmt.Println("All nodes have consistent blockchain state!")
	} else {
		fmt.Println("WARNING: Blockchain state is not consistent across all nodes!")
	}
	
	// Stop P2P nodes
	fmt.Println("\nStopping P2P nodes...")
	for i, node := range nodes {
		node.P2PNode.Stop()
		fmt.Printf("Node %d P2P service stopped\n", i)
	}
	
	fmt.Println("\nNetwork simulation completed")
} 