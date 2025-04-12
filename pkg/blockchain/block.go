package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"time"
)

// Block represents a block in the blockchain
type Block struct {
	Index           uint64          `json:"index"`
	Timestamp       int64           `json:"timestamp"`
	Transactions    []*Transaction  `json:"transactions"`
	PrevHash        string          `json:"prevHash"`
	Hash            string          `json:"hash"`
	Validator       string          `json:"validator"`
	HumanProof      string          `json:"humanProof"`
	Reward          uint64          `json:"reward"`
	IsMultisig      bool            `json:"isMultisig"`      // Whether the block was validated by multisig wallet
	Signers         []string        `json:"signers"`         // List of wallets that signed the block
	RequiredSigners int             `json:"requiredSigners"` // Number of required signatures
	Signatures      map[string]string `json:"signatures"`    // Map of signer addresses to their signatures
	Signature       []byte          `json:"signature"`       // Block signature
}

// CalculateHash calculates the hash of the block
func (b *Block) CalculateHash() string {
	record := bytes.Join(
		[][]byte{
			UintToHex(b.Index),
			IntToHex(b.Timestamp),
			[]byte(b.PrevHash),
			[]byte(b.Validator),
			[]byte(b.HumanProof),
			SerializeTransactions(b.Transactions),
		},
		[]byte{},
	)

	h := sha256.New()
	h.Write(record)
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// UintToHex converts a uint64 to a byte slice
func UintToHex(num uint64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		return []byte{}
	}
	return buff.Bytes()
}

// IntToHex converts an int64 to a byte slice
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		return []byte{}
	}
	return buff.Bytes()
}

// SerializeTransactions converts transactions to bytes for hashing
func SerializeTransactions(txs []*Transaction) []byte {
	var txBytes [][]byte
	for _, tx := range txs {
		txJson, err := json.Marshal(tx)
		if err != nil {
			continue
		}
		txBytes = append(txBytes, txJson)
	}
	return bytes.Join(txBytes, []byte{})
}

// NewBlock creates a new block in the blockchain
func NewBlock(index uint64, transactions []*Transaction, prevHash string, validator string, humanProof string) *Block {
	block := &Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		Transactions: transactions,
		PrevHash:     prevHash,
		Validator:    validator,
		HumanProof:   humanProof,
		Reward:       0, // Default reward
	}
	block.Hash = block.CalculateHash()
	return block
}

// Sign signs the block with the given private key
func (b *Block) Sign(privateKey *ecdsa.PrivateKey) error {
	// Create a hash of the block data
	hash := b.CalculateHash()
	
	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, []byte(hash))
	if err != nil {
		return err
	}

	// Combine r and s into a single signature
	signature := append(r.Bytes(), s.Bytes()...)
	b.Signature = signature
	return nil
}

// Verify verifies the block signature
func (b *Block) Verify(publicKey *ecdsa.PublicKey) error {
	if b.Signature == nil || len(b.Signature) == 0 {
		return errors.New("block is not signed")
	}

	// Split signature into r and s components
	r := new(big.Int).SetBytes(b.Signature[:len(b.Signature)/2])
	s := new(big.Int).SetBytes(b.Signature[len(b.Signature)/2:])

	// Create a hash of the block data
	hash := b.CalculateHash()

	// Verify the signature
	valid := ecdsa.Verify(publicKey, []byte(hash), r, s)
	if !valid {
		return errors.New("invalid block signature")
	}

	return nil
}

// GetSignatures returns the block's signatures
func (b *Block) GetSignatures() map[string]string {
	return b.Signatures
} 