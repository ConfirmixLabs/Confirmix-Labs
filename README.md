# ConfirmixLabs

This project is a hybrid blockchain network that combines Proof of Authority (PoA) and Proof of Humanity (PoH) consensus mechanisms, developed using the Go programming language.

## Table of Contents
- [Project Structure](#project-structure)
- [Features](#features)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Project Structure

```
confirmix/
├── main.go            # Main entry point
├── pkg/               # Package directory
│   ├── blockchain/    # Blockchain implementation
│   ├── consensus/     # PoA and PoH consensus mechanisms
│   └── util/          # Utility functions
├── examples/          # Example applications
│   └── web_server.go  # Example web server implementation
├── web/               # Next.js web interface
│   ├── app/           # Web application
│   │   ├── components/# UI components
│   │   └── lib/       # Helper libraries and API services
├── go.mod             # Go module definition
└── README.md          # This file
```

## Features

### Blockchain Core
- [x] Basic blockchain data structure
- [x] Block creation and validation
- [x] Transaction management
- [x] Genesis block creation

### Consensus Mechanism
- [x] PoA (Proof of Authority) implementation
- [x] Validator management
- [x] Human verification integration (PoH)
- [x] Hybrid consensus engine
- [x] Block time adjustment (currently 5 seconds)

### API and Web Interface
- [x] RESTful API
- [x] Modern Next.js web interface
- [x] TypeScript support
- [x] Responsive design with Tailwind CSS
- [x] Transaction creation and monitoring
- [x] Blockchain state visualization

## Installation

### Prerequisites

- Go 1.21 or higher
- Node.js 18 or higher
- npm or yarn
- Git

### Step-by-Step Installation

1. Clone the repository:
```bash
git clone https://github.com/ConfirmixLabs/confirmix.git
cd confirmix
```

2. Install Go dependencies:
```bash
go mod download
```

3. Install Node.js dependencies:
```bash
cd web
npm install
```

## Configuration

### Environment Variables

Create a `.env` file in the root directory with the following variables:

```env
NODE_ADDRESS=127.0.0.1
NODE_PORT=8000
VALIDATOR_MODE=false
POH_VERIFY=false
```

### Node Configuration

You can configure your node using command-line flags or environment variables:

```bash
./blockchain node \
  --address=127.0.0.1 \
  --port=8000 \
  --validator=true \
  --poh-verify=true
```

## Usage

### Starting the Blockchain Node

1. Start a standard node:
```bash
go run main.go
```

2. Start a validator node:
```bash
go run main.go --validator=true --poh-verify=true
```

### Starting the Web Interface

```bash
cd web
npm run dev
```

### Example Usage Scenarios

1. **Creating a Transaction**
```bash
curl -X POST http://localhost:8000/api/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "from": "0x123...",
    "to": "0x456...",
    "amount": 100,
    "signature": "0x789..."
  }'
```

2. **Checking Blockchain Status**
```bash
curl http://localhost:8000/api/status
```

3. **Registering as a Validator**
```bash
curl -X POST http://localhost:8000/api/validators/register \
  -H "Content-Type: application/json" \
  -d '{
    "address": "0x123...",
    "humanProof": "your-proof-token"
  }'
```

## Troubleshooting

### Common Issues

1. **Node Fails to Start**
   - Check if the port is already in use
   - Verify all required environment variables are set
   - Ensure you have proper permissions

2. **Web Interface Connection Issues**
   - Verify the blockchain node is running
   - Check if the API endpoint is correct
   - Ensure CORS is properly configured

3. **Validator Registration Fails**
   - Verify your human proof token is valid
   - Check if you have sufficient permissions
   - Ensure your node is properly configured as a validator

### Logs

Logs are stored in the `logs` directory. You can check them for detailed error information:

```bash
tail -f logs/blockchain.log
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow the Go code style guide
- Write tests for new features
- Update documentation when adding new features
- Use meaningful commit messages

## License

MIT License - see [LICENSE](LICENSE) for details 