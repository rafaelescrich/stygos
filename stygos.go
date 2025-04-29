package stygos

import (
	"encoding/binary"
	"errors"
	"math/big"
)

// Type definitions for common Ethereum types
type Word [32]byte    // 32-byte word (bytes32 in Solidity)
type Address [20]byte // 20-byte Ethereum address

// Error definitions
var (
	ErrInvalidLength = errors.New("invalid length")
	ErrInvalidInput  = errors.New("invalid input")
	ErrMemoryLimit   = errors.New("memory limit exceeded")
)

// Constants
const (
	MaxCallDataSize = 1024 * 1024 // 1MB limit
	MaxTopics       = 4
)

// Function pointers for host functions
var (
	ReadArgs            func(ptr *byte) uint32
	WriteResult         func(ptr *byte, len uint32)
	StorageLoadBytes32  func(key_ptr *byte, value_ptr *byte)
	StorageStoreBytes32 func(key_ptr *byte, value_ptr *byte)
	MsgValue            func(value_ptr *byte)
	BlockNumber         func(value_ptr *byte)
	EmitLog             func(ptr *byte, len uint32, topics_count uint32, topic1_ptr *byte, topic2_ptr *byte, topic3_ptr *byte, topic4_ptr *byte)
	NativeKeccak256     func(ptr *byte, len uint32, result_ptr *byte)
	MemoryGrow          func(pages uint32)
)

// --- High-level API wrappers ---

// GetCallData returns the input data for the current call
func GetCallData() ([]byte, error) {
	// First call with nil to get the length
	var dummyByte byte
	length := ReadArgs(&dummyByte)
	if length == 0 {
		return []byte{}, nil
	}

	// Validate length
	if length > MaxCallDataSize {
		return nil, ErrMemoryLimit
	}

	// Allocate buffer and read the actual data
	data := make([]byte, length)
	ReadArgs(&data[0])
	return data, nil
}

// SetReturnData sets the return data for the current call
func SetReturnData(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if len(data) > MaxCallDataSize {
		return ErrMemoryLimit
	}
	WriteResult(&data[0], uint32(len(data)))
	return nil
}

// StorageLoad loads a 32-byte word from storage using a 32-byte key
func StorageLoad(key Word) Word {
	var value Word
	StorageLoadBytes32(&key[0], &value[0])
	return value
}

// StorageStore stores a 32-byte word to storage using a 32-byte key
func StorageStore(key, value Word) {
	StorageStoreBytes32(&key[0], &value[0])
}

// GetMsgValue returns the ETH value sent with the transaction as a big.Int
func GetMsgValue() *big.Int {
	var valueBytes Word
	MsgValue(&valueBytes[0])
	return new(big.Int).SetBytes(valueBytes[:])
}

// GetBlockNumber returns the current block number
func GetBlockNumber() uint64 {
	var blockNum [8]byte
	BlockNumber(&blockNum[0])
	return binary.LittleEndian.Uint64(blockNum[:])
}

// Keccak256 computes the Keccak256 hash of the input data
func Keccak256(data []byte) Word {
	var result Word
	if len(data) == 0 {
		return result
	}
	NativeKeccak256(&data[0], uint32(len(data)), &result[0])
	return result
}

// EmitEvent emits an EVM log with the given topics and data
func EmitEvent(data []byte, topics ...Word) error {
	if len(topics) > MaxTopics {
		return ErrInvalidInput
	}

	var topicPtrs [4]*byte
	topicsCount := uint32(len(topics))

	for i := uint32(0); i < topicsCount; i++ {
		topicPtrs[i] = &topics[i][0]
	}

	var dataPtr *byte
	dataLen := uint32(0)
	if len(data) > 0 {
		if len(data) > MaxCallDataSize {
			return ErrMemoryLimit
		}
		dataPtr = &data[0]
		dataLen = uint32(len(data))
	}

	EmitLog(dataPtr, dataLen, topicsCount, topicPtrs[0], topicPtrs[1], topicPtrs[2], topicPtrs[3])
	return nil
}

// --- Utility functions ---

// PadAddress pads an Ethereum address to a full 32-byte word
func PadAddress(addr Address) Word {
	var result Word
	copy(result[12:], addr[:])
	return result
}

// AddressFromWord extracts an Ethereum address from a 32-byte word
func AddressFromWord(word Word) Address {
	var addr Address
	copy(addr[:], word[12:])
	return addr
}

// WordFromUint64 creates a 32-byte word from a uint64 value
func WordFromUint64(value uint64) Word {
	var result Word
	binary.BigEndian.PutUint64(result[24:], value)
	return result
}

// Uint64FromWord extracts a uint64 from a 32-byte word
func Uint64FromWord(word Word) uint64 {
	return binary.BigEndian.Uint64(word[24:])
}

// WordFromBigInt creates a 32-byte word from a big.Int value
func WordFromBigInt(value *big.Int) Word {
	var result Word
	bytes := value.Bytes()
	if len(bytes) > 32 {
		bytes = bytes[len(bytes)-32:]
	}
	// Big-endian encoding, right-aligned
	copy(result[32-len(bytes):], bytes)
	return result
}

// BigIntFromWord creates a big.Int from a 32-byte word
func BigIntFromWord(word Word) *big.Int {
	return new(big.Int).SetBytes(word[:])
}

// --- Memory management helpers ---

// GrowMemory requests additional memory from the host
// Each page is 64KiB (65536 bytes)
func GrowMemory(additionalPages uint32) error {
	if additionalPages == 0 {
		return nil
	}
	MemoryGrow(additionalPages)
	return nil
}

// EnsureMemory ensures that enough memory is available
// It grows memory if needed to accommodate the requested size
func EnsureMemory(sizeBytes uint32) error {
	if sizeBytes == 0 {
		return nil
	}
	// Calculate how many 64KiB pages are needed
	const pageSize uint32 = 65536
	pagesNeeded := (sizeBytes + pageSize - 1) / pageSize
	if pagesNeeded > 0 {
		return GrowMemory(pagesNeeded)
	}
	return nil
}
