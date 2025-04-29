package stygos

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"
	"unsafe"

	"golang.org/x/crypto/sha3"
)

// MockRuntime provides an in-memory implementation of the Stylus host environment
// for local testing purposes.
type MockRuntime struct {
	Storage map[[32]byte][32]byte // Mock storage: key -> value
	Logs    [][]byte              // Mock event logs
	Args    []byte                // Mock input arguments
	Result  []byte                // Mock execution result
	Value   *big.Int              // Mock msg.value
	Block   uint64                // Mock block number
	mu      sync.Mutex            // Mutex for thread safety
}

// activeRuntime holds the currently active runtime (either real host or mock).
// This is a placeholder; actual wiring will depend on build tags or similar mechanisms.
// For now, we assume mock is always active when not building with TinyGo.
var activeRuntime *MockRuntime

// NewMockRuntime creates a new instance of the mock runtime.
func NewMockRuntime() *MockRuntime {
	return &MockRuntime{
		Storage: make(map[[32]byte][32]byte),
		Logs:    make([][]byte, 0),
		Value:   big.NewInt(0),
		Block:   1, // Start block number at 1
	}
}

// UseRuntime sets the provided MockRuntime as the active runtime for testing.
func UseRuntime(mock *MockRuntime) {
	activeRuntime = mock
}

// --- Mock Implementations of Host Functions ---

// Note: These functions mimic the behavior of the host imports for testing.
// They interact with the MockRuntime state.

func mock_read_args(ptr *byte) uint32 {
	if activeRuntime == nil {
		panic("mock runtime not initialized")
	}
	activeRuntime.mu.Lock()
	defer activeRuntime.mu.Unlock()

	argsLen := len(activeRuntime.Args)
	if argsLen == 0 {
		return 0
	}
	// Unsafe pointer manipulation to copy data into the Wasm memory space (simulated)
	// In a real Go test environment, we'd pass slices directly.
	// This is a simplified representation.
	buf := unsafeSlice(ptr, uint32(argsLen))
	copy(buf, activeRuntime.Args)
	return uint32(argsLen)
}

func mock_write_result(ptr *byte, length uint32) {
	if activeRuntime == nil {
		panic("mock runtime not initialized")
	}
	activeRuntime.mu.Lock()
	defer activeRuntime.mu.Unlock()

	buf := unsafeSlice(ptr, length)
	activeRuntime.Result = make([]byte, length)
	copy(activeRuntime.Result, buf)
}

func mock_storage_load_bytes32(keyPtr, valuePtr *byte) {
	if activeRuntime == nil {
		panic("mock runtime not initialized")
	}
	activeRuntime.mu.Lock()
	defer activeRuntime.mu.Unlock()

	key := *(*[32]byte)(unsafe.Pointer(keyPtr))
	value, exists := activeRuntime.Storage[key]
	if exists {
		valueBuf := unsafeSlice(valuePtr, 32)
		copy(valueBuf, value[:])
	} else {
		// If key doesn't exist, return zero bytes
		valueBuf := unsafeSlice(valuePtr, 32)
		for i := range valueBuf {
			valueBuf[i] = 0
		}
	}
}

func mock_storage_store_bytes32(keyPtr, valuePtr *byte) {
	if activeRuntime == nil {
		panic("mock runtime not initialized")
	}
	activeRuntime.mu.Lock()
	defer activeRuntime.mu.Unlock()

	key := *(*[32]byte)(unsafe.Pointer(keyPtr))
	valueSlice := unsafeSlice(valuePtr, 32)
	var value [32]byte
	copy(value[:], valueSlice)

	// Check if value is zero, if so, delete from storage (EVM behavior)
	isZero := true
	for _, b := range value {
		if b != 0 {
			isZero = false
			break
		}
	}

	if isZero {
		delete(activeRuntime.Storage, key)
	} else {
		activeRuntime.Storage[key] = value
	}
}

