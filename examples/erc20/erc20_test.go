package main

import (
	"testing"

	"github.com/rafaelescrich/stygos"
)

func TestERC20(t *testing.T) {
	// Initialize mock runtime
	mock := stygos.NewMockRuntime()
	stygos.UseRuntime(mock)

	// Initialize test addresses
	var owner, spender, recipient stygos.Address
	copy(owner[:], []byte("owner12345678901234"))
	copy(spender[:], []byte("spender12345678901"))
	copy(recipient[:], []byte("recipient123456789"))

	// Test storage keys
	nameKey := stygos.Keccak256([]byte("name"))
	symbolKey := stygos.Keccak256([]byte("symbol"))
	decimalsKey := stygos.Keccak256([]byte("decimals"))
	totalSupplyKey := stygos.Keccak256([]byte("totalSupply"))
	balancePrefix := stygos.Keccak256([]byte("balance"))
	allowancePrefix := stygos.Keccak256([]byte("allowance"))

	// Initialize contract state
	stygos.StorageStore(nameKey, stygos.WordFromUint64(0))   // "TestToken"
	stygos.StorageStore(symbolKey, stygos.WordFromUint64(0)) // "TTK"
	stygos.StorageStore(decimalsKey, stygos.WordFromUint64(18))
	stygos.StorageStore(totalSupplyKey, stygos.WordFromUint64(1000000))

	// Set initial owner balance
	ownerBalanceKey := stygos.Keccak256(append(balancePrefix[:], owner[:]...))
	stygos.StorageStore(ownerBalanceKey, stygos.WordFromUint64(1000))

	// Set initial allowance
	allowanceKey := stygos.Keccak256(append(append(allowancePrefix[:], owner[:]...), spender[:]...))
	stygos.StorageStore(allowanceKey, stygos.WordFromUint64(1000))

	// Set caller to owner for testing
	callerKey := stygos.Keccak256([]byte("caller"))
	stygos.StorageStore(callerKey, stygos.PadAddress(owner))

	// Test transfer
	err := transfer(recipient, 500)
	if err != nil {
		t.Errorf("Transfer failed: %v", err)
	}

	// Verify balances after transfer
	ownerBalance := getBalance(owner)
	if ownerBalance != 500 {
		t.Errorf("Expected owner balance 500, got %d", ownerBalance)
	}

	recipientBalance := getBalance(recipient)
	if recipientBalance != 500 {
		t.Errorf("Expected recipient balance 500, got %d", recipientBalance)
	}

	// Test approve and allowance
	err = approve(spender, 1000)
	if err != nil {
		t.Errorf("Approve failed: %v", err)
	}

	allowance := getAllowance(owner, spender)
	if allowance != 1000 {
		t.Errorf("Expected allowance 1000, got %d", allowance)
	}

	// Set caller to spender for transferFrom test
	stygos.StorageStore(callerKey, stygos.PadAddress(spender))

	// Test transferFrom
	err = transferFrom(owner, recipient, 500)
	if err != nil {
		t.Errorf("TransferFrom failed: %v", err)
	}

	// Verify final balances
	ownerBalance = getBalance(owner)
	if ownerBalance != 0 {
		t.Errorf("Expected owner balance 0, got %d", ownerBalance)
	}

	recipientBalance = getBalance(recipient)
	if recipientBalance != 1000 {
		t.Errorf("Expected recipient balance 1000, got %d", recipientBalance)
	}

	// Verify allowance was reduced
	allowance = getAllowance(owner, spender)
	if allowance != 500 {
		t.Errorf("Expected allowance 500, got %d", allowance)
	}
}
