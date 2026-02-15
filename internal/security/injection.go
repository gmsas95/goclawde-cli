package security

import (
	"regexp"
	"strings"
)

var (
	ErrPromptInjection = errors.New("potential prompt injection detected")
)

type PromptInjectionDetector struct {
	literalPatterns []string
	regexPatterns   []*regexp.Regexp
}

var injectionLiterals = []string{
	"ignore previous instructions",
	"ignore all previous",
	"disregard all previous",
	"forget all previous",
	"ignore the above",
	"disregard the above",
	"your new instructions",
	"your new task",
	"new directive",
	"override your",
	"system override",
	"jailbreak",
	"simulate",
	"pretend you are",
	"act as if",
	"you are now",
	"developer mode",
	"debug mode",
	"maintenance mode",
}

var injectionRegexes = []string{
	`(?i)ignore\s+(all\s+)?(previous|above)\s+(instructions?|prompts?|rules?|directives?)`,
	`(?i)disregard\s+(all\s+)?(previous|above)\s+(instructions?|prompts?|rules?)`,
	`(?i)forget\s+(all\s+)?(previous|above)\s+(instructions?|context)`,
	`(?i)you\s+are\s+now\s+(a|an)\s+\w+`,
	`(?i)(pretend|act|simulate)\s+(that\s+)?you\s+are`,
	`(?i)(override|bypass)\s+(all\s+)?(rules?|restrictions?|filters?)`,
	`(?i)system:\s*you\s+must`,
	`(?i)<\|.*\|>`,
	`(?i)\[system\].*\[\/system\]`,
	`(?i)###\s*instruction`,
	`(?i)###\s*system`,
}

func NewPromptInjectionDetector() *PromptInjectionDetector {
	detector := &PromptInjectionDetector{
		literalPatterns: make([]string, len(injectionLiterals)),
		regexPatterns:   make([]*regexp.Regexp, 0, len(injectionRegexes)),
	}

	for i, lit := range injectionLiterals {
		detector.literalPatterns[i] = strings.ToLower(lit)
	}

	for _, pattern := range injectionRegexes {
		re, err := regexp.Compile(pattern)
		if err == nil {
			detector.regexPatterns = append(detector.regexPatterns, re)
		}
	}

	return detector
}

func (d *PromptInjectionDetector) Detect(input string) bool {
	inputLower := strings.ToLower(input)

	for _, lit := range d.literalPatterns {
		if strings.Contains(inputLower, lit) {
			return true
		}
	}

	for _, re := range d.regexPatterns {
		if re.MatchString(input) {
			return true
		}
	}

	return false
}

func (d *PromptInjectionDetector) Validate(input string) error {
	if d.Detect(input) {
		return ErrPromptInjection
	}
	return nil
}

func DetectPromptInjection(input string) bool {
	return NewPromptInjectionDetector().Detect(input)
}

func ValidatePrompt(input string) error {
	return NewPromptInjectionDetector().Validate(input)
}
