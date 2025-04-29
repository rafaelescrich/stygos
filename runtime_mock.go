//go:build !tinygo

package stygos

// This file wires the host functions to the mock implementations when not building with tinygo.

func init() {
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

