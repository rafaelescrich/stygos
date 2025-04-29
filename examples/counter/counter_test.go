package main

import (
	"encoding/binary"
	"testing"

	"github.com/rafaelescrich/stygos"
)

func TestCounter(t *testing.T) {
	mock := stygos.NewMockRuntime()
	stygos.UseRuntime(mock)

	tests := []struct {
		name     string
		command  byte
		wantVal  uint32
		wantLogs int
	}{
		{"Initial Get", CMD_GET, 0, 0},
		{"First Increment", CMD_INCREMENT, 1, 1},
		{"Second Increment", CMD_INCREMENT, 2, 1},
		{"Decrement", CMD_DECREMENT, 1, 1},
		{"Reset", CMD_RESET, 0, 1},
		{"Decrement At Zero", CMD_DECREMENT, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.Args = []byte{tt.command}
			mock.Logs = nil

			result := entrypoint()
			if result != 0 {
				t.Errorf("entrypoint() = %v, want 0", result)
			}

			val := binary.BigEndian.Uint32(mock.Result)
			if val != tt.wantVal {
				t.Errorf("counter value = %v, want %v", val, tt.wantVal)
			}

			if len(mock.Logs) != tt.wantLogs {
				t.Errorf("got %v logs, want %v", len(mock.Logs), tt.wantLogs)
			}
		})
	}
}

func TestInvalidInput(t *testing.T) {
	mock := stygos.NewMockRuntime()
	stygos.UseRuntime(mock)

	mock.Args = []byte{255} // Invalid command
	result := entrypoint()
	if result != 0 {
		t.Errorf("entrypoint() with invalid command = %v, want 0", result)
	}

	val := binary.BigEndian.Uint32(mock.Result)
	if val != 0 {
		t.Errorf("counter value = %v, want 0", val)
	}
}

func TestEventEmission(t *testing.T) {
	mock := stygos.NewMockRuntime()
	stygos.UseRuntime(mock)

	mock.Args = []byte{CMD_INCREMENT}
	entrypoint()

	if len(mock.Logs) != 1 {
		t.Fatalf("got %v logs, want 1", len(mock.Logs))
	}

	eventData := mock.Logs[0]
	if len(eventData) != 36 {
		t.Errorf("got data length %v, want 36", len(eventData))
	}

	action := string(eventData[:32])
	if action != "Increment" {
		t.Errorf("got action %v, want Increment", action)
	}

	value := binary.BigEndian.Uint32(eventData[32:])
	if value != 1 {
		t.Errorf("got value %v, want 1", value)
	}
}
