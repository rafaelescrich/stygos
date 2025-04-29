//go:build !tinygo

package stygos

// This file provides stub implementations of the host functions for regular Go testing.
// These stubs will be replaced by the mock implementations in host_mock.go when testing.

// read_args stub implementation for regular Go testing
func read_args(ptr *byte) uint32 {
	// This will be replaced by mock_read_args in runtime_mock.go
	return 0
}

// write_result stub implementation for regular Go testing
func write_result(ptr *byte, len uint32) {
	// This will be replaced by mock_write_result in runtime_mock.go
}

// storage_load_bytes32 stub implementation for regular Go testing
func storage_load_bytes32(key_ptr *byte, value_ptr *byte) {
	// This will be replaced by mock_storage_load_bytes32 in runtime_mock.go
}

// storage_store_bytes32 stub implementation for regular Go testing
func storage_store_bytes32(key_ptr *byte, value_ptr *byte) {
	// This will be replaced by mock_storage_store_bytes32 in runtime_mock.go
}

// msg_value stub implementation for regular Go testing
func msg_value(value_ptr *byte) {
	// This will be replaced by mock_msg_value in runtime_mock.go
}

// block_number stub implementation for regular Go testing
func block_number(value_ptr *byte) {
	// This will be replaced by mock_block_number in runtime_mock.go
}

// emit_log stub implementation for regular Go testing
func emit_log(ptr *byte, len uint32, topics_count uint32, topic1_ptr *byte, topic2_ptr *byte, topic3_ptr *byte, topic4_ptr *byte) {
	// This will be replaced by mock_emit_log in runtime_mock.go
}

// native_keccak256 stub implementation for regular Go testing
func native_keccak256(ptr *byte, len uint32, result_ptr *byte) {
	// This will be replaced by mock_native_keccak256 in runtime_mock.go
}

// memory_grow stub implementation for regular Go testing
func memory_grow(pages uint32) {
	// This will be replaced by mock_memory_grow in runtime_mock.go
}
