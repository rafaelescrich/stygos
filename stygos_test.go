package stygos

import (
	"bytes"
	"math/big"
	"testing"

	"golang.org/x/crypto/sha3"
)

func TestStorageRoundTrip(t *testing.T) {
	// Setup mock runtime
	mock := NewMockRuntime()
	UseRuntime(mock)

	// Test key and value
	key := Word{1, 2, 3, 4, 5}
	value := Word{10, 20, 30, 40, 50}

	// Store value
	StorageStore(key, value)

	// Load value
	loadedValue := StorageLoad(key)

	// Verify
	if !bytes.Equal(value[:], loadedValue[:]) {
		t.Errorf("Storage round trip failed. Expected %v, got %v", value, loadedValue)
	}
}

func TestKeccak256(t *testing.T) {
	// Setup mock runtime
	mock := NewMockRuntime()
	UseRuntime(mock)

	// Test data
	data := []byte("test data for keccak")

	// Compute hash
	hash := Keccak256(data)

	// Compute expected hash using the same implementation
	expected := make([]byte, 32)
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	h.Sum(expected[:0])

	// Verify hash matches
	if !bytes.Equal(hash[:], expected) {
		t.Errorf("Keccak256 implementation not working as expected")
	}
}

func TestEmitEvent(t *testing.T) {
	// Setup mock runtime
	mock := NewMockRuntime()
	UseRuntime(mock)

	// Test data
	data := []byte("event data")
	topic1 := Word{1, 2, 3}
	topic2 := Word{4, 5, 6}

	// Emit event
	EmitEvent(data, topic1, topic2)

	// Verify
	if len(mock.Logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(mock.Logs))
	}
}

func TestWordConversions(t *testing.T) {
	// Test uint64 conversion
	value := uint64(123456789)
	word := WordFromUint64(value)
	converted := Uint64FromWord(word)
	if converted != value {
		t.Errorf("uint64 conversion failed. Expected %d, got %d", value, converted)
	}

	// Test big.Int conversion
	bigValue := new(big.Int).SetUint64(987654321)
	word = WordFromBigInt(bigValue)
	convertedBig := BigIntFromWord(word)
	if bigValue.Cmp(convertedBig) != 0 {
		t.Errorf("big.Int conversion failed. Expected %s, got %s", bigValue.String(), convertedBig.String())
	}

	// Test address padding
	var addr Address
	for i := 0; i < 20; i++ {
		addr[i] = byte(i + 1)
	}
	paddedAddr := PadAddress(addr)
	extractedAddr := AddressFromWord(paddedAddr)
	if !bytes.Equal(addr[:], extractedAddr[:]) {
		t.Errorf("Address padding/extraction failed")
	}
}

func TestGetCallData(t *testing.T) {
	// Setup mock runtime
	mock := NewMockRuntime()
	UseRuntime(mock)

	// Set mock call data
	testData := []byte{1, 2, 3, 4, 5}
	mock.Args = testData

	// Get call data
	callData, err := GetCallData()
	if err != nil {
		t.Fatalf("GetCallData failed: %v", err)
	}

	// Verify
	if !bytes.Equal(testData, callData) {
		t.Errorf("GetCallData failed. Expected %v, got %v", testData, callData)
	}
}

func TestSetReturnData(t *testing.T) {
	// Setup mock runtime
	mock := NewMockRuntime()
	UseRuntime(mock)

	// Set return data
	testData := []byte{10, 20, 30, 40, 50}
	SetReturnData(testData)

	// Verify
	if !bytes.Equal(testData, mock.Result) {
		t.Errorf("SetReturnData failed. Expected %v, got %v", testData, mock.Result)
	}
}
