package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"confirmix/pkg/blockchain"
)

// PeerMessage represents a message in the P2P network
type PeerMessage struct {
	Type    string          `json:"type"`
	From    string          `json:"from"`
	Payload json.RawMessage `json:"payload"`
}

// BlockMessage represents a serialized block
type BlockMessage struct {
	Block *blockchain.Block `json:"block"`
}

// TransactionMessage represents a serialized transaction
type TransactionMessage struct {
	Transaction *blockchain.Transaction `json:"transaction"`
}

// DiscoveryMessage represents peer discovery information
type DiscoveryMessage struct {
	PeerAddresses []string `json:"peer_addresses"`
}

// P2PNode represents a node in the P2P network
type P2PNode struct {
	address       string
	port          int
	peerAddresses map[string]bool
	blockchain    *blockchain.Blockchain
	listener      net.Listener
	peersMutex    sync.RWMutex
	stopChan      chan struct{}
	isRunning     bool
	msgHandlers   map[string]func(from string, payload []byte) error
}

// NewP2PNode creates a new P2P network node
func NewP2PNode(address string, port int, blockchain *blockchain.Blockchain) *P2PNode {
	node := &P2PNode{
		address:       address,
		port:          port,
		peerAddresses: make(map[string]bool),
		blockchain:    blockchain,
		stopChan:      make(chan struct{}),
		isRunning:     false,
		msgHandlers:   make(map[string]func(from string, payload []byte) error),
	}

	// Register default message handlers
	node.RegisterHandler("block", node.handleBlockMessage)
	node.RegisterHandler("transaction", node.handleTransactionMessage)
	node.RegisterHandler("discovery", node.handleDiscoveryMessage)

	return node
}

// Start starts the P2P node
func (node *P2PNode) Start() error {
	// Start listening for incoming connections
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", node.address, node.port))
	if err != nil {
		return fmt.Errorf("failed to start P2P node: %v", err)
	}
	node.listener = listener
	node.isRunning = true

	// Start accepting connections
	go node.acceptConnections()

	// Start peer discovery routine
	go node.discoveryRoutine()

	return nil
}

// Stop stops the P2P node
func (node *P2PNode) Stop() {
	if node.isRunning {
		close(node.stopChan)
		node.listener.Close()
		node.isRunning = false
	}
}

// RegisterHandler registers a message handler
func (node *P2PNode) RegisterHandler(msgType string, handler func(from string, payload []byte) error) {
	node.msgHandlers[msgType] = handler
}

// ConnectToPeer connects to a peer node
func (node *P2PNode) ConnectToPeer(peerAddress string) error {
	node.peersMutex.Lock()
	defer node.peersMutex.Unlock()

	// Skip if already connected
	if node.peerAddresses[peerAddress] {
		return nil
	}

	// Establish connection
	conn, err := net.Dial("tcp", peerAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to peer %s: %v", peerAddress, err)
	}
	defer conn.Close()

	// Add to peer list
	node.peerAddresses[peerAddress] = true

	// Send discovery message to peer
	node.sendDiscoveryMessage(conn)

	return nil
}

// Broadcast sends a message to all peers
func (node *P2PNode) Broadcast(msgType string, payload interface{}) error {
	node.peersMutex.RLock()
	defer node.peersMutex.RUnlock()

	for peerAddr := range node.peerAddresses {
		conn, err := net.Dial("tcp", peerAddr)
		if err != nil {
			log.Printf("Failed to connect to peer %s: %v", peerAddr, err)
			continue
		}

		err = node.sendMessage(conn, msgType, payload)
		conn.Close()
		if err != nil {
			log.Printf("Failed to send message to peer %s: %v", peerAddr, err)
		}
	}

	return nil
}

// BroadcastBlock broadcasts a new block to all peers
func (node *P2PNode) BroadcastBlock(block *blockchain.Block) error {
	blockMsg := BlockMessage{Block: block}
	return node.Broadcast("block", blockMsg)
}

// BroadcastTransaction broadcasts a new transaction to all peers
func (node *P2PNode) BroadcastTransaction(tx *blockchain.Transaction) error {
	txMsg := TransactionMessage{Transaction: tx}
	return node.Broadcast("transaction", txMsg)
}

