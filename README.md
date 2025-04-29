# Stygos: Go SDK for Arbitrum Stylus

Stygos is a Go SDK for writing WebAssembly smart contracts that run on Arbitrum Stylus. It provides a familiar and idiomatic Go interface for interacting with the Stylus environment.

## Features

- Host bindings for Stylus environment functions
- Mock runtime for local testing
- High-level API for common operations
- Example counter contract
- Build tools for compiling, optimizing, and compressing Wasm binaries

## Requirements

- Go 1.18+
- TinyGo 0.30+
- Binaryen (for wasm-opt)
- Brotli (for compression)

## Installation

1. Install Go:
   ```
   sudo apt-get install -y golang-go
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
│   └── counter/           # Example counter contract
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

### Testing

Run the unit tests:
```
go test ./...
```

## License

This project is licensed under the [MIT License](LICENSE).

## Acknowledgments

- [Arbitrum Stylus team](https://arbitrum.io/stylus)
- [TinyGo project](https://tinygo.org)
- [GopherCon LATAM 2025](https://gopherconlatam.org/eng)
