package errors

import "fmt"

type AppError struct {
	Code    string
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func New(code, message string, cause ...error) *AppError {
	var c error
	if len(cause) > 0 {
		c = cause[0]
	}
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   c,
	}
}

var (
	ErrConfigNotFound = &AppError{Code: "CONFIG_001", Message: "configuration not found"}
	ErrConfigInvalid  = &AppError{Code: "CONFIG_002", Message: "invalid configuration"}

	ErrProviderNotConfigured = &AppError{Code: "LLM_001", Message: "no LLM provider configured"}
	ErrProviderUnavailable   = &AppError{Code: "LLM_002", Message: "LLM provider unavailable"}
	ErrRateLimited           = &AppError{Code: "LLM_003", Message: "rate limit exceeded"}

	ErrMemoryNotFound  = &AppError{Code: "MEMORY_001", Message: "memory not found"}
	ErrMemoryCorrupted = &AppError{Code: "MEMORY_002", Message: "memory corrupted"}

	ErrConversationNotFound = &AppError{Code: "CONV_001", Message: "conversation not found"}

	ErrSkillNotFound  = &AppError{Code: "SKILL_001", Message: "skill not found"}
	ErrSkillExecution = &AppError{Code: "SKILL_002", Message: "skill execution failed"}

	ErrChannelNotConfigured = &AppError{Code: "CHAN_001", Message: "channel not configured"}
	ErrChannelUnavailable   = &AppError{Code: "CHAN_002", Message: "channel unavailable"}

	ErrUnauthorized = &AppError{Code: "AUTH_001", Message: "unauthorized"}
	ErrForbidden    = &AppError{Code: "AUTH_002", Message: "forbidden"}

	ErrNotFound   = &AppError{Code: "GEN_001", Message: "resource not found"}
	ErrBadRequest = &AppError{Code: "GEN_002", Message: "bad request"}
	ErrInternal   = &AppError{Code: "GEN_003", Message: "internal error"}
)

func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

func GetCode(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return "UNKNOWN"
}

func Wrap(err error, code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}
