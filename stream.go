package connectlog

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
)

// loggedStreamConn wraps a streaming connection to track and log messages
type loggedStreamConn struct {
	connect.StreamingHandlerConn
	logger        *slog.Logger
	ctx           context.Context
	sentCount     int
	receivedCount int
	debugEnabled  bool
}

func newLoggedStreamConn(ctx context.Context, conn connect.StreamingHandlerConn, logger *slog.Logger) *loggedStreamConn {
	return &loggedStreamConn{
		StreamingHandlerConn: conn,
		logger:               logger,
		ctx:                  ctx,
		debugEnabled:         logger.Enabled(ctx, slog.LevelDebug),
	}
}

func (c *loggedStreamConn) Send(msg any) error {
	if err := c.StreamingHandlerConn.Send(msg); err != nil {
		return err
	}
	c.sentCount++
	if c.debugEnabled {
		c.logger.Debug("stream message sent",
			slog.Int("number", c.sentCount),
			slog.Int("size", calculateSize(msg)),
			slog.Any("response", msg),
		)
	}
	return nil
}

func (c *loggedStreamConn) Receive(msg any) error {
	if err := c.StreamingHandlerConn.Receive(msg); err != nil {
		return err
	}

	c.receivedCount++
	if c.debugEnabled {
		c.logger.Debug("stream message received",
			slog.Int("number", c.receivedCount),
			slog.Int("size", calculateSize(msg)),
			slog.Any("receive", msg),
		)
	}

	return nil
}
