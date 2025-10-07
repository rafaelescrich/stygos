package main

import (
	"encoding/binary"
	"math/big"

	"github.com/rafaelescrich/stygos"
)

// Simple NFT contract implementation
// Demonstrates NFT functionality using Stygos

// Storage keys
var (
	nameKey        = stygos.Keccak256([]byte("name"))
	symbolKey      = stygos.Keccak256([]byte("symbol"))
	totalSupplyKey = stygos.Keccak256([]byte("totalSupply"))
	ownerPrefix    = stygos.Keccak256([]byte("owner"))
	balancePrefix  = stygos.Keccak256([]byte("balance"))
	approvalPrefix = stygos.Keccak256([]byte("approval"))
	metadataPrefix = stygos.Keccak256([]byte("metadata"))
)

// Commands
const (
	CMD_INITIALIZE    = 0
	CMD_MINT          = 1
	CMD_TRANSFER      = 2
	CMD_APPROVE       = 3
	CMD_TRANSFER_FROM = 4
	CMD_GET_OWNER     = 5
	CMD_GET_BALANCE   = 6
	CMD_GET_APPROVAL  = 7
	CMD_SET_METADATA  = 8
	CMD_GET_METADATA  = 9
)

//export entrypoint
func entrypoint() int32 {
	callData, err := stygos.GetCallData()
	if err != nil || len(callData) < 1 {
		return 1 // Invalid input
	}

	command := callData[0]
	args := callData[1:]

	switch command {
	case CMD_INITIALIZE:
		return handleInitialize(args)
	case CMD_MINT:
		return handleMint(args)
	case CMD_TRANSFER:
		return handleTransfer(args)
	case CMD_APPROVE:
		return handleApprove(args)
	case CMD_TRANSFER_FROM:
		return handleTransferFrom(args)
	case CMD_GET_OWNER:
		return handleGetOwner(args)
	case CMD_GET_BALANCE:
		return handleGetBalance(args)
	case CMD_GET_APPROVAL:
		return handleGetApproval(args)
	case CMD_SET_METADATA:
		return handleSetMetadata(args)
	case CMD_GET_METADATA:
		return handleGetMetadata(args)
	default:
		return 1 // Unknown command
	}
}

// handleInitialize initializes the NFT contract
func handleInitialize(args []byte) int32 {
	if len(args) < 2 {
		return 1
	}

	nameLen := int(args[0])
	symbolLen := int(args[1])

	if len(args) < 2+nameLen+symbolLen {
		return 1
	}

	name := args[2 : 2+nameLen]
	symbol := args[2+nameLen : 2+nameLen+symbolLen]

	// Store name and symbol
	nameWord := stygos.WordFromBigInt(new(big.Int).SetBytes(name))
	stygos.StorageStore(nameKey, nameWord)

	symbolWord := stygos.WordFromBigInt(new(big.Int).SetBytes(symbol))
	stygos.StorageStore(symbolKey, symbolWord)

	// Initialize total supply
	stygos.StorageStore(totalSupplyKey, stygos.WordFromUint64(0))

	return 0
}

// handleMint mints a new NFT
func handleMint(args []byte) int32 {
	if len(args) < 20 {
		return 1
	}

	var to stygos.Address
	copy(to[:], args[:20])

	// Get current total supply
	totalSupply := stygos.Uint64FromWord(stygos.StorageLoad(totalSupplyKey))
	tokenId := totalSupply + 1

	// Set owner
	ownerKey := getOwnerKey(tokenId)
	stygos.StorageStore(ownerKey, stygos.PadAddress(to))

	// Update balance
	balanceKey := getBalanceKey(to)
	currentBalance := stygos.Uint64FromWord(stygos.StorageLoad(balanceKey))
	stygos.StorageStore(balanceKey, stygos.WordFromUint64(currentBalance+1))

	// Update total supply
	stygos.StorageStore(totalSupplyKey, stygos.WordFromUint64(tokenId))

	// Emit event
	emitTransfer(stygos.Address{}, to, tokenId)

	return 0
}

// handleTransfer transfers an NFT
func handleTransfer(args []byte) int32 {
	if len(args) < 40 {
		return 1
	}

	var to stygos.Address
	copy(to[:], args[:20])

	tokenId := binary.BigEndian.Uint64(args[20:28])

	// Check ownership
	ownerKey := getOwnerKey(tokenId)
	currentOwner := stygos.AddressFromWord(stygos.StorageLoad(ownerKey))

	caller := getCaller()
	if currentOwner != caller {
		return 1
	}

	// Update owner
	stygos.StorageStore(ownerKey, stygos.PadAddress(to))

	// Update balances
	fromBalanceKey := getBalanceKey(currentOwner)
	fromBalance := stygos.Uint64FromWord(stygos.StorageLoad(fromBalanceKey))
	stygos.StorageStore(fromBalanceKey, stygos.WordFromUint64(fromBalance-1))

	toBalanceKey := getBalanceKey(to)
	toBalance := stygos.Uint64FromWord(stygos.StorageLoad(toBalanceKey))
	stygos.StorageStore(toBalanceKey, stygos.WordFromUint64(toBalance+1))

	// Clear approval
	approvalKey := getApprovalKey(tokenId)
	stygos.StorageStore(approvalKey, stygos.WordFromUint64(0))

	// Emit event
	emitTransfer(currentOwner, to, tokenId)

	return 0
}

// handleApprove approves an address to transfer an NFT
func handleApprove(args []byte) int32 {
	if len(args) < 28 {
		return 1
	}

	var to stygos.Address
	copy(to[:], args[:20])

	tokenId := binary.BigEndian.Uint64(args[20:28])

	// Check ownership
	ownerKey := getOwnerKey(tokenId)
	owner := stygos.AddressFromWord(stygos.StorageLoad(ownerKey))

	caller := getCaller()
	if owner != caller {
		return 1
	}

	// Set approval
	approvalKey := getApprovalKey(tokenId)
	stygos.StorageStore(approvalKey, stygos.PadAddress(to))

	// Emit event
	emitApproval(owner, to, tokenId)

	return 0
}

// handleTransferFrom transfers an NFT from one address to another
func handleTransferFrom(args []byte) int32 {
	if len(args) < 60 {
		return 1
	}

	var from stygos.Address
	copy(from[:], args[:20])

	var to stygos.Address
	copy(to[:], args[20:40])

	tokenId := binary.BigEndian.Uint64(args[40:48])

	// Check ownership
	ownerKey := getOwnerKey(tokenId)
	owner := stygos.AddressFromWord(stygos.StorageLoad(ownerKey))

	if owner != from {
		return 1
	}

	// Check approval
	caller := getCaller()
	approvalKey := getApprovalKey(tokenId)
	approved := stygos.AddressFromWord(stygos.StorageLoad(approvalKey))

	if caller != approved && caller != owner {
		return 1
	}

	// Update owner
	stygos.StorageStore(ownerKey, stygos.PadAddress(to))

	// Update balances
	fromBalanceKey := getBalanceKey(from)
	fromBalance := stygos.Uint64FromWord(stygos.StorageLoad(fromBalanceKey))
	stygos.StorageStore(fromBalanceKey, stygos.WordFromUint64(fromBalance-1))

	toBalanceKey := getBalanceKey(to)
	toBalance := stygos.Uint64FromWord(stygos.StorageLoad(toBalanceKey))
	stygos.StorageStore(toBalanceKey, stygos.WordFromUint64(toBalance+1))

	// Clear approval
	stygos.StorageStore(approvalKey, stygos.WordFromUint64(0))

	// Emit event
	emitTransfer(from, to, tokenId)

	return 0
}

// handleGetOwner returns the owner of an NFT
func handleGetOwner(args []byte) int32 {
	if len(args) < 8 {
		return 1
	}

	tokenId := binary.BigEndian.Uint64(args[:8])
	ownerKey := getOwnerKey(tokenId)
	owner := stygos.StorageLoad(ownerKey)

	result := make([]byte, 20)
	copy(result, owner[12:32])

	stygos.SetReturnData(result)
	return 0
}

