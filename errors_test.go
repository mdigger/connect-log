// errors_test.go
package connectlog

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
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
				require.Nil(t, err)
				return
			}

			require.NotNil(t, err)
			require.Equal(t, tt.expected.code, err.Code())
			require.Equal(t, tt.expected.message, err.Message())

			logValue := err.LogValue()
			if tt.expected.details {
				require.Greater(t, len(logValue.Group()), 2) // code + message + details
			} else {
				require.Len(t, logValue.Group(), 2) // just code + message
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
		require.Len(t, attrs, 4) // code + message + field + reason

		var foundField, foundReason bool
		for _, attr := range attrs {
			switch attr.Key {
			case "field":
				require.Equal(t, "password", attr.Value.String())
				foundField = true
			case "reason":
				require.Equal(t, "too short", attr.Value.String())
				foundReason = true
			}
		}
		require.True(t, foundField)
		require.True(t, foundReason)
	})

	t.Run("nil error", func(t *testing.T) {
		var err *loggableError
		require.Equal(t, slog.Value{}, err.LogValue())
	})
}

func TestErrorReuse(t *testing.T) {
	// Verify we reuse the same instances for context errors
	require.Same(t,
		newLoggableError(context.Canceled),
		newLoggableError(context.Canceled))
	require.Same(t,
		newLoggableError(context.DeadlineExceeded),
		newLoggableError(os.ErrDeadlineExceeded))
}
