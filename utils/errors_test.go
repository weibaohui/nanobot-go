package utils

import (
	"errors"
	"testing"
)

func TestWrapError(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		err      error
		expected string
	}{
		{"with error", "query", errors.New("db error"), "query: db error"},
		{"nil error", "query", nil, "error is nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.op, tt.err)
			if tt.err == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if result.Error() != tt.expected {
				t.Errorf("WrapError() = %q, want %q", result.Error(), tt.expected)
			}
		})
	}
}

func TestWrapErrorWithField(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		field    string
		value    any
		err      error
		expected string
	}{
		{"with error", "query", "id", 123, errors.New("not found"), "query id=123: not found"},
		{"nil error", "query", "id", 123, nil, "error is nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapErrorWithField(tt.op, tt.field, tt.value, tt.err)
			if tt.err == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if result.Error() != tt.expected {
				t.Errorf("WrapErrorWithField() = %q, want %q", result.Error(), tt.expected)
			}
		})
	}
}

func TestIsTimeoutError(t *testing.T) {
	if !IsTimeoutError(ErrTimeout) {
		t.Error("ErrTimeout should be a timeout error")
	}
	if IsTimeoutError(errors.New("other")) {
		t.Error("other error should not be a timeout error")
	}
}

func TestIsCanceledError(t *testing.T) {
	if !IsCanceledError(ErrCanceled) {
		t.Error("ErrCanceled should be a canceled error")
	}
	if IsCanceledError(errors.New("other")) {
		t.Error("other error should not be a canceled error")
	}
}