// acceptConnections accepts incoming connections
func (node *P2PNode) acceptConnections() {
	for {
		select {
		case <-node.stopChan:
			return
		default:
			conn, err := node.listener.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v", err)
				continue
			}

			go node.handleConnection(conn)
		}
	}
}

// handleConnection processes an incoming connection
func (node *P2PNode) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Set read deadline to prevent hanging
	conn.SetReadDeadline(time.Now().Add(time.Minute))

	// Decode message
	var msg PeerMessage
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("Failed to decode message: %v", err)
		return
	}

	// Handle message based on type
	handler, exists := node.msgHandlers[msg.Type]
	if !exists {
		log.Printf("Unknown message type: %s", msg.Type)
		return
	}

	// Process message
	if err := handler(msg.From, msg.Payload); err != nil {
		log.Printf("Error handling message: %v", err)
	}
}

// sendMessage sends a message to a peer
func (node *P2PNode) sendMessage(conn net.Conn, msgType string, payload interface{}) error {
	// Encode payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Create message
	msg := PeerMessage{
		Type:    msgType,
		From:    fmt.Sprintf("%s:%d", node.address, node.port),
		Payload: payloadBytes,
	}

	// Send message
	encoder := json.NewEncoder(conn)
	return encoder.Encode(msg)
}

// sendDiscoveryMessage sends a discovery message to a peer
func (node *P2PNode) sendDiscoveryMessage(conn net.Conn) error {
	// Get all known peers
	node.peersMutex.RLock()
	peerAddresses := make([]string, 0, len(node.peerAddresses))
	for addr := range node.peerAddresses {
		peerAddresses = append(peerAddresses, addr)
	}
	node.peersMutex.RUnlock()

	// Add own address
	ownAddr := fmt.Sprintf("%s:%d", node.address, node.port)
	peerAddresses = append(peerAddresses, ownAddr)

	// Create discovery message
	discoveryMsg := DiscoveryMessage{PeerAddresses: peerAddresses}

	// Send message
	return node.sendMessage(conn, "discovery", discoveryMsg)
}

// discoveryRoutine periodically sends discovery messages to all peers
func (node *P2PNode) discoveryRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-node.stopChan:
			return
		case <-ticker.C:
			node.peersMutex.RLock()
			for peerAddr := range node.peerAddresses {
				conn, err := net.Dial("tcp", peerAddr)
				if err != nil {
					log.Printf("Failed to connect to peer %s: %v", peerAddr, err)
					continue
				}

				node.sendDiscoveryMessage(conn)
				conn.Close()
			}
			node.peersMutex.RUnlock()
		}
	}
}

// handleBlockMessage processes a received block
func (node *P2PNode) handleBlockMessage(from string, payload []byte) error {
	var blockMsg BlockMessage
	if err := json.Unmarshal(payload, &blockMsg); err != nil {
		return fmt.Errorf("failed to unmarshal block message: %v", err)
	}

	// Add block to blockchain
	return node.blockchain.AddBlock(blockMsg.Block)
}

// handleTransactionMessage processes a received transaction
func (node *P2PNode) handleTransactionMessage(from string, payload []byte) error {
	var txMsg TransactionMessage
	if err := json.Unmarshal(payload, &txMsg); err != nil {
		return fmt.Errorf("failed to unmarshal transaction message: %v", err)
	}

	// Add transaction to blockchain
	return node.blockchain.AddTransaction(txMsg.Transaction)
}

// handleDiscoveryMessage processes a received discovery message
func (node *P2PNode) handleDiscoveryMessage(from string, payload []byte) error {
	var discoveryMsg DiscoveryMessage
	if err := json.Unmarshal(payload, &discoveryMsg); err != nil {
		return fmt.Errorf("failed to unmarshal discovery message: %v", err)
	}

	// Connect to new peers
	for _, peerAddr := range discoveryMsg.PeerAddresses {
		ownAddr := fmt.Sprintf("%s:%d", node.address, node.port)
		if peerAddr != ownAddr {
			go node.ConnectToPeer(peerAddr)
		}
	}

	return nil
} 