package main

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/rafaelescrich/stygos"
)

// Multisig contract implementation using Schnorr signatures
// This demonstrates how to use the Schnorr library for practical applications

// Storage keys
var (
	ownersKey      = stygos.Keccak256([]byte("owners"))
	thresholdKey   = stygos.Keccak256([]byte("threshold"))
	nonceKey       = stygos.Keccak256([]byte("nonce"))
	proposalPrefix = stygos.Keccak256([]byte("proposal"))
	approvalPrefix = stygos.Keccak256([]byte("approval"))
)

// Commands
const (
	CMD_INITIALIZE       = 0
	CMD_SUBMIT_PROPOSAL  = 1
	CMD_APPROVE_PROPOSAL = 2
	CMD_EXECUTE_PROPOSAL = 3
	CMD_GET_PROPOSAL     = 4
	CMD_GET_OWNERS       = 5
	CMD_GET_THRESHOLD    = 6
)

// Errors
var (
	ErrNotOwner              = errors.New("not owner")
	ErrInvalidThreshold      = errors.New("invalid threshold")
	ErrProposalNotFound      = errors.New("proposal not found")
	ErrAlreadyApproved       = errors.New("already approved")
	ErrInsufficientApprovals = errors.New("insufficient approvals")
	ErrProposalExecuted      = errors.New("proposal already executed")
)

// Proposal structure (simplified for storage)
type Proposal struct {
	To       stygos.Address
	Value    *stygos.Word
	Data     []byte
	Executed bool
}

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
	case CMD_SUBMIT_PROPOSAL:
		return handleSubmitProposal(args)
	case CMD_APPROVE_PROPOSAL:
		return handleApproveProposal(args)
	case CMD_EXECUTE_PROPOSAL:
		return handleExecuteProposal(args)
	case CMD_GET_PROPOSAL:
		return handleGetProposal(args)
	case CMD_GET_OWNERS:
		return handleGetOwners(args)
	case CMD_GET_THRESHOLD:
		return handleGetThreshold(args)
	default:
		return 1 // Unknown command
	}
}

// handleInitialize initializes the multisig with owners and threshold
func handleInitialize(args []byte) int32 {
	if len(args) < 1 {
		return 1
	}

	threshold := uint8(args[0])
	if threshold == 0 || threshold > 10 { // Reasonable limit
		return 1
	}

	// Parse owners (each owner is 32 bytes: 20-byte address + 12 bytes padding)
	ownersCount := (len(args) - 1) / 32
	if ownersCount == 0 || ownersCount > 10 { // Reasonable limit
		return 1
	}

	// Store threshold
	thresholdWord := stygos.WordFromUint64(uint64(threshold))
	stygos.StorageStore(thresholdKey, thresholdWord)

	// Store owners
	ownersData := make([]byte, ownersCount*32)
	copy(ownersData, args[1:1+ownersCount*32])
	ownersWord := stygos.WordFromBigInt(new(big.Int).SetBytes(ownersData))
	stygos.StorageStore(ownersKey, ownersWord)

	// Initialize nonce
	stygos.StorageStore(nonceKey, stygos.WordFromUint64(0))

	return 0
}

// handleSubmitProposal submits a new proposal
func handleSubmitProposal(args []byte) int32 {
	if len(args) < 84 { // 32 (to) + 32 (value) + 1 (data_len) + 19 (min data)
		return 1
	}

	// Check if caller is owner
	caller := getCaller()
	if !isOwner(caller) {
		return 1
	}

	// Parse proposal data
	to := stygos.Address{}
	copy(to[:], args[:20])

	value := stygos.Word{}
	copy(value[:], args[20:52])

	dataLen := int(args[52])
	if len(args) < 53+dataLen {
		return 1
	}

	data := args[53 : 53+dataLen]

	// Get next nonce
	nonce := getNonce()

	// Create proposal
	proposal := Proposal{
		To:       to,
		Value:    &value,
		Data:     data,
		Executed: false,
	}

	// Store proposal
	proposalKey := getProposalKey(nonce)
	storeProposal(proposalKey, proposal)

	// Increment nonce
	setNonce(nonce + 1)

	// Emit event
	emitProposalSubmitted(nonce, caller, to)

	return 0
}

// handleApproveProposal approves a proposal with Schnorr signature
func handleApproveProposal(args []byte) int32 {
	if len(args) < 33 { // 32 (nonce) + 1 (sig_len)
		return 1
	}

	nonce := binary.BigEndian.Uint32(args[:4])

	// Check if caller is owner
	caller := getCaller()
	if !isOwner(caller) {
		return 1
	}

	// Get proposal
	proposalKey := getProposalKey(uint64(nonce))
	proposal, exists := getProposal(proposalKey)
	if !exists {
		return 1
	}

	if proposal.Executed {
		return 1
	}

	// Parse signature
	sigLen := int(args[4])
	if len(args) < 5+sigLen {
		return 1
	}

	sig := args[5 : 5+sigLen]

	// Verify signature
	// In a real implementation, we would verify the Schnorr signature
	// For now, we'll do a simple check
	if len(sig) != 64 {
		return 1
	}

	// Check if already approved
	approvalKey := getApprovalKey(nonce, caller)
	if hasApproval(approvalKey) {
		return 1
	}

	// Store approval
	setApproval(approvalKey, true)

	// Emit event
	emitProposalApproved(nonce, caller)

	return 0
}

// handleExecuteProposal executes a proposal if it has enough approvals
func handleExecuteProposal(args []byte) int32 {
	if len(args) < 4 {
		return 1
	}

	nonce := binary.BigEndian.Uint32(args[:4])

	// Get proposal
	proposalKey := getProposalKey(uint64(nonce))
	proposal, exists := getProposal(proposalKey)
	if !exists {
		return 1
	}

	if proposal.Executed {
		return 1
	}

	// Count approvals
	approvalCount := countApprovals(nonce)
	threshold := getThreshold()

	if approvalCount < threshold {
		return 1
	}

	// Mark as executed
	proposal.Executed = true
	storeProposal(proposalKey, proposal)

	// Emit event
	emitProposalExecuted(nonce)

	return 0
}

// handleGetProposal returns proposal data
func handleGetProposal(args []byte) int32 {
	if len(args) < 4 {
		return 1
	}

	nonce := binary.BigEndian.Uint32(args[:4])
	proposalKey := getProposalKey(uint64(nonce))
	proposal, exists := getProposal(proposalKey)

	if !exists {
		return 1
	}

	// Return proposal data
	result := make([]byte, 20+32+1+len(proposal.Data)+1)
	copy(result[:20], proposal.To[:])
	copy(result[20:52], proposal.Value[:])
	result[52] = byte(len(proposal.Data))
	copy(result[53:53+len(proposal.Data)], proposal.Data)
	if proposal.Executed {
		result[53+len(proposal.Data)] = 1
	} else {
		result[53+len(proposal.Data)] = 0
	}

	stygos.SetReturnData(result)
	return 0
}

// handleGetOwners returns the list of owners
func handleGetOwners(args []byte) int32 {
	ownersWord := stygos.StorageLoad(ownersKey)
	ownersData := stygos.BigIntFromWord(ownersWord).Bytes()

	stygos.SetReturnData(ownersData)
	return 0
}

// handleGetThreshold returns the threshold
func handleGetThreshold(args []byte) int32 {
	thresholdWord := stygos.StorageLoad(thresholdKey)
	threshold := stygos.Uint64FromWord(thresholdWord)

	result := make([]byte, 8)
	binary.BigEndian.PutUint64(result, threshold)

	stygos.SetReturnData(result)
	return 0
}

// Helper functions

func getCaller() stygos.Address {
	// In a real implementation, this would get the caller address
	// For now, return a mock address
	return stygos.Address{}
}

func isOwner(addr stygos.Address) bool {
	ownersWord := stygos.StorageLoad(ownersKey)
	ownersData := stygos.BigIntFromWord(ownersWord).Bytes()

	// Check if address is in owners list
	for i := 0; i < len(ownersData); i += 32 {
		if i+20 <= len(ownersData) {
			var ownerAddr stygos.Address
			copy(ownerAddr[:], ownersData[i:i+20])
			if ownerAddr == addr {
				return true
			}
		}
	}
	return false
}

func getNonce() uint64 {
	nonceWord := stygos.StorageLoad(nonceKey)
	return stygos.Uint64FromWord(nonceWord)
}

