.PHONY: build optimize compress deploy test clean

# Build the contract
build:
	tinygo build -target=wasi -opt=z -panic=trap -o erc20.wasm .

# Optimize the WASM binary
optimize: build
	wasm-opt -Oz erc20.wasm -o erc20.opt.wasm

# Compress the optimized binary
compress: optimize
	brotli -c erc20.opt.wasm > erc20.wasm.br

# Deploy to Arbitrum Stylus testnet
deploy: compress
	cargo stylus deploy --private-key=${PRIVATE_KEY} --wasm-file-path=erc20.wasm.br

# Run tests
test:
	go test -v .

# Clean build artifacts
clean:
	rm -f erc20.wasm erc20.opt.wasm erc20.wasm.br 