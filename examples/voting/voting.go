package main

import (
	"encoding/binary"
	"math/big"

	"github.com/rafaelescrich/stygos"
)

// Voting contract implementation
// Demonstrates governance and voting mechanisms using Stygos

// Storage keys
var (
	votingPeriodKey   = stygos.Keccak256([]byte("votingPeriod"))
	quorumKey         = stygos.Keccak256([]byte("quorum"))
	proposalCountKey  = stygos.Keccak256([]byte("proposalCount"))
	proposalPrefix    = stygos.Keccak256([]byte("proposal"))
	votePrefix        = stygos.Keccak256([]byte("vote"))
	voterWeightPrefix = stygos.Keccak256([]byte("voterWeight"))
)

// Commands
const (
	CMD_INITIALIZE       = 0
	CMD_CREATE_PROPOSAL  = 1
	CMD_VOTE             = 2
	CMD_EXECUTE_PROPOSAL = 3
	CMD_GET_PROPOSAL     = 4
	CMD_GET_VOTE         = 5
	CMD_SET_VOTER_WEIGHT = 6
)

// Vote types
const (
	VOTE_AGAINST = 0
	VOTE_FOR     = 1
	VOTE_ABSTAIN = 2
)

// Proposal status
const (
	STATUS_PENDING   = 0
	STATUS_ACTIVE    = 1
	STATUS_DEFEATED  = 2
	STATUS_SUCCEEDED = 3
	STATUS_EXECUTED  = 4
)

// Proposal structure
type Proposal struct {
	Proposer     stygos.Address
	StartBlock   uint64
	EndBlock     uint64
	ForVotes     uint64
	AgainstVotes uint64
	AbstainVotes uint64
	Executed     bool
	Description  []byte
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
	case CMD_CREATE_PROPOSAL:
		return handleCreateProposal(args)
	case CMD_VOTE:
		return handleVote(args)
	case CMD_EXECUTE_PROPOSAL:
		return handleExecuteProposal(args)
	case CMD_GET_PROPOSAL:
		return handleGetProposal(args)
	case CMD_GET_VOTE:
		return handleGetVote(args)
	case CMD_SET_VOTER_WEIGHT:
		return handleSetVoterWeight(args)
	default:
		return 1 // Unknown command
	}
}

// handleInitialize initializes the voting system
func handleInitialize(args []byte) int32 {
	if len(args) < 8 {
		return 1
	}

	votingPeriod := binary.BigEndian.Uint64(args[:8])
	quorum := binary.BigEndian.Uint64(args[8:16])

	// Store configuration
	stygos.StorageStore(votingPeriodKey, stygos.WordFromUint64(votingPeriod))
	stygos.StorageStore(quorumKey, stygos.WordFromUint64(quorum))
	stygos.StorageStore(proposalCountKey, stygos.WordFromUint64(0))

	return 0
}

// handleCreateProposal creates a new proposal
func handleCreateProposal(args []byte) int32 {
	if len(args) < 1 {
		return 1
	}

	descriptionLen := int(args[0])
	if len(args) < 1+descriptionLen {
		return 1
	}

	description := args[1 : 1+descriptionLen]

	// Get current block and voting period
	currentBlock := stygos.GetBlockNumber()
	votingPeriod := stygos.Uint64FromWord(stygos.StorageLoad(votingPeriodKey))

	// Create proposal
	proposal := Proposal{
		Proposer:     getCaller(),
		StartBlock:   currentBlock,
		EndBlock:     currentBlock + votingPeriod,
		ForVotes:     0,
		AgainstVotes: 0,
		AbstainVotes: 0,
		Executed:     false,
		Description:  description,
	}

	// Get next proposal ID
	proposalCount := stygos.Uint64FromWord(stygos.StorageLoad(proposalCountKey))
	proposalId := proposalCount + 1

	// Store proposal
	proposalKey := getProposalKey(proposalId)
	storeProposal(proposalKey, proposal)

	// Increment proposal count
	stygos.StorageStore(proposalCountKey, stygos.WordFromUint64(proposalId))

	// Emit event
	emitProposalCreated(proposalId, proposal.Proposer, description)

	return 0
}

// handleVote casts a vote on a proposal
func handleVote(args []byte) int32 {
	if len(args) < 9 { // 8 (proposalId) + 1 (vote)
		return 1
	}

	proposalId := binary.BigEndian.Uint64(args[:8])
	voteType := args[8]

	if voteType > VOTE_ABSTAIN {
		return 1
	}

	// Get proposal
	proposalKey := getProposalKey(proposalId)
	proposal, exists := getProposal(proposalKey)
	if !exists {
		return 1
	}

	// Check if voting is active
	currentBlock := stygos.GetBlockNumber()
	if currentBlock < proposal.StartBlock || currentBlock > proposal.EndBlock {
		return 1
	}

	// Check if already voted
	voter := getCaller()
	voteKey := getVoteKey(proposalId, voter)
	if hasVote(voteKey) {
		return 1
	}

	// Get voter weight
	voterWeight := getVoterWeight(voter)
	if voterWeight == 0 {
		return 1
	}

	// Update proposal votes
	switch voteType {
	case VOTE_FOR:
		proposal.ForVotes += voterWeight
	case VOTE_AGAINST:
		proposal.AgainstVotes += voterWeight
	case VOTE_ABSTAIN:
		proposal.AbstainVotes += voterWeight
	}

	// Store updated proposal
	storeProposal(proposalKey, proposal)

	// Store vote
	setVote(voteKey, voteType, voterWeight)

	// Emit event
	emitVoteCast(proposalId, voter, voteType, voterWeight)

	return 0
}

// handleExecuteProposal executes a successful proposal
func handleExecuteProposal(args []byte) int32 {
	if len(args) < 8 {
		return 1
	}

	proposalId := binary.BigEndian.Uint64(args[:8])

	// Get proposal
	proposalKey := getProposalKey(proposalId)
	proposal, exists := getProposal(proposalKey)
	if !exists {
		return 1
	}

	if proposal.Executed {
		return 1
	}

	// Check if voting period has ended
	currentBlock := stygos.GetBlockNumber()
	if currentBlock <= proposal.EndBlock {
		return 1
	}

	// Check if proposal succeeded
	totalVotes := proposal.ForVotes + proposal.AgainstVotes + proposal.AbstainVotes
	quorum := stygos.Uint64FromWord(stygos.StorageLoad(quorumKey))

	if totalVotes < quorum {
		return 1
	}

	if proposal.ForVotes <= proposal.AgainstVotes {
		return 1
	}

	// Mark as executed
	proposal.Executed = true
	storeProposal(proposalKey, proposal)

	// Emit event
	emitProposalExecuted(proposalId)

	return 0
}

// handleGetProposal returns proposal data
func handleGetProposal(args []byte) int32 {
	if len(args) < 8 {
		return 1
	}

	proposalId := binary.BigEndian.Uint64(args[:8])
	proposalKey := getProposalKey(proposalId)
	proposal, exists := getProposal(proposalKey)

	if !exists {
		return 1
	}

	// Return proposal data
	result := make([]byte, 20+8+8+8+8+8+1+1+len(proposal.Description))
	offset := 0

	copy(result[offset:offset+20], proposal.Proposer[:])
	offset += 20

	binary.BigEndian.PutUint64(result[offset:offset+8], proposal.StartBlock)
	offset += 8

	binary.BigEndian.PutUint64(result[offset:offset+8], proposal.EndBlock)
	offset += 8

	binary.BigEndian.PutUint64(result[offset:offset+8], proposal.ForVotes)
	offset += 8

	binary.BigEndian.PutUint64(result[offset:offset+8], proposal.AgainstVotes)
	offset += 8

	binary.BigEndian.PutUint64(result[offset:offset+8], proposal.AbstainVotes)
	offset += 8

	if proposal.Executed {
		result[offset] = 1
	} else {
		result[offset] = 0
	}
	offset += 1

	result[offset] = byte(len(proposal.Description))
	offset += 1

	copy(result[offset:offset+len(proposal.Description)], proposal.Description)

	stygos.SetReturnData(result)
	return 0
}

// handleGetVote returns vote data for a voter on a proposal
func handleGetVote(args []byte) int32 {
	if len(args) < 28 { // 8 (proposalId) + 20 (voter)
		return 1
	}

	proposalId := binary.BigEndian.Uint64(args[:8])
	var voter stygos.Address
	copy(voter[:], args[8:28])

	voteKey := getVoteKey(proposalId, voter)
	voteType, weight := getVote(voteKey)

	result := make([]byte, 2)
	result[0] = voteType
	result[1] = byte(weight)

	stygos.SetReturnData(result)
	return 0
}

// handleSetVoterWeight sets the voting weight for a voter
func handleSetVoterWeight(args []byte) int32 {
	if len(args) < 21 { // 20 (voter) + 1 (weight)
		return 1
	}

	var voter stygos.Address
	copy(voter[:], args[:20])
	weight := uint8(args[20])

	voterWeightKey := getVoterWeightKey(voter)
	stygos.StorageStore(voterWeightKey, stygos.WordFromUint64(uint64(weight)))

	// Emit event
	emitVoterWeightSet(voter, weight)

	return 0
}

// Helper functions

func getCaller() stygos.Address {
	// In a real implementation, this would get the caller address
	// For now, return a mock address
	return stygos.Address{}
}

func getProposalKey(proposalId uint64) stygos.Word {
	proposalIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(proposalIdBytes, proposalId)
	return stygos.Keccak256(append(proposalPrefix[:], proposalIdBytes...))
}

func getVoteKey(proposalId uint64, voter stygos.Address) stygos.Word {
	proposalIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(proposalIdBytes, proposalId)
	return stygos.Keccak256(append(append(votePrefix[:], proposalIdBytes...), voter[:]...))
}

func getVoterWeightKey(voter stygos.Address) stygos.Word {
	return stygos.Keccak256(append(voterWeightPrefix[:], voter[:]...))
}

func storeProposal(key stygos.Word, proposal Proposal) {
	// Serialize proposal
	data := make([]byte, 20+8+8+8+8+8+1+1+len(proposal.Description))
	offset := 0

	copy(data[offset:offset+20], proposal.Proposer[:])
	offset += 20

	binary.BigEndian.PutUint64(data[offset:offset+8], proposal.StartBlock)
	offset += 8

	binary.BigEndian.PutUint64(data[offset:offset+8], proposal.EndBlock)
	offset += 8

	binary.BigEndian.PutUint64(data[offset:offset+8], proposal.ForVotes)
	offset += 8

	binary.BigEndian.PutUint64(data[offset:offset+8], proposal.AgainstVotes)
	offset += 8

	binary.BigEndian.PutUint64(data[offset:offset+8], proposal.AbstainVotes)
	offset += 8

	if proposal.Executed {
		data[offset] = 1
	} else {
		data[offset] = 0
	}
	offset += 1

	data[offset] = byte(len(proposal.Description))
	offset += 1

	copy(data[offset:offset+len(proposal.Description)], proposal.Description)

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
	offset := 0

	copy(proposal.Proposer[:], data[offset:offset+20])
	offset += 20

	proposal.StartBlock = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	proposal.EndBlock = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	proposal.ForVotes = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	proposal.AgainstVotes = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	proposal.AbstainVotes = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	proposal.Executed = data[offset] == 1
	offset += 1

	descriptionLen := int(data[offset])
	offset += 1

	if len(data) < offset+descriptionLen {
		return Proposal{}, false
	}

	proposal.Description = make([]byte, descriptionLen)
	copy(proposal.Description, data[offset:offset+descriptionLen])

	return proposal, true
}

func hasVote(key stygos.Word) bool {
	voteWord := stygos.StorageLoad(key)
	return voteWord != (stygos.Word{})
}

func setVote(key stygos.Word, voteType uint8, weight uint64) {
	voteData := make([]byte, 2)
	voteData[0] = voteType
	voteData[1] = byte(weight)

	voteWord := stygos.WordFromBigInt(new(big.Int).SetBytes(voteData))
	stygos.StorageStore(key, voteWord)
}

func getVote(key stygos.Word) (uint8, uint64) {
	voteWord := stygos.StorageLoad(key)
	if voteWord == (stygos.Word{}) {
		return 0, 0
	}

	data := stygos.BigIntFromWord(voteWord).Bytes()
	if len(data) < 2 {
		return 0, 0
	}

	return data[0], uint64(data[1])
}

func getVoterWeight(voter stygos.Address) uint64 {
	voterWeightKey := getVoterWeightKey(voter)
	voterWeightWord := stygos.StorageLoad(voterWeightKey)
	return stygos.Uint64FromWord(voterWeightWord)
}

// Event emission functions

func emitProposalCreated(proposalId uint64, proposer stygos.Address, description []byte) {
	eventData := make([]byte, 8+20+len(description))
	binary.BigEndian.PutUint64(eventData[:8], proposalId)
	copy(eventData[8:28], proposer[:])
	copy(eventData[28:28+len(description)], description)

	eventHash := stygos.Keccak256([]byte("ProposalCreated(uint64,address,bytes)"))
	stygos.EmitEvent(eventData, eventHash)
}

func emitVoteCast(proposalId uint64, voter stygos.Address, voteType uint8, weight uint64) {
	eventData := make([]byte, 8+20+1+8)
	binary.BigEndian.PutUint64(eventData[:8], proposalId)
	copy(eventData[8:28], voter[:])
	eventData[28] = voteType
	binary.BigEndian.PutUint64(eventData[29:37], weight)

	eventHash := stygos.Keccak256([]byte("VoteCast(uint64,address,uint8,uint64)"))
	stygos.EmitEvent(eventData, eventHash)
}

func emitProposalExecuted(proposalId uint64) {
	eventData := make([]byte, 8)
	binary.BigEndian.PutUint64(eventData, proposalId)

	eventHash := stygos.Keccak256([]byte("ProposalExecuted(uint64)"))
	stygos.EmitEvent(eventData, eventHash)
}

func emitVoterWeightSet(voter stygos.Address, weight uint8) {
	eventData := make([]byte, 20+1)
	copy(eventData[:20], voter[:])
	eventData[20] = weight

	eventHash := stygos.Keccak256([]byte("VoterWeightSet(address,uint8)"))
	stygos.EmitEvent(eventData, eventHash)
}
