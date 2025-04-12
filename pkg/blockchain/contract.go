package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// ContractState represents the state of a smart contract
type ContractState map[string]interface{}

// Contract represents a smart contract in the blockchain
type Contract struct {
	Address  string        `json:"address"`
	Code     string        `json:"code"`
	Creator  string        `json:"creator"`
	State    ContractState `json:"state"`
	Deployed bool          `json:"deployed"`
}

// ContractFunction represents a callable function in a smart contract
type ContractFunction struct {
	Name       string   `json:"name"`
	Parameters []string `json:"parameters"`
	Code       string   `json:"code"`
}

// ContractManager manages smart contracts in the blockchain
type ContractManager struct {
	contracts map[string]*Contract
	mutex     sync.RWMutex
}

// NewContractManager creates a new contract manager
func NewContractManager() *ContractManager {
	return &ContractManager{
		contracts: make(map[string]*Contract),
	}
}

// DeployContract deploys a new contract to the blockchain
func (cm *ContractManager) DeployContract(code string, creator string) (string, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	// Generate a contract address based on creator and timestamp
	contractAddress := fmt.Sprintf("contract-%s-%d", creator[:8], GetTimestamp())
	
	// Create a new contract
	contract := &Contract{
		Address:  contractAddress,
		Code:     code,
		Creator:  creator,
		State:    make(ContractState),
		Deployed: true,
	}
	
	// Store the contract
	cm.contracts[contractAddress] = contract
	
	return contractAddress, nil
}

// GetContract returns a contract by its address
func (cm *ContractManager) GetContract(address string) (*Contract, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	contract, exists := cm.contracts[address]
	if !exists {
		return nil, errors.New("contract not found")
	}
	
	return contract, nil
}

// CallContract calls a function on a contract with the given parameters
func (cm *ContractManager) CallContract(contractAddress string, function string, params []interface{}, caller string) (interface{}, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	// Get the contract
	contract, exists := cm.contracts[contractAddress]
	if !exists {
		return nil, errors.New("contract not found")
	}
	
	if !contract.Deployed {
		return nil, errors.New("contract not deployed")
	}
	
	// In a real implementation, this would parse and execute the contract code
	// For this demo, we'll just update the state based on the function name
	
	// Simple example implementation for a token contract
	switch function {
	case "transfer":
		if len(params) < 2 {
			return nil, errors.New("transfer requires recipient and amount parameters")
		}
		
		recipient, ok := params[0].(string)
		if !ok {
			return nil, errors.New("recipient must be a string")
		}
		
		amount, ok := params[1].(float64)
		if !ok {
			return nil, errors.New("amount must be a number")
		}
		
		// Get balances from state
		callerBalance, ok := contract.State[caller].(float64)
		if !ok {
			callerBalance = 0
		}
		
		recipientBalance, ok := contract.State[recipient].(float64)
		if !ok {
			recipientBalance = 0
		}
		
		// Check if caller has enough balance
		if callerBalance < amount {
			return nil, errors.New("insufficient balance")
		}
		
		// Update balances
		contract.State[caller] = callerBalance - amount
		contract.State[recipient] = recipientBalance + amount
		
		return true, nil
		
	case "balanceOf":
		if len(params) < 1 {
			return nil, errors.New("balanceOf requires account parameter")
		}
		
		account, ok := params[0].(string)
		if !ok {
			return nil, errors.New("account must be a string")
		}
		
		balance, ok := contract.State[account].(float64)
		if !ok {
			balance = 0
		}
		
		return balance, nil
		
	case "mint":
		if caller != contract.Creator {
			return nil, errors.New("only creator can mint")
		}
		
		if len(params) < 2 {
			return nil, errors.New("mint requires recipient and amount parameters")
		}
		
		recipient, ok := params[0].(string)
		if !ok {
			return nil, errors.New("recipient must be a string")
		}
		
		amount, ok := params[1].(float64)
		if !ok {
			return nil, errors.New("amount must be a number")
		}
		
		// Get recipient balance
		recipientBalance, ok := contract.State[recipient].(float64)
		if !ok {
			recipientBalance = 0
		}
		
		// Update balance
		contract.State[recipient] = recipientBalance + amount
		
		return true, nil
		
	default:
		return nil, fmt.Errorf("unknown function: %s", function)
	}
}

// GetAllContracts returns all deployed contracts
func (cm *ContractManager) GetAllContracts() []*Contract {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	contracts := make([]*Contract, 0, len(cm.contracts))
	for _, contract := range cm.contracts {
		if contract.Deployed {
			contracts = append(contracts, contract)
		}
	}
	
	return contracts
}

// SerializeContract serializes a contract to JSON
func SerializeContract(contract *Contract) ([]byte, error) {
	return json.Marshal(contract)
}

// DeserializeContract deserializes a contract from JSON
func DeserializeContract(data []byte) (*Contract, error) {
	contract := &Contract{}
	err := json.Unmarshal(data, contract)
	return contract, err
}

// GetTimestamp gets the current timestamp
func GetTimestamp() int64 {
	return CurrentTimestamp()
} 