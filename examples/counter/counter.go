package main

import (
	"encoding/binary"

	"github.com/rafaelescrich/stygos"
)

// Storage keys
var (
	counterKey = stygos.Keccak256([]byte("counter"))
)

// Commands
const (
	CMD_GET       = 0
	CMD_INCREMENT = 1
	CMD_DECREMENT = 2
	CMD_RESET     = 3
)

// Counter contract implementation
func main() {
	// This function is required by Go but not used directly by Stylus
}

//export entrypoint
func entrypoint() int32 {
	// Get the call data
	callData, err := stygos.GetCallData()
	if err != nil {
		return 1 // Error getting call data
	}

	// Default to GET if no command is provided
	command := CMD_GET
	if len(callData) >= 1 {
		command = int(callData[0])
	}

	// Get the current counter value
	counterValue := getCounter()

	// Process the command
	switch command {
	case CMD_INCREMENT:
		counterValue++
		setCounter(counterValue)
		// Emit an event for the increment
		emitCounterEvent("Increment", counterValue)
	case CMD_DECREMENT:
		if counterValue > 0 {
			counterValue--
		}
		setCounter(counterValue)
		// Emit an event for the decrement
		emitCounterEvent("Decrement", counterValue)
	case CMD_RESET:
		counterValue = 0
		setCounter(counterValue)
		// Emit an event for the reset
		emitCounterEvent("Reset", counterValue)
	case CMD_GET:
		// No state change, just return the current value
	}

	// Return the current counter value
	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, counterValue)
	stygos.SetReturnData(result)

	return 0 // Success
}

// getCounter retrieves the current counter value from storage
func getCounter() uint32 {
	valueWord := stygos.StorageLoad(counterKey)
	return binary.BigEndian.Uint32(valueWord[28:32])
}

// setCounter stores the counter value in storage
func setCounter(value uint32) {
	var valueWord stygos.Word
	binary.BigEndian.PutUint32(valueWord[28:32], value)
	stygos.StorageStore(counterKey, valueWord)
}

// emitCounterEvent emits an event with the counter value
func emitCounterEvent(action string, value uint32) {
	// Create event data
	data := make([]byte, 36) // action string + uint32
	copy(data, action)
	binary.BigEndian.PutUint32(data[32:], value)

	// Create event topic (keccak256 of "CounterEvent(string,uint32)")
	eventSignature := stygos.Keccak256([]byte("CounterEvent(string,uint32)"))

	// Emit the event
	stygos.EmitEvent(data, eventSignature)
}
