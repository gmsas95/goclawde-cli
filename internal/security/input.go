package security

import (
	"errors"
	"unicode"
)

var (
	ErrInputTooLarge      = errors.New("input exceeds maximum size")
	ErrNullByteDetected   = errors.New("null byte detected in input")
	ErrHighWhitespaceRatio = errors.New("suspicious whitespace ratio")
	ErrRepetitiveContent  = errors.New("excessive repetition detected")
)

type InputValidator struct {
	MaxSize          int64
	MaxWhitespaceRatio float64
	MaxRepetition    int
}

func NewInputValidator() *InputValidator {
	return &InputValidator{
		MaxSize:           100 * 1024,
		MaxWhitespaceRatio: 0.8,
		MaxRepetition:     100,
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

func hasExcessiveRepetition(input string, maxLen int) bool {
	if len(input) < maxLen {
		return false
	}

	runes := []rune(input)
	consecutiveCount := 1

	for i := 1; i < len(runes); i++ {
		if runes[i] == runes[i-1] {
			consecutiveCount++
			if consecutiveCount > maxLen {
				return true
			}
		} else {
			consecutiveCount = 1
		}
	}

	return false
}

func ValidateInput(input string) error {
	return NewInputValidator().Validate(input)
}