// handleGetBalance returns the balance of an address
func handleGetBalance(args []byte) int32 {
	if len(args) < 20 {
		return 1
	}

	var owner stygos.Address
	copy(owner[:], args[:20])

	balanceKey := getBalanceKey(owner)
	balance := stygos.Uint64FromWord(stygos.StorageLoad(balanceKey))

	result := make([]byte, 8)
	binary.BigEndian.PutUint64(result, balance)

	stygos.SetReturnData(result)
	return 0
}

// handleGetApproval returns the approved address for an NFT
func handleGetApproval(args []byte) int32 {
	if len(args) < 8 {
		return 1
	}

	tokenId := binary.BigEndian.Uint64(args[:8])
	approvalKey := getApprovalKey(tokenId)
	approved := stygos.StorageLoad(approvalKey)

	result := make([]byte, 20)
	copy(result, approved[12:32])

	stygos.SetReturnData(result)
	return 0
}

// handleSetMetadata sets metadata for an NFT
func handleSetMetadata(args []byte) int32 {
	if len(args) < 9 {
		return 1
	}

	tokenId := binary.BigEndian.Uint64(args[:8])
	metadataLen := int(args[8])

	if len(args) < 9+metadataLen {
		return 1
	}

	metadata := args[9 : 9+metadataLen]

	// Check ownership
	ownerKey := getOwnerKey(tokenId)
	owner := stygos.AddressFromWord(stygos.StorageLoad(ownerKey))

	caller := getCaller()
	if owner != caller {
		return 1
	}

	// Store metadata
	metadataKey := getMetadataKey(tokenId)
	metadataWord := stygos.WordFromBigInt(new(big.Int).SetBytes(metadata))
	stygos.StorageStore(metadataKey, metadataWord)

	return 0
}

// handleGetMetadata returns metadata for an NFT
func handleGetMetadata(args []byte) int32 {
	if len(args) < 8 {
		return 1
	}

	tokenId := binary.BigEndian.Uint64(args[:8])
	metadataKey := getMetadataKey(tokenId)
	metadata := stygos.StorageLoad(metadataKey)

	metadataBytes := stygos.BigIntFromWord(metadata).Bytes()
	stygos.SetReturnData(metadataBytes)
	return 0
}

// Helper functions

func getCaller() stygos.Address {
	// In a real implementation, this would get the caller address
	// For now, return a mock address
	return stygos.Address{}
}

func getOwnerKey(tokenId uint64) stygos.Word {
	tokenIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tokenIdBytes, tokenId)
	return stygos.Keccak256(append(ownerPrefix[:], tokenIdBytes...))
}

func getBalanceKey(owner stygos.Address) stygos.Word {
	return stygos.Keccak256(append(balancePrefix[:], owner[:]...))
}

func getApprovalKey(tokenId uint64) stygos.Word {
	tokenIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tokenIdBytes, tokenId)
	return stygos.Keccak256(append(approvalPrefix[:], tokenIdBytes...))
}

func getMetadataKey(tokenId uint64) stygos.Word {
	tokenIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tokenIdBytes, tokenId)
	return stygos.Keccak256(append(metadataPrefix[:], tokenIdBytes...))
}

// Event emission functions

func emitTransfer(from, to stygos.Address, tokenId uint64) {
	eventData := make([]byte, 20+20+8)
	copy(eventData[:20], from[:])
	copy(eventData[20:40], to[:])
	binary.BigEndian.PutUint64(eventData[40:48], tokenId)

	eventHash := stygos.Keccak256([]byte("Transfer(address,address,uint64)"))
	stygos.EmitEvent(eventData, eventHash)
}

func emitApproval(owner, approved stygos.Address, tokenId uint64) {
	eventData := make([]byte, 20+20+8)
	copy(eventData[:20], owner[:])
	copy(eventData[20:40], approved[:])
	binary.BigEndian.PutUint64(eventData[40:48], tokenId)

	eventHash := stygos.Keccak256([]byte("Approval(address,address,uint64)"))
	stygos.EmitEvent(eventData, eventHash)
}
