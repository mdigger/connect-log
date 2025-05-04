package connectlog

import (
	"log/slog"
)

type Options struct {
	Logger        *slog.Logger
	RedactHeaders []string
	ContextLogFn  ContextLogFunc
}

type Option func(*Options)

func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

func WithRedactHeaders(headers []string) Option {
	return func(o *Options) {
		o.RedactHeaders = headers
	}
}

func WithContextLogFn(fn ContextLogFunc) Option {
	return func(o *Options) {
		o.ContextLogFn = fn
	}
}
