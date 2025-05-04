# ConnectRPC Logging Interceptor

A structured logging interceptor for ConnectRPC with rich features:

## Features

- Request/response logging with timing
- Header redaction for sensitive data
- Error classification and logging
- Stream message tracking
- Context-aware logging

## Installation

```sh
go get github.com/mdigger/connectlog
```

## Usage

```go
import (
	"log/slog"
	"github.com/mdigger/connectlog"
)

// Basic usage
interceptor := connectlog.NewLoggingInterceptor()

// With options
interceptor := connectlog.NewLoggingInterceptor(
	connectlog.WithLogger(slog.Default()),
	connectlog.WithRedactHeaders([]string{"token"}),
	connectlog.WithContextLogFn(func(ctx context.Context) []slog.Attr {
		return []slog.Attr{slog.String("trace_id", "123")}
	}),
)

client := pingv1connect.NewPingServiceClient(
	http.DefaultClient,
	"http://localhost:8080",
	connect.WithInterceptors(interceptor),
)
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithLogger` | Custom slog logger | slog.Default() |
| `WithRedactHeaders` | Headers to redact | ["authorization", "token"] |
| `WithContextLogFn` | Function to extract context fields | nil |

## Log Format

Logs include:
- Service/method names
- Peer information
- Duration
- Payload sizes
- Error codes and messages
- Stream message counts

## Best Practices

1. Use debug level for payload logging in development
2. Redact all sensitive headers
3. Add request-scoped fields via context
