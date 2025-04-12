# Confirmix Labs Blockchain

## Overview

This project is a hybrid blockchain built using the Go programming language that combines Proof of Authority (PoA) and Proof of Humanity (PoH) consensus mechanisms. The system provides a secure and efficient blockchain infrastructure where only authenticated human validators can participate in block production.

## Project Structure

The project is developed with a modular architecture, with the main directories being:

```
./
├── cmd/                # Command line application and entry point
├── pkg/                # Main library code
│   ├── blockchain/     # Core blockchain structures
│   ├── consensus/      # PoA and PoH consensus mechanisms
│   ├── network/        # P2P network functions
│   ├── validator/      # Validator management
│   └── util/           # Helper functions
├── examples/           # Example applications
│   ├── web_server.go   # Example web server implementation
│   ├── basic.go        # Basic blockchain functions
│   ├── contract.go     # Smart contract example
│   └── cli_client.go   # Command line client
│   └── network_simulation.go # P2P network simulation
├── cmd/                # Command line application
│   └── blockchain/     # Blockchain node application
├── web/                # Web interface
│   ├── app/            # Next.js application directory
│   ├── public/         # Static files
└── test/               # Test files
```

## Core Components

### 1. Blockchain Core (`pkg/blockchain/`)

- **`block.go`**: Defines block structure and functions
  - Hash calculation
  - Block validation
  - Signature management
  - Human verification token

- **`blockchain.go`**: Contains chain structure and basic functions
  - Blockchain management
  - Transaction pool
  - Validator registration
  - Genesis block creation

- **`transaction.go`**: Defines transaction structure and validation mechanisms
  - Transaction creation and signing
  - Transaction validation
  - Regular and smart contract transactions

### 2. Consensus Mechanisms (`pkg/consensus/`)

- **`poa.go`**: Proof of Authority consensus algorithm
  - Authorized validators
  - Round-robin block production
  - Block signing and validation

- **`poh.go`**: Proof of Humanity verification system
  - Human verification record
  - Verification tokens
  - Verification time management

- **`hybrid.go`**: Hybrid consensus engine that combines PoA and PoH
  - Validator selection
  - Block production
  - PoA+PoH validation

- **`poh_external.go`**: Integration with external human verification systems
  - Connection to services like BrightID or Proof of Humanity
  - External verification protocols

## API Documentation

### RESTful API Endpoints

#### Blockchain Status
```
GET /api/status
```
Returns the current status of the blockchain including:
- Current block height
- Number of validators
- Network status
- Consensus state

Response:
```json
{
  "blockHeight": 1234,
  "validatorCount": 5,
  "networkStatus": "active",
  "consensusState": "hybrid",
  "lastBlockTime": "2024-03-26T12:00:00Z"
}
```

#### Transaction Management
```
POST /api/transactions
```
Create a new transaction.

Request:
```json
{
  "from": "0x123...",
  "to": "0x456...",
  "amount": 100,
  "signature": "0x789..."
}
```

Response:
```json
{
  "transactionId": "0xabc...",
  "status": "pending",
  "timestamp": "2024-03-26T12:00:00Z"
}
```

#### Validator Management
```
POST /api/validators/register
```
Register as a validator.

Request:
```json
{
  "address": "0x123...",
  "humanProof": "your-proof-token",
  "stake": 1000
}
```

Response:
```json
{
  "validatorId": "0xdef...",
  "status": "pending",
  "verificationRequired": true
}
```

## Developer Guide

### Setting Up Development Environment

1. Install Go 1.21 or higher
2. Install Node.js 18 or higher
3. Clone the repository
4. Install dependencies:
```bash
go mod download
cd web && npm install
```

### Code Style Guide

1. **Go Code Style**
   - Use `gofmt` for formatting
   - Follow Go standard naming conventions
   - Write tests for all new features
   - Document exported functions and types

2. **JavaScript/TypeScript Code Style**
   - Use ESLint with recommended settings
   - Follow TypeScript best practices
   - Write unit tests using Jest
   - Document components and functions

### Testing

1. **Go Tests**
```bash
go test ./...
```

2. **Web Tests**
```bash
cd web
npm test
```

### Security Best Practices

1. **Key Management**
   - Never commit private keys to the repository
   - Use environment variables for sensitive data
   - Implement proper key rotation policies

2. **API Security**
   - Implement rate limiting
   - Use HTTPS for all API endpoints
   - Validate all input data
   - Implement proper authentication and authorization

3. **Smart Contract Security**
   - Follow the checks-effects-interactions pattern
   - Implement proper access control
   - Use safe math operations
   - Test for common vulnerabilities

## Performance Optimization

### Block Production
- Optimize block validation
- Implement efficient transaction processing
- Use caching where appropriate

### Network Layer
- Implement efficient peer discovery
- Optimize block propagation
- Use compression for large payloads

### Storage
- Implement efficient data structures
- Use proper indexing
- Implement caching strategies

## Monitoring and Logging

### Logging Levels
- ERROR: Critical issues requiring immediate attention
- WARN: Potential issues that need monitoring
- INFO: General operational information
- DEBUG: Detailed information for troubleshooting

### Metrics
- Block production time
- Transaction processing time
- Network latency
- Validator performance

## Deployment

### Docker Support
```bash
docker build -t confirmix .
docker run -p 8000:8000 confirmix
```

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: confirmix
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: confirmix
        image: confirmix:latest
        ports:
        - containerPort: 8000
```

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

### Pull Request Guidelines
- Include tests for new features
- Update documentation
- Follow the code style guide
- Provide clear commit messages

## License

MIT License - see [LICENSE](LICENSE) for details
``` 