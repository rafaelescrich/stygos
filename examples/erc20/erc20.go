package main

import (
	"encoding/binary"
	"errors"

	"github.com/rafaelescrich/stygos"
)

// Storage keys
var (
	nameKey         = stygos.Keccak256([]byte("name"))
	symbolKey       = stygos.Keccak256([]byte("symbol"))
	decimalsKey     = stygos.Keccak256([]byte("decimals"))
	totalSupplyKey  = stygos.Keccak256([]byte("totalSupply"))
	balancePrefix   = stygos.Keccak256([]byte("balance"))
	allowancePrefix = stygos.Keccak256([]byte("allowance"))
)

// Commands
const (
	CMD_NAME          = 0
	CMD_SYMBOL        = 1
	CMD_DECIMALS      = 2
	CMD_TOTAL_SUPPLY  = 3
	CMD_BALANCE_OF    = 4
	CMD_TRANSFER      = 5
	CMD_ALLOWANCE     = 6
	CMD_APPROVE       = 7
	CMD_TRANSFER_FROM = 8
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
	case CMD_NAME:
		name := getName()
		stygos.SetReturnData([]byte(name))
	case CMD_SYMBOL:
		symbol := getSymbol()
		stygos.SetReturnData([]byte(symbol))
	case CMD_DECIMALS:
		decimals := getDecimals()
		result := make([]byte, 1)
		result[0] = decimals
		stygos.SetReturnData(result)
	case CMD_TOTAL_SUPPLY:
		supply := getTotalSupply()
		result := make([]byte, 8)
		binary.BigEndian.PutUint64(result, supply)
		stygos.SetReturnData(result)
	case CMD_BALANCE_OF:
		if len(args) != 20 {
			return 1
		}
		var addr stygos.Address
		copy(addr[:], args)
		balance := getBalance(addr)
		result := make([]byte, 8)
		binary.BigEndian.PutUint64(result, balance)
		stygos.SetReturnData(result)
	case CMD_TRANSFER:
		if len(args) != 40 {
			return 1
		}
		var to stygos.Address
		copy(to[:], args[:20])
		amount := binary.BigEndian.Uint64(args[20:])
		err := transfer(to, amount)
		if err != nil {
			return 1
		}
	case CMD_ALLOWANCE:
		if len(args) != 40 {
			return 1
		}
		var owner, spender stygos.Address
		copy(owner[:], args[:20])
		copy(spender[:], args[20:])
		allowance := getAllowance(owner, spender)
		result := make([]byte, 8)
		binary.BigEndian.PutUint64(result, allowance)
		stygos.SetReturnData(result)
	case CMD_APPROVE:
		if len(args) != 40 {
			return 1
		}
		var spender stygos.Address
		copy(spender[:], args[:20])
		amount := binary.BigEndian.Uint64(args[20:])
		err := approve(spender, amount)
		if err != nil {
			return 1
		}
	case CMD_TRANSFER_FROM:
		if len(args) != 60 {
			return 1
		}
		var from, to stygos.Address
		copy(from[:], args[:20])
		copy(to[:], args[20:40])
		amount := binary.BigEndian.Uint64(args[40:])
		err := transferFrom(from, to, amount)
		if err != nil {
			return 1
		}
	default:
		return 1
	}

	return 0
}

func getName() string {
	value := stygos.StorageLoad(nameKey)
	return string(value[:])
}

func getSymbol() string {
	value := stygos.StorageLoad(symbolKey)
	return string(value[:])
}

func getDecimals() uint8 {
	value := stygos.StorageLoad(decimalsKey)
	return value[31]
}

func getTotalSupply() uint64 {
	value := stygos.StorageLoad(totalSupplyKey)
	return stygos.Uint64FromWord(value)
}

func getBalance(addr stygos.Address) uint64 {
	key := stygos.Keccak256(append(balancePrefix[:], addr[:]...))
	value := stygos.StorageLoad(key)
	return stygos.Uint64FromWord(value)
}

func transfer(to stygos.Address, amount uint64) error {
	caller := stygos.AddressFromWord(stygos.StorageLoad(stygos.Keccak256([]byte("caller"))))
	balance := getBalance(caller)
	if balance < amount {
		return errors.New("insufficient balance")
	}

	// Update sender balance
	senderKey := stygos.Keccak256(append(balancePrefix[:], caller[:]...))
	senderValue := stygos.WordFromUint64(balance - amount)
	stygos.StorageStore(senderKey, senderValue)

	// Update recipient balance
	recipientKey := stygos.Keccak256(append(balancePrefix[:], to[:]...))
	recipientBalance := getBalance(to)
	recipientValue := stygos.WordFromUint64(recipientBalance + amount)
	stygos.StorageStore(recipientKey, recipientValue)

	return nil
}

func getAllowance(owner, spender stygos.Address) uint64 {
	key := stygos.Keccak256(append(append(allowancePrefix[:], owner[:]...), spender[:]...))
	value := stygos.StorageLoad(key)
	return stygos.Uint64FromWord(value)
}

func approve(spender stygos.Address, amount uint64) error {
	caller := stygos.AddressFromWord(stygos.StorageLoad(stygos.Keccak256([]byte("caller"))))
	key := stygos.Keccak256(append(append(allowancePrefix[:], caller[:]...), spender[:]...))
	value := stygos.WordFromUint64(amount)
	stygos.StorageStore(key, value)
	return nil
}

func transferFrom(from, to stygos.Address, amount uint64) error {
	caller := stygos.AddressFromWord(stygos.StorageLoad(stygos.Keccak256([]byte("caller"))))
	allowance := getAllowance(from, caller)
	if allowance < amount {
		return errors.New("insufficient allowance")
	}

	fromBalance := getBalance(from)
	if fromBalance < amount {
		return errors.New("insufficient balance")
	}

	// Update allowance
	allowanceKey := stygos.Keccak256(append(append(allowancePrefix[:], from[:]...), caller[:]...))
	allowanceValue := stygos.WordFromUint64(allowance - amount)
	stygos.StorageStore(allowanceKey, allowanceValue)

	// Update from balance
	fromKey := stygos.Keccak256(append(balancePrefix[:], from[:]...))
	fromValue := stygos.WordFromUint64(fromBalance - amount)
	stygos.StorageStore(fromKey, fromValue)

	// Update to balance
	toKey := stygos.Keccak256(append(balancePrefix[:], to[:]...))
	toBalance := getBalance(to)
	toValue := stygos.WordFromUint64(toBalance + amount)
	stygos.StorageStore(toKey, toValue)

	return nil
}