func mock_msg_value(valuePtr *byte) {
	if activeRuntime == nil {
		panic("mock runtime not initialized")
	}
	activeRuntime.mu.Lock()
	defer activeRuntime.mu.Unlock()

	valueBuf := unsafeSlice(valuePtr, 32)
	// Clear the buffer first
	for i := range valueBuf {
		valueBuf[i] = 0
	}
	activeRuntime.Value.FillBytes(valueBuf)
}

func mock_block_number(valuePtr *byte) {
	if activeRuntime == nil {
		panic("mock runtime not initialized")
	}
	activeRuntime.mu.Lock()
	defer activeRuntime.mu.Unlock()

	valueBuf := unsafeSlice(valuePtr, 8)
	binary.LittleEndian.PutUint64(valueBuf, activeRuntime.Block)
}

func mock_emit_log(ptr *byte, length uint32, topicsCount uint32, topic1Ptr, topic2Ptr, topic3Ptr, topic4Ptr *byte) {
	if activeRuntime == nil {
		panic("mock runtime not initialized")
	}
	activeRuntime.mu.Lock()
	defer activeRuntime.mu.Unlock()

	logEntry := new(bytes.Buffer)
	logEntry.Write([]byte(fmt.Sprintf("Topics: %d\n", topicsCount)))

	topics := []*byte{topic1Ptr, topic2Ptr, topic3Ptr, topic4Ptr}
	for i := uint32(0); i < topicsCount; i++ {
		if topics[i] != nil {
			topicData := unsafeSlice(topics[i], 32)
			logEntry.Write([]byte(fmt.Sprintf("  Topic %d: %x\n", i+1, topicData)))
		}
	}

	if length > 0 {
		data := unsafeSlice(ptr, length)
		logEntry.Write([]byte(fmt.Sprintf("Data: %x\n", data)))
	}

	activeRuntime.Logs = append(activeRuntime.Logs, logEntry.Bytes())
}

func mock_native_keccak256(ptr *byte, length uint32, resultPtr *byte) {
	// Use real Keccak256 implementation from golang.org/x/crypto/sha3
	resultBuf := unsafeSlice(resultPtr, 32)

	// Clear the result buffer
	for i := range resultBuf {
		resultBuf[i] = 0
	}

	// Compute real Keccak256 hash
	if length > 0 {
		data := unsafeSlice(ptr, length)
		hash := sha3.NewLegacyKeccak256()
		hash.Write(data)
		hash.Sum(resultBuf[:0])
	}
}

func mock_memory_grow(pages uint32) {
	// In a mock environment, memory growth is usually not explicitly simulated
	// unless specific memory limit tests are needed.
	// We can add logging here if required.
	// fmt.Printf("Mock: memory_grow called with %d pages\n", pages)
}

// unsafeSlice creates a Go slice backed by the Wasm memory pointer and length.
// Use with extreme caution, only for interacting with Wasm boundaries.
func unsafeSlice(ptr *byte, length uint32) []byte {
	return unsafe.Slice(ptr, length)
}

// --- Wiring ---
// Depending on build tags (e.g., `tinygo` vs `testing`), the actual host functions
// or these mock functions would be called. This wiring logic is omitted here for brevity
// but would typically involve conditional compilation or an interface.

// Example of how the real functions might be assigned (conceptual):
/*
var (
	ReadArgs             = read_args
	WriteResult          = write_result
	StorageLoadBytes32   = storage_load_bytes32
	StorageStoreBytes32  = storage_store_bytes32
	MsgValue             = msg_value
	BlockNumber          = block_number
	EmitLog              = emit_log
	NativeKeccak256      = native_keccak256
	MemoryGrow           = memory_grow
)

func init() {
	// If built for testing (not tinygo), overwrite with mocks
	// This would typically use build tags like: //go:build !tinygo
	ReadArgs = mock_read_args
	WriteResult = mock_write_result
	StorageLoadBytes32 = mock_storage_load_bytes32
	StorageStoreBytes32 = mock_storage_store_bytes32
	MsgValue = mock_msg_value
	BlockNumber = mock_block_number
	EmitLog = mock_emit_log
	NativeKeccak256 = mock_native_keccak256
	MemoryGrow = mock_memory_grow
}
*/
