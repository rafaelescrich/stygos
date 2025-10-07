# Stygos: Go SDK for Arbitrum Stylus

Stygos is a Go SDK for writing WebAssembly smart contracts that run on Arbitrum Stylus. It provides a familiar and idiomatic Go interface for interacting with the Stylus environment.

## Features

- Host bindings for Stylus environment functions
- Mock runtime for local testing
- High-level API for common operations
- **Schnorr BIP-340 signature verification** (much faster than Solidity)
- **Comprehensive example contracts**:
  - Counter contract
  - ERC20 token
  - Multisig wallet with Schnorr signatures
  - Voting/governance system
  - NFT contract
- Build tools for compiling, optimizing, and compressing Wasm binaries

## Requirements

- Go 1.18+
- TinyGo 0.30+
- Binaryen (for wasm-opt)
- Brotli (for compression)

## Installation

1. Install Go:
   ```
   # Download Go from the official website
   wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
   sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
   echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
   source ~/.bashrc
   ```

2. Install TinyGo:
   ```
   wget https://github.com/tinygo-org/tinygo/releases/download/v0.37.0/tinygo_0.37.0_amd64.deb
   sudo dpkg -i tinygo_0.37.0_amd64.deb
   export PATH=$PATH:/usr/local/bin
   ```

3. Install Binaryen and Brotli:
   ```
   sudo apt-get install -y binaryen brotli
   ```

4. Install goimports (optional):
   ```
   go install golang.org/x/tools/cmd/goimports@latest
   ```

## Project Structure

```
stygos/
├── host_import_tinygo.go  # Host function imports for TinyGo builds
├── host_import_go.go      # Host function stubs for regular Go builds
├── host_mock.go           # Mock implementation for testing
├── runtime_mock.go        # Wiring for mock runtime
├── stygos.go              # Core API
├── stygos_test.go         # Unit tests
├── Makefile               # Build automation
├── examples/
│   ├── counter/           # Simple counter contract
│   ├── erc20/             # ERC20 token implementation
│   ├── schnorr/           # Schnorr BIP-340 signature verification
│   ├── multisig/          # Multisig wallet with Schnorr signatures
│   ├── voting/            # Governance voting system
│   └── nft/               # NFT contract implementation
└── cmd/
    └── stygos/            # CLI tool (future)
```

## Usage

### Writing a Contract

Here's a simple counter contract example:

```go
package main

import (
    "encoding/binary"
    "github.com/rafaelescrich/stygos"
)

// Storage keys
var (
    counterKey = stygos.Keccak256([]byte("counter"))
)

// Commands
const (
    CMD_GET      = 0
    CMD_INCREMENT = 1
    CMD_DECREMENT = 2
    CMD_RESET     = 3
)

//export entrypoint
func entrypoint() int32 {
    // Get the call data
    callData := stygos.GetCallData()
    
    // Default to GET if no command is provided
    command := CMD_GET
    if len(callData) >= 1 {
        command = int(callData[0])
    }
    
    // Get the current counter value
    counterValue := getCounter()
    
    // Process the command
    switch command {
    case CMD_INCREMENT:
        counterValue++
        setCounter(counterValue)
    case CMD_DECREMENT:
        if counterValue > 0 {
            counterValue--
        }
        setCounter(counterValue)
    case CMD_RESET:
        counterValue = 0
        setCounter(counterValue)
    }
    
    // Return the current counter value
    result := make([]byte, 4)
    binary.BigEndian.PutUint32(result, counterValue)
    stygos.SetReturnData(result)
    
    return 0 // Success
}

// getCounter retrieves the current counter value from storage
func getCounter() uint32 {
    valueWord := stygos.StorageLoad(counterKey)
    return binary.BigEndian.Uint32(valueWord[28:32])
}

// setCounter stores the counter value in storage
func setCounter(value uint32) {
    var valueWord stygos.Word
    binary.BigEndian.PutUint32(valueWord[28:32], value)
    stygos.StorageStore(counterKey, valueWord)
}
```

