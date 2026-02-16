package security

import (
	"errors"
	"unicode"
)

var (
	ErrInputTooLarge       = errors.New("input exceeds maximum size")
	ErrNullByteDetected    = errors.New("null byte detected in input")
	ErrHighWhitespaceRatio = errors.New("suspicious whitespace ratio")
	ErrRepetitiveContent   = errors.New("excessive repetition detected")
)

type InputValidator struct {
	MaxSize            int64
	MaxWhitespaceRatio float64
	MaxRepetition      int
}

func NewInputValidator() *InputValidator {
	return &InputValidator{
		MaxSize:            100 * 1024,
		MaxWhitespaceRatio: 0.8,
		MaxRepetition:      10000, // High threshold to avoid false positives
	}
}

func (v *InputValidator) Validate(input string) error {
	if int64(len(input)) > v.MaxSize {
		return ErrInputTooLarge
	}

	for i := 0; i < len(input); i++ {
		if input[i] == 0 {
			return ErrNullByteDetected
		}
	}

	if v.MaxWhitespaceRatio > 0 && len(input) > 0 {
		whitespaceCount := 0
		for _, r := range input {
			if unicode.IsSpace(r) {
				whitespaceCount++
			}
		}
		ratio := float64(whitespaceCount) / float64(len(input))
		if ratio > v.MaxWhitespaceRatio {
			return ErrHighWhitespaceRatio
		}
	}

	if v.MaxRepetition > 0 && len(input) > v.MaxRepetition {
		if hasExcessiveRepetition(input, v.MaxRepetition) {
			return ErrRepetitiveContent
		}
	}

	return nil
}

func hasExcessiveRepetition(input string, threshold int) bool {
	if len(input) < threshold {
		return false
	}

	// Count frequency of each character
	charCount := make(map[rune]int)
	for _, r := range input {
		charCount[r]++
	}

	// Check if any single character dominates the input
	maxCount := 0
	for _, count := range charCount {
		if count > maxCount {
			maxCount = count
		}
	}

	// If any character appears more than threshold times, it's repetitive
	if maxCount >= threshold {
		return true
	}

	// Also check for consecutive repetition (original logic)
	runes := []rune(input)
	consecutiveCount := 1
	maxConsecutive := 1

	for i := 1; i < len(runes); i++ {
		if runes[i] == runes[i-1] {
			consecutiveCount++
			if consecutiveCount > maxConsecutive {
				maxConsecutive = consecutiveCount
			}
		} else {
			consecutiveCount = 1
		}
	}

	return maxConsecutive >= threshold
}

func ValidateInput(input string) error {
	return NewInputValidator().Validate(input)
}
