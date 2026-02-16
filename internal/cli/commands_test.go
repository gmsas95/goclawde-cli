package cli

import (
	"testing"
)

func TestChannelStatus(t *testing.T) {
	tests := []struct {
		enabled  bool
		expected string
	}{
		{true, "✅ enabled"},
		{false, "❌ disabled"},
	}

	for _, tt := range tests {
		result := channelStatus(tt.enabled)
		if result != tt.expected {
			t.Errorf("channelStatus(%v) = %q, want %q", tt.enabled, result, tt.expected)
		}
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		token    string
		expected string
	}{
		{"1234567890", "1234...7890"},
		{"1234567890abcdef", "1234...cdef"},
		{"short", "***"},
		{"", "***"},
		{"1234567", "***"},
		{"sk-1234567890abcdef", "sk-1...cdef"},
	}

	for _, tt := range tests {
		result := maskToken(tt.token)
		if result != tt.expected {
			t.Errorf("maskToken(%q) = %q, want %q", tt.token, result, tt.expected)
		}
	}
}

func TestPrintFunctions(t *testing.T) {
	PrintExtendedHelp()
	PrintProjectHelp()
	PrintBatchHelp()
	PrintConfigHelp()
	PrintChannelsHelp()
	PrintGatewayHelp()
	PrintInteractiveHelp()
}

func TestHandleCommandsNoArgs(t *testing.T) {
	HandleProjectCommand([]string{})
	HandlePersonaCommand([]string{})
	HandleUserCommand([]string{})
	HandleConfigCommand([]string{})
	HandleChannelsCommand([]string{})
	HandleGatewayCommand([]string{}, nil)
	HandleBatchCommand([]string{})
}

func TestHandleBatchCommandHelp(t *testing.T) {
	HandleBatchCommand([]string{"-h"})
}