### Building and Deploying

1. Using Docker (recommended):
   ```bash
   # Build the Docker image
   docker build -t stygos .

   # Run tests
   docker run stygos make test

   # Deploy to Stylus testnet (pass private key as environment variable)
   docker run -e PRIVATE_KEY=your_private_key_here stygos make deploy
   ```

2. Manual build:
   ```bash
   # Compile with TinyGo
   tinygo build -target=wasi -opt=z -panic=trap -o counter.wasm ./examples/counter

   # Optimize with wasm-opt
   wasm-opt -Oz counter.wasm -o counter.opt.wasm

   # Compress with Brotli
   brotli -c counter.opt.wasm > counter.wasm.br

   # Deploy to Stylus testnet
   cargo stylus deploy --private-key=YOUR_TESTNET_PRIVKEY --wasm-file-path=counter.wasm.br
   ```

### Schnorr BIP-340 Signature Verification

Stygos includes a high-performance Go implementation of Schnorr BIP-340 signature verification, which is significantly faster than the equivalent Solidity implementation:

```go
package main

import (
    "github.com/rafaelescrich/stygos"
)

//export entrypoint
func entrypoint() int32 {
    callData, err := stygos.GetCallData()
    if err != nil || len(callData) < 1 {
        return 1
    }

    command := callData[0]
    args := callData[1:]

    switch command {
    case CMD_VERIFY:
        return handleVerify(args)
    case CMD_ADAPTOR_VERIFY:
        return handleAdaptorVerify(args)
    case CMD_EXTRACT:
        return handleExtract(args)
    // ... other commands
    }
    return 0
}

func handleVerify(args []byte) int32 {
    // Parse message, signature, and public key
    msgLen := int(args[0])
    msg := args[1 : 1+msgLen]
    pkX := args[1+msgLen : 1+msgLen+32]
    sig := args[1+msgLen+32 : 1+msgLen+32+64]
    
    valid := verify(msg, sig, pkX)
    if valid {
        return 0
    }
    return 1
}
```

**Key Features:**
- **BIP-340 compliant** Schnorr signature verification
- **Adaptor signatures** for atomic swaps and payment channels
- **Secret extraction** from adaptor signatures
- **Point operations** (addition, multiplication, lifting)
- **Much faster** than Solidity EC arithmetic

### Example Contracts

#### Multisig Wallet
A multisig wallet implementation using Schnorr signatures for approvals:

```go
// Commands
const (
    CMD_INITIALIZE     = 0
    CMD_SUBMIT_PROPOSAL = 1
    CMD_APPROVE_PROPOSAL = 2
    CMD_EXECUTE_PROPOSAL = 3
)

func handleApproveProposal(args []byte) int32 {
    // Parse proposal nonce and Schnorr signature
    nonce := binary.BigEndian.Uint32(args[:4])
    sig := args[4:68] // 64-byte Schnorr signature
    
    // Verify signature and store approval
    // ... implementation details
}
```

#### Voting System
A governance voting system with configurable quorum and voting periods:

```go
func handleVote(args []byte) int32 {
    proposalId := binary.BigEndian.Uint64(args[:8])
    voteType := args[8] // FOR, AGAINST, or ABSTAIN
    
    // Cast vote and update proposal state
    // ... implementation details
}
```

#### NFT Contract
A complete NFT implementation with metadata support:

```go
func handleMint(args []byte) int32 {
    var to stygos.Address
    copy(to[:], args[:20])
    
    // Mint new NFT and update balances
    // ... implementation details
}
```

### Testing

Run the unit tests:
```
go test ./...
```

Run tests for specific examples:
```
go test ./examples/schnorr/...
go test ./examples/multisig/...
go test ./examples/voting/...
go test ./examples/nft/...
```

## License

This project is licensed under the [MIT License](LICENSE).

## Acknowledgments

- [Arbitrum Stylus team](https://arbitrum.io/stylus)
- [TinyGo project](https://tinygo.org)
- [GopherCon LATAM 2025](https://gopherconlatam.org/eng)
