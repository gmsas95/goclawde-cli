package security

import (
	"strings"
	"testing"
)

func TestInputValidator_ValidInput(t *testing.T) {
	validator := NewInputValidator()
	validInputs := []string{
		"Hello, world!",
		"This is a normal message.",
		"What's the weather like?",
		"Can you help me with my code?",
		"Search for documents about AI",
		strings.Repeat("a", 1000),
	}

	for _, input := range validInputs {
		err := validator.Validate(input)
		if err != nil {
			t.Errorf("Valid input rejected: %s (error: %v)", input[:50], err)
		}
	}
}

func TestInputValidator_TooLarge(t *testing.T) {
	validator := NewInputValidator()
	validator.MaxSize = 100

	largeInput := strings.Repeat("a", 200)
	err := validator.Validate(largeInput)
	if err != ErrInputTooLarge {
		t.Errorf("Large input not rejected, got: %v", err)
	}
}

func TestInputValidator_NullByte(t *testing.T) {
	validator := NewInputValidator()

	inputsWithNull := []string{
		"hello\x00world",
		"\x00",
		"test\x00",
		"\x00test",
		"normal text\x00more text",
	}

	for _, input := range inputsWithNull {
		err := validator.Validate(input)
		if err != ErrNullByteDetected {
			t.Errorf("Null byte not detected in: %q", input)
		}
	}
}

func TestInputValidator_HighWhitespaceRatio(t *testing.T) {
	validator := NewInputValidator()
	validator.MaxWhitespaceRatio = 0.8

	highWhitespaceInputs := []string{
		strings.Repeat(" ", 100),
		strings.Repeat("\t", 100),
		strings.Repeat("\n", 100),
		"   \t   \n   \r   ",
	}

	for _, input := range highWhitespaceInputs {
		err := validator.Validate(input)
		if err != ErrHighWhitespaceRatio {
			t.Errorf("High whitespace ratio not detected")
		}
	}
}

func TestInputValidator_LowWhitespaceRatio(t *testing.T) {
	validator := NewInputValidator()

	lowWhitespaceInputs := []string{
		"ThisIsAllOneWord",
		"This is normal text with normal spacing.",
		"a b c d e f g h i j",
	}

	for _, input := range lowWhitespaceInputs {
		err := validator.Validate(input)
		if err != nil {
			t.Errorf("Normal whitespace input rejected: %v", err)
		}
	}
}

func TestInputValidator_RepetitiveContent(t *testing.T) {
	validator := NewInputValidator()
	validator.MaxRepetition = 100

	repetitiveInputs := []string{
		strings.Repeat("a", 200),
		strings.Repeat("x", 150),
	}

	for _, input := range repetitiveInputs {
		err := validator.Validate(input)
		if err != ErrRepetitiveContent {
			t.Errorf("Repetitive content not detected for length %d", len(input))
		}
	}
}

func TestInputValidator_NonRepetitiveContent(t *testing.T) {
	validator := NewInputValidator()

	nonRepetitiveInputs := []string{
		"abcdefghijklmnopqrstuvwxyz",
		"The quick brown fox jumps over the lazy dog.",
		"1234567890!@#$%^&*()",
	}

	for _, input := range nonRepetitiveInputs {
		err := validator.Validate(input)
		if err != nil {
			t.Errorf("Non-repetitive input rejected: %v", err)
		}
	}
}

func TestInputValidator_EmptyInput(t *testing.T) {
	validator := NewInputValidator()

	err := validator.Validate("")
	if err != nil {
		t.Errorf("Empty input rejected: %v", err)
	}
}

func TestInputValidator_CustomMaxSize(t *testing.T) {
	validator := &InputValidator{
		MaxSize:           50,
		MaxWhitespaceRatio: 0.8,
		MaxRepetition:     100,
	}

	err := validator.Validate(strings.Repeat("a", 40))
	if err != nil {
		t.Errorf("Input under limit rejected: %v", err)
	}

	err = validator.Validate(strings.Repeat("a", 60))
	if err != ErrInputTooLarge {
		t.Errorf("Input over limit not rejected")
	}
}

func TestInputValidator_DisabledWhitespaceCheck(t *testing.T) {
	validator := &InputValidator{
		MaxSize:            10000,
		MaxWhitespaceRatio: 0,
		MaxRepetition:      0,
	}

	input := strings.Repeat(" ", 100)
	err := validator.Validate(input)
	if err != nil {
		t.Errorf("Whitespace check not disabled: %v", err)
	}
}

func TestInputValidator_DisabledRepetitionCheck(t *testing.T) {
	validator := &InputValidator{
		MaxSize:           10000,
		MaxWhitespaceRatio: 0.8,
		MaxRepetition:      0,
	}

	input := strings.Repeat("a", 200)
	err := validator.Validate(input)
	if err != nil {
		t.Errorf("Repetition check not disabled: %v", err)
	}
}

func TestInputValidator_MixedContent(t *testing.T) {
	validator := NewInputValidator()

	input := "Hello! This is a test.\nWith multiple lines.\n\tAnd some tabs too."
	err := validator.Validate(input)
	if err != nil {
		t.Errorf("Mixed content rejected: %v", err)
	}
}

func TestInputValidator_UnicodeContent(t *testing.T) {
	validator := NewInputValidator()

	unicodeInputs := []string{
		"日本語のテスト",
		"Тест на русском",
		"Ελληνικά κείμενο",
		"Emoji test: 🎉🚀✨",
		"Mixed: Hello 世界 мир",
	}

	for _, input := range unicodeInputs {
		err := validator.Validate(input)
		if err != nil {
			t.Errorf("Unicode content rejected: %s (%v)", input, err)
		}
	}
}

func TestInputValidator_EdgeCases(t *testing.T) {
	validator := NewInputValidator()

	edgeCases := []string{
		"a",
		" ",
		"\n",
		"\t",
		".",
		"!",
		"?",
	}

	for _, input := range edgeCases {
		err := validator.Validate(input)
		if err == ErrInputTooLarge {
			t.Errorf("Edge case rejected as too large: %q", input)
		}
	}
}

func TestInputValidator_Newlines(t *testing.T) {
	validator := NewInputValidator()

	input := "Line 1\nLine 2\nLine 3\nLine 4"
	err := validator.Validate(input)
	if err != nil {
		t.Errorf("Input with newlines rejected: %v", err)
	}
}

func TestInputValidator_TabSeparators(t *testing.T) {
	validator := NewInputValidator()

	input := "col1\tcol2\tcol3\tcol4"
	err := validator.Validate(input)
	if err != nil {
		t.Errorf("Input with tabs rejected: %v", err)
	}
}

func TestValidateInput_Helper(t *testing.T) {
	err := ValidateInput("Hello, world!")
	if err != nil {
		t.Errorf("Helper function rejected valid input: %v", err)
	}

	err = ValidateInput("hello\x00world")
	if err != ErrNullByteDetected {
		t.Errorf("Helper function did not detect null byte")
	}
}

func BenchmarkInputValidator_Validate(b *testing.B) {
	validator := NewInputValidator()
	input := "This is a normal input string for benchmarking."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(input)
	}
}

func BenchmarkInputValidator_Validate_Large(b *testing.B) {
	validator := NewInputValidator()
	input := strings.Repeat("This is a test sentence. ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(input)
	}
}
