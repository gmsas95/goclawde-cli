// Package reflection implements the Reflection Engine for self-auditing memory system
package reflection

import (
	"encoding/json"
	"strings"
)

// Helper functions used across reflection package

// parseJSON unmarshals JSON string
func parseJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

// toJSON marshals to JSON string
func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// truncateString truncates a string to max length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// formatNumber formats a number with commas
func formatNumber(n int) string {
	if n < 1000 {
		return string(rune('0' + n))
	}

	// Simple formatting for thousands
	result := ""
	str := ""
	for n > 0 {
		digit := n % 10
		str = string(rune('0'+digit)) + str
		n /= 10
		if len(str)%4 == 3 && n > 0 {
			str = "," + str
		}
	}
	result = str
	return result
}
