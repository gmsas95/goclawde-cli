// Package idgen provides utilities for generating unique identifiers with prefixes.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// Generate creates a unique ID with the given prefix.
// The ID format is: prefix_ + 16 hex characters (8 bytes)
// Example: Generate("task") returns "task_a1b2c3d4e5f67890"
func Generate(prefix string) string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if crypto/rand fails
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(bytes)
}

// GenerateWithSuffix creates a unique ID with prefix and custom suffix length.
// The ID format is: prefix_ + hex characters (length/2 bytes)
func GenerateWithSuffix(prefix string, byteLength int) string {
	if byteLength < 1 {
		byteLength = 8
	}
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(bytes)
}

// Common prefixes used across the application
const (
	PrefixTask         = "task"
	PrefixCalendar     = "cal"
	PrefixHealth       = "hlth"
	PrefixExpense      = "exp"
	PrefixShopping     = "shop"
	PrefixIntelligence = "int"
	PrefixKnowledge    = "know"
	PrefixNote         = "note"
	PrefixDocument     = "doc"
	PrefixReminder     = "rem"
	PrefixEvent        = "evt"
	PrefixProject      = "proj"
	PrefixUser         = "usr"
)