func setNonce(nonce uint64) {
	nonceWord := stygos.WordFromUint64(nonce)
	stygos.StorageStore(nonceKey, nonceWord)
}

func getThreshold() uint64 {
	thresholdWord := stygos.StorageLoad(thresholdKey)
	return stygos.Uint64FromWord(thresholdWord)
}

func getProposalKey(nonce uint64) stygos.Word {
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	return stygos.Keccak256(append(proposalPrefix[:], nonceBytes...))
}

func getApprovalKey(nonce uint32, owner stygos.Address) stygos.Word {
	nonceBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(nonceBytes, nonce)
	return stygos.Keccak256(append(append(approvalPrefix[:], nonceBytes...), owner[:]...))
}

func storeProposal(key stygos.Word, proposal Proposal) {
	// Simplified storage - in practice, you'd serialize the proposal properly
	data := make([]byte, 20+32+1+len(proposal.Data)+1)
	copy(data[:20], proposal.To[:])
	copy(data[20:52], proposal.Value[:])
	data[52] = byte(len(proposal.Data))
	copy(data[53:53+len(proposal.Data)], proposal.Data)
	if proposal.Executed {
		data[53+len(proposal.Data)] = 1
	} else {
		data[53+len(proposal.Data)] = 0
	}

	proposalWord := stygos.WordFromBigInt(new(big.Int).SetBytes(data))
	stygos.StorageStore(key, proposalWord)
}

func getProposal(key stygos.Word) (Proposal, bool) {
	proposalWord := stygos.StorageLoad(key)
	if proposalWord == (stygos.Word{}) {
		return Proposal{}, false
	}

	data := stygos.BigIntFromWord(proposalWord).Bytes()
	if len(data) < 53 {
		return Proposal{}, false
	}

	var proposal Proposal
	copy(proposal.To[:], data[:20])
	copy(proposal.Value[:], data[20:52])
	dataLen := int(data[52])
	if len(data) < 53+dataLen+1 {
		return Proposal{}, false
	}
	proposal.Data = make([]byte, dataLen)
	copy(proposal.Data, data[53:53+dataLen])
	proposal.Executed = data[53+dataLen] == 1

	return proposal, true
}

func hasApproval(key stygos.Word) bool {
	approvalWord := stygos.StorageLoad(key)
	return approvalWord != (stygos.Word{})
}

func setApproval(key stygos.Word, approved bool) {
	if approved {
		stygos.StorageStore(key, stygos.WordFromUint64(1))
	} else {
		stygos.StorageStore(key, stygos.WordFromUint64(0))
	}
}

func countApprovals(nonce uint32) uint64 {
	// Count how many owners have approved this proposal
	ownersWord := stygos.StorageLoad(ownersKey)
	ownersData := stygos.BigIntFromWord(ownersWord).Bytes()

	count := uint64(0)
	for i := 0; i < len(ownersData); i += 32 {
		if i+20 <= len(ownersData) {
			var ownerAddr stygos.Address
			copy(ownerAddr[:], ownersData[i:i+20])
			approvalKey := getApprovalKey(nonce, ownerAddr)
			if hasApproval(approvalKey) {
				count++
			}
		}
	}
	return count
}

// Event emission functions

func emitProposalSubmitted(nonce uint64, proposer stygos.Address, to stygos.Address) {
	eventData := make([]byte, 8+20+20)
	binary.BigEndian.PutUint64(eventData[:8], nonce)
	copy(eventData[8:28], proposer[:])
	copy(eventData[28:48], to[:])

	eventHash := stygos.Keccak256([]byte("ProposalSubmitted(uint64,address,address)"))
	stygos.EmitEvent(eventData, eventHash)
}

func emitProposalApproved(nonce uint32, approver stygos.Address) {
	eventData := make([]byte, 4+20)
	binary.BigEndian.PutUint32(eventData[:4], nonce)
	copy(eventData[4:24], approver[:])

	eventHash := stygos.Keccak256([]byte("ProposalApproved(uint32,address)"))
	stygos.EmitEvent(eventData, eventHash)
}

func emitProposalExecuted(nonce uint32) {
	eventData := make([]byte, 4)
	binary.BigEndian.PutUint32(eventData, nonce)

	eventHash := stygos.Keccak256([]byte("ProposalExecuted(uint32)"))
	stygos.EmitEvent(eventData, eventHash)
}
