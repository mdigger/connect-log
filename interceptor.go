package connectlog

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
)

// ContextLogFunc defines a function type that extracts additional log attributes from context.
type ContextLogFunc func(context.Context) []slog.Attr

// LoggingInterceptor implements ConnectRPC interceptors for structured logging.
type LoggingInterceptor struct {
	logger        *slog.Logger
	redactHeaders []string
	contextLogFn  ContextLogFunc
}

// var _ connect.Interceptor = (*LoggingInterceptor)(nil)

// NewLoggingInterceptor creates a new logging interceptor instance.
func NewLoggingInterceptor(opts ...Option) *LoggingInterceptor {
	options := Options{
		Logger:        slog.Default(),
		RedactHeaders: []string{"authorization", "token"},
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &LoggingInterceptor{
		logger:        options.Logger,
		redactHeaders: options.RedactHeaders,
		contextLogFn:  options.ContextLogFn,
	}
}

// initRequestLogger initializes the base logger with common request attributes
func (i *LoggingInterceptor) initRequestLogger(ctx context.Context, spec connect.Spec, peer connect.Peer) *slog.Logger {
	procedure := strings.TrimPrefix(spec.Procedure, "/")
	idx := strings.Index(procedure, "/")
	service, method := procedure[:idx], procedure[idx+1:]

	logger := i.logger.With(
		slog.String("service", service),
		slog.String("method", method),
		slog.String("protocol", peer.Protocol),
		slog.String("addr", peer.Addr),
	)

	// Add custom fields from context if configured
	if i.contextLogFn != nil {
		for _, attr := range i.contextLogFn(ctx) {
			logger = logger.With(attr)
		}
	}

	return logger
}

// WrapUnary implements unary request/response logging middleware.
func (i *LoggingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		logger := i.initRequestLogger(ctx, req.Spec(), req.Peer())

		// Debug logging for request start with headers and body
		if logger.Enabled(ctx, slog.LevelDebug) {
			headers := redactHeadersMap(req.Header(), i.redactHeaders)
			logger.Debug("request started",
				slog.Any("request", req.Any()),
				slog.Any("headers", headers),
			)
		}

		// Execute the RPC call
		res, err := next(ctx, req)

		// Prepare log attributes
		logAttrs := []any{
			slog.Duration("duration", time.Since(start)),
		}

		// Add payload sizes if available
		if reqSize := calculateSize(req.Any()); reqSize >= 0 {
			logAttrs = append(logAttrs, slog.Int("request_size", reqSize))
		}

		if err != nil {
			// Handle different error types
			connErr := newLoggableError(err)
			logAttrs = append(logAttrs, slog.Any("error", connErr))

			// Determine log level based on error type
			if connErr.Code() < connect.CodeInternal {
				logger.Warn("request failed", logAttrs...)
			} else {
				logger.Error("request failed", logAttrs...)
			}
		} else {
			// Debug logging for response with headers
			if logger.Enabled(ctx, slog.LevelDebug) {
				headers := redactHeadersMap(res.Header(), i.redactHeaders)
				logger.Debug("response completed",
					slog.Any("response", res.Any()),
					slog.Any("headers", headers),
				)
			}

			// Success case logging
			if resSize := calculateSize(res.Any()); resSize >= 0 {
				logAttrs = append(logAttrs, slog.Int("response_size", resSize))
			}

			logger.Info("request completed", logAttrs...)
		}

		return res, err
	}
}

// WrapStreamingHandler implements streaming request logging middleware.
func (i *LoggingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := time.Now()
		logger := i.initRequestLogger(ctx, conn.Spec(), conn.Peer())

		// Debug logging for stream start with headers
		if logger.Enabled(ctx, slog.LevelDebug) {
			headers := redactHeadersMap(conn.RequestHeader(), i.redactHeaders)
			logger.Debug("stream started",
				slog.Any("headers", headers),
			)
		}

		// Wrap the connection to log messages
		wrappedConn := newLoggedStreamConn(ctx, conn, logger)

		// Execute the stream
		err := next(ctx, wrappedConn)

		logAttrs := []any{
			slog.Group("messages",
				slog.Int("sent", wrappedConn.sentCount),
				slog.Int("received", wrappedConn.receivedCount),
			),
			slog.Duration("duration", time.Since(start)),
		}

		if err != nil && !errors.Is(err, io.EOF) {
			connErr := newLoggableError(err)
			logAttrs = append(logAttrs, slog.Any("error", connErr))

			if connErr.Code() < connect.CodeInternal {
				logger.Warn("stream failed", logAttrs...)
			} else {
				logger.Error("stream failed", logAttrs...)
			}
		} else {
			logger.Info("stream completed", logAttrs...)
		}

		return err
	}
}
