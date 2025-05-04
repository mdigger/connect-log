// errors.go
package connectlog

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"connectrpc.com/connect"
)

// loggableError wraps a connect.Error and implements slog.LogValuer
// for structured logging of errors with Connect codes and details.
type loggableError struct {
	*connect.Error
}

// Predefined errors for common context cases to avoid allocations
var (
	errCanceled = &loggableError{
		Error: connect.NewError(connect.CodeCanceled, errors.New("request canceled")),
	}
	errDeadline = &loggableError{
		Error: connect.NewError(connect.CodeDeadlineExceeded, errors.New("request deadline exceeded")),
	}
)

// newLoggableError creates a LoggableError from any error value.
// It preserves connect.Error values, handles context errors specially,
// and wraps all other errors as Unknown.
func newLoggableError(err error) *loggableError {
	if err == nil {
		return nil
	}

	// Preserve existing connect.Error values
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return &loggableError{Error: connectErr}
	}

	// Handle context errors with predefined wrappers
	switch {
	case errors.Is(err, context.Canceled):
		return errCanceled
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded):
		return errDeadline
	}

	// Wrap all other errors as Unknown
	return &loggableError{
		Error: connect.NewError(connect.CodeUnknown, err),
	}
}

// LogValue implements slog.LogValuer, providing structured error details.
// It always includes the error code and message, and adds any additional
// details from the original error if it implements slog.LogValuer.
func (e *loggableError) LogValue() slog.Value {
	if e == nil {
		return slog.Value{}
	}

	attrs := []slog.Attr{
		slog.String("code", e.Code().String()),
		slog.String("message", e.Message()),
	}

	// Extract details from the original error (bypassing connect.Error wrapper)
	origErr := e.Unwrap()
	if origErr == nil {
		origErr = e.Error
	}

	var logValuer slog.LogValuer
	if errors.As(origErr, &logValuer) {
		logValue := logValuer.LogValue()
		switch logValue.Kind() {
		case slog.KindGroup:
			attrs = append(attrs, logValue.Group()...)
		default:
			attrs = append(attrs, slog.Any("details", logValue))
		}
	}

	return slog.GroupValue(attrs...)
}
