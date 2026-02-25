package idgen

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	// Test basic generation
	id := Generate("test")
	if id == "" {
		t.Error("Generate returned empty string")
	}

	// Check prefix
	if !strings.HasPrefix(id, "test_") {
		t.Errorf("Expected prefix 'test_', got: %s", id)
	}

	// Check uniqueness
	id2 := Generate("test")
	if id == id2 {
		t.Error("Generate returned duplicate IDs")
	}
}

func TestGenerateWithDifferentPrefixes(t *testing.T) {
	prefixes := []string{"task", "cal", "exp", "hlth", "shop"}

	for _, prefix := range prefixes {
		id := Generate(prefix)
		expectedPrefix := prefix + "_"
		if !strings.HasPrefix(id, expectedPrefix) {
			t.Errorf("Expected prefix '%s', got: %s", expectedPrefix, id)
		}
	}
}

func TestGenerateWithSuffix(t *testing.T) {
	// Test with default byte length
	id := GenerateWithSuffix("custom", 8)
	if !strings.HasPrefix(id, "custom_") {
		t.Errorf("Expected prefix 'custom_', got: %s", id)
	}

	// Test with custom byte length
	id2 := GenerateWithSuffix("custom", 16)
	if !strings.HasPrefix(id2, "custom_") {
		t.Errorf("Expected prefix 'custom_', got: %s", id2)
	}

	// The ID with 16 bytes should be longer than with 8 bytes
	if len(id2) <= len(id) {
		t.Error("Longer byte length should produce longer ID")
	}
}

func TestConstants(t *testing.T) {
	// Ensure all constants are defined
	constants := []string{
		PrefixTask,
		PrefixCalendar,
		PrefixHealth,
		PrefixExpense,
		PrefixShopping,
		PrefixIntelligence,
		PrefixKnowledge,
		PrefixNote,
		PrefixDocument,
		PrefixReminder,
		PrefixEvent,
		PrefixProject,
		PrefixUser,
	}

	for _, c := range constants {
		if c == "" {
			t.Error("Constant should not be empty")
		}
	}
}
