//go:build tinygo

package stygos

// This file defines the low-level host functions provided by the Arbitrum Stylus environment.
// These functions are imported from the host environment using //go:wasmimport directives.

//go:wasmimport stylus read_args
func read_args(ptr *byte) uint32

//go:wasmimport stylus write_result
func write_result(ptr *byte, len uint32)

//go:wasmimport stylus storage_load_bytes32
func storage_load_bytes32(key_ptr *byte, value_ptr *byte)

//go:wasmimport stylus storage_store_bytes32
func storage_store_bytes32(key_ptr *byte, value_ptr *byte)

//go:wasmimport stylus msg_value
func msg_value(value_ptr *byte)

//go:wasmimport stylus block_number
func block_number(value_ptr *byte)

//go:wasmimport stylus emit_log
func emit_log(ptr *byte, len uint32, topics_count uint32, topic1_ptr *byte, topic2_ptr *byte, topic3_ptr *byte, topic4_ptr *byte)

//go:wasmimport stylus native_keccak256
func native_keccak256(ptr *byte, len uint32, result_ptr *byte)

//go:wasmimport vm_hooks memory_grow
func memory_grow(pages uint32)
