build:
	@echo "Compiling Go code to Wasm using TinyGo..."
	@tinygo build -target=wasi -opt=z -panic=trap -o examples/counter/counter.wasm ./examples/counter
	@ls -lh examples/counter/counter.wasm

opt:
	@echo "Optimizing Wasm using wasm-opt..."
	@wasm-opt -Oz examples/counter/counter.wasm -o examples/counter/counter.opt.wasm
	@ls -lh examples/counter/counter.opt.wasm

compress:
	@echo "Compressing optimized Wasm using Brotli..."
	@brotli -c examples/counter/counter.opt.wasm > examples/counter/counter.wasm.br
	@ls -lh examples/counter/counter.wasm.br

all: build opt compress

check-size:
	@echo "Checking final compressed size..."
	@ls -lh examples/counter/counter.wasm.br
	# Add size check logic if needed, e.g., using awk or stat

test:
	@echo "Running Go tests..."
	@go test ./...

.PHONY: build opt compress all check-size test

