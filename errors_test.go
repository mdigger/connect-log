// errors_test.go
package connectlog

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"reflect"
	"testing"

	"connectrpc.com/connect"
)

type detailedError struct {
	field  string
	reason string
}

func (e detailedError) Error() string {
	return "validation error"
}

func (e detailedError) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("field", e.field),
		slog.String("reason", e.reason),
	)
}

func TestNewLoggableError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected struct {
			code    connect.Code
			message string
			details bool
		}
	}{
		{
			name:  "nil error",
			input: nil,
		},
		{
			name:  "regular error",
			input: errors.New("something went wrong"),
			expected: struct {
				code    connect.Code
				message string
				details bool
			}{
				code:    connect.CodeUnknown,
				message: "something went wrong",
			},
		},
		{
			name:  "context canceled",
			input: context.Canceled,
			expected: struct {
				code    connect.Code
				message string
				details bool
			}{
				code:    connect.CodeCanceled,
				message: "request canceled",
			},
		},
		{
			name:  "context deadline",
			input: context.DeadlineExceeded,
			expected: struct {
				code    connect.Code
				message string
				details bool
			}{
				code:    connect.CodeDeadlineExceeded,
				message: "request deadline exceeded",
			},
		},
		{
			name: "connect error",
			input: connect.NewError(connect.CodeNotFound,
				errors.New("resource not found")),
			expected: struct {
				code    connect.Code
				message string
				details bool
			}{
				code:    connect.CodeNotFound,
				message: "resource not found",
			},
		},
		{
			name: "error with details",
			input: detailedError{
				field:  "email",
				reason: "invalid format",
			},
			expected: struct {
				code    connect.Code
				message string
				details bool
			}{
				code:    connect.CodeUnknown,
				message: "validation error",
				details: true,
			},
		},
		{
			name: "connect error wrapping context",
			input: connect.NewError(connect.CodeAborted,
				context.Canceled),
			expected: struct {
				code    connect.Code
				message string
				details bool
			}{
				code:    connect.CodeAborted,
				message: "context canceled", // Original message preserved
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newLoggableError(tt.input)
			if tt.input == nil {
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected non-nil error")
			}

			if got := err.Code(); got != tt.expected.code {
				t.Errorf("expected code %v, got %v", tt.expected.code, got)
			}

			if got := err.Message(); got != tt.expected.message {
				t.Errorf("expected message %q, got %q", tt.expected.message, got)
			}

			logValue := err.LogValue()
			groupLen := len(logValue.Group())
			if tt.expected.details {
				if groupLen <= 2 {
					t.Errorf("expected more than 2 attributes in log value, got %d", groupLen)
				}
			} else {
				if groupLen != 2 {
					t.Errorf("expected exactly 2 attributes in log value, got %d", groupLen)
				}
			}
		})
	}
}

func TestLoggableError_LogValue(t *testing.T) {
	t.Run("with details", func(t *testing.T) {
		err := newLoggableError(detailedError{
			field:  "password",
			reason: "too short",
		})

		attrs := err.LogValue().Group()
		if len(attrs) != 4 {
			t.Fatalf("expected 4 attributes, got %d", len(attrs))
		}

		var foundField, foundReason bool
		for _, attr := range attrs {
			switch attr.Key {
			case "field":
				if attr.Value.String() != "password" {
					t.Errorf("expected field 'password', got %q", attr.Value.String())
				}
				foundField = true
			case "reason":
				if attr.Value.String() != "too short" {
					t.Errorf("expected reason 'too short', got %q", attr.Value.String())
				}
				foundReason = true
			}
		}
		if !foundField {
			t.Error("field attribute not found")
		}
		if !foundReason {
			t.Error("reason attribute not found")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		var err *loggableError
		if got := err.LogValue(); !reflect.DeepEqual(got, slog.Value{}) {
			t.Errorf("expected empty slog.Value, got %v", got)
		}
	})
}

func TestErrorReuse(t *testing.T) {
	// Verify we reuse the same instances for context errors
	err1 := newLoggableError(context.Canceled)
	err2 := newLoggableError(context.Canceled)
	if err1 != err2 {
		t.Error("expected same instance for context.Canceled")
	}

	err3 := newLoggableError(context.DeadlineExceeded)
	err4 := newLoggableError(os.ErrDeadlineExceeded)
	if err3 != err4 {
		t.Error("expected same instance for deadline exceeded errors")
	}
}
