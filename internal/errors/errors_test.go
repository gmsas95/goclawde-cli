package errors

import (
	"fmt"
	"strings"
	"testing"
)

func TestAppError(t *testing.T) {
	err := New("TEST_001", "test error")

	if err.Code != "TEST_001" {
		t.Errorf("expected code TEST_001, got %s", err.Code)
	}
	if err.Message != "test error" {
		t.Errorf("expected message 'test error', got %s", err.Message)
	}
}

func TestAppErrorWithCause(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := New("TEST_001", "test error", cause)

	if err.Cause != cause {
		t.Errorf("expected cause to be set")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "underlying error") {
		t.Errorf("expected error string to contain cause, got %s", errStr)
	}
}

func TestAppErrorUnwrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := New("TEST_001", "test error", cause)

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("expected unwrap to return cause")
	}
}

func TestIsAppError(t *testing.T) {
	appErr := New("TEST_001", "test error")
	stdErr := fmt.Errorf("standard error")

	if !IsAppError(appErr) {
		t.Error("expected IsAppError to return true for AppError")
	}
	if IsAppError(stdErr) {
		t.Error("expected IsAppError to return false for standard error")
	}
}

func TestGetCode(t *testing.T) {
	appErr := New("TEST_001", "test error")
	stdErr := fmt.Errorf("standard error")

	if GetCode(appErr) != "TEST_001" {
		t.Errorf("expected code TEST_001, got %s", GetCode(appErr))
	}
	if GetCode(stdErr) != "UNKNOWN" {
		t.Errorf("expected code UNKNOWN for standard error, got %s", GetCode(stdErr))
	}
}

func TestWrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := Wrap(cause, "WRAP_001", "wrapped error")

	if err.Code != "WRAP_001" {
		t.Errorf("expected code WRAP_001, got %s", err.Code)
	}
	if err.Cause != cause {
		t.Error("expected cause to be set")
	}
}

func TestPredefinedErrors(t *testing.T) {
	if ErrConfigNotFound.Code != "CONFIG_001" {
		t.Errorf("unexpected code for ErrConfigNotFound")
	}
	if ErrProviderNotConfigured.Code != "LLM_001" {
		t.Errorf("unexpected code for ErrProviderNotConfigured")
	}
	if ErrUnauthorized.Code != "AUTH_001" {
		t.Errorf("unexpected code for ErrUnauthorized")
	}
}
