package security

import (
	"fmt"
)

type SecurityGuard struct {
	shell     *ShellSecurityConfig
	input     *InputValidator
	secrets   *SecretScanner
	injection *PromptInjectionDetector
}

func NewSecurityGuard() *SecurityGuard {
	return &SecurityGuard{
		shell:     NewShellSecurityConfig(),
		input:     NewInputValidator(),
		secrets:   NewSecretScanner(),
		injection: NewPromptInjectionDetector(),
	}
}

type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

func (v *ValidationResult) AddError(err string) {
	v.Errors = append(v.Errors, err)
	v.Valid = false
}

func (v *ValidationResult) AddWarning(warn string) {
	v.Warnings = append(v.Warnings, warn)
}

func (g *SecurityGuard) ValidateUserInput(input string) *ValidationResult {
	result := &ValidationResult{Valid: true, Errors: []string{}, Warnings: []string{}}

	if err := g.input.Validate(input); err != nil {
		result.AddError(fmt.Sprintf("Input validation failed: %v", err))
	}

	if g.injection.Detect(input) {
		result.AddWarning("Potential prompt injection patterns detected")
	}

	if secrets := g.secrets.Scan(input); len(secrets) > 0 {
		for _, s := range secrets {
			result.AddWarning(fmt.Sprintf("Potential secret detected: %s at position %d", s.Type, s.Start))
		}
	}

	return result
}

func (g *SecurityGuard) ValidateCommand(command string) error {
	return g.shell.ValidateCommand(command)
}

func (g *SecurityGuard) ValidatePath(path, workspace string) (*SafePath, error) {
	return ValidatePathInWorkspace(path, workspace)
}

func (g *SecurityGuard) SanitizeInput(input string) string {
	return g.secrets.Redact(input)
}

func (g *SecurityGuard) CheckPromptSafety(prompt string) *PromptSafetyResult {
	result := &PromptSafetyResult{
		Safe:      true,
		RiskLevel: "low",
		Flags:     []string{},
	}

	if g.injection.Detect(prompt) {
		result.Safe = false
		result.RiskLevel = "high"
		result.Flags = append(result.Flags, "prompt_injection")
	}

	if secrets := g.secrets.Scan(prompt); len(secrets) > 0 {
		result.RiskLevel = "medium"
		result.Flags = append(result.Flags, "contains_secrets")
		for _, s := range secrets {
			result.Flags = append(result.Flags, fmt.Sprintf("secret:%s", s.Type))
		}
	}

	return result
}

type PromptSafetyResult struct {
	Safe      bool
	RiskLevel string
	Flags     []string
}

var DefaultGuard = NewSecurityGuard()

func ValidateUserInput(input string) *ValidationResult {
	return DefaultGuard.ValidateUserInput(input)
}

func SanitizeInput(input string) string {
	return DefaultGuard.SanitizeInput(input)
}

func CheckPromptSafety(prompt string) *PromptSafetyResult {
	return DefaultGuard.CheckPromptSafety(prompt)
}
