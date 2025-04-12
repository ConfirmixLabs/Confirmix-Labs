# PoA-PoH Hybrid Blockchain

## Project Overview

This project implements a hybrid blockchain that combines Proof of Authority (PoA) and Proof of Humanity (PoH) consensus mechanisms using Go. The system enables a secure, efficient blockchain where only verified human validators can participate in block production.

## System Architecture

### High-Level Architecture
```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Layer                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │ Web Client  │  │ CLI Client  │  │ API Client  │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                         API Layer                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │ REST API    │  │ WebSocket   │  │ RPC API     │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Core Layer                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │ Blockchain  │  │ Consensus   │  │ Network     │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Storage Layer                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │ Block Store │  │ State DB    │  │ Cache       │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

### Consensus Flow
```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Validator  │     │  PoA Check  │     │  PoH Check  │
│ Selection   │────▶│  (Authority)│────▶│  (Humanity) │
└─────────────┘     └─────────────┘     └─────────────┘
                           │                   │
                           ▼                   ▼
                    ┌─────────────┐     ┌─────────────┐
                    │  Block      │     │  Human      │
                    │  Production │     │  Verification│
                    └─────────────┘     └─────────────┘
                           │                   │
                           └─────────┬─────────┘
                                     ▼
                              ┌─────────────┐
                              │  Block      │
                              │  Validation │
                              └─────────────┘
```

## Key Components

### 1. Core Blockchain (`pkg/blockchain/`)
- **Block Structure**: Defines the basic block structure with PoA validator information and PoH verification data.
- **Blockchain**: Implements the chain management, including transaction pool, validator registry, and human proof verification.

### 2. Consensus Mechanisms (`pkg/consensus/`)
- **Proof of Authority (PoA)**: Implements a round-robin block production by authorized validators.
- **Proof of Humanity (PoH)**: Implements human verification system to prevent Sybil attacks.
- **Hybrid Consensus**: Combines both mechanisms, requiring validators to be both authorized and verified as humans.

### 3. P2P Networking (`pkg/network/`)
- Implements peer discovery, message broadcasting, and communication between nodes.
- Handles propagation of new blocks and transactions throughout the network.

### 4. Command Line Interface (`cmd/blockchain/`)
- Provides a user interface for node operation, including validator registration and configuration.

## Technical Specifications

### Performance Metrics
- Block Time: 5 seconds
- Transaction Throughput: 1000 TPS
- Block Size: 2MB
- Network Latency: < 100ms
- Validator Count: 5-100 nodes

### Security Features
- ECDSA for digital signatures
- SHA-256 for hashing
- TLS 1.3 for network security
- Rate limiting and DDoS protection
- Input validation and sanitization

### Storage Requirements
- Block Storage: ~1GB per 1M blocks
- State Storage: ~500MB per 1M accounts
- Cache Size: 1GB recommended

## How the Hybrid Consensus Works

1. **Node Registration**: When a node wants to become a validator, it first needs to register and complete the human verification process.

2. **Human Verification**: The node provides proof of humanity, which is verified and stored in the blockchain.

3. **Validator Authorization**: Once verified as human, the node can be authorized as a validator in the PoA system.

4. **Block Production**: Authorized and human-verified validators take turns producing blocks in a round-robin fashion.

5. **Block Validation**: When a new block is received, nodes verify both the PoA signature and the PoH verification of the validator who produced it.

## Benefits of the Hybrid Approach

- **Efficiency**: PoA provides fast block production without expensive computation.
- **Sybil Resistance**: PoH prevents a single entity from controlling multiple validator nodes.
- **Decentralization**: The combination reduces centralization risks that exist in pure PoA systems.
- **Scalability**: The system can scale to many nodes while maintaining performance.

## Usage

### Starting a Node

```bash
./blockchain node --validator=true --poh-verify=true
```

### Node Configuration

Nodes can be configured using the `--config` flag to point to a configuration file, or via command-line parameters:

- `--address`: Node address (default: 127.0.0.1)
- `--port`: Node port (default: 8000)
- `--validator`: Run as a validator (default: false)
- `--poh-verify`: Enable PoH verification (default: false)
- `--peers`: Comma-separated list of peer addresses

## Monitoring and Metrics

### Key Metrics
- Block Production Time
- Transaction Processing Time
- Network Latency
- Validator Performance
- Memory Usage
- CPU Usage
- Disk I/O

### Monitoring Tools
- Prometheus for metrics collection
- Grafana for visualization
- ELK Stack for logging
- Custom monitoring dashboard

## Future Improvements

1. **Enhanced PoH Integration**: Connect to external PoH verification services like BrightID or Proof of Humanity.
2. **Governance System**: Add on-chain governance for validator management.
3. **Smart Contracts**: Add support for smart contract execution.
4. **Performance Optimizations**: Improve transaction throughput and block propagation.
5. **Web Interface**: Create a web-based dashboard for monitoring the blockchain.

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

### Development Guidelines
- Follow Go code style guide
- Write tests for new features
- Update documentation
- Use meaningful commit messages

## License

MIT License - see [LICENSE](LICENSE) for details 