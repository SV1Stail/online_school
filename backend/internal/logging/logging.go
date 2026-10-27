package logging

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
)

func New(level string, pretty bool) zerolog.Logger {
	parsedLevel, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		parsedLevel = zerolog.InfoLevel
	}

	var output io.Writer = os.Stdout
	if pretty {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	return zerolog.New(output).
		Level(parsedLevel).
		With().
		Timestamp().
		Str("service", "github.com/SV1Stail/online_school-api").
		Logger()
}

type PGXLogger struct {
	logger zerolog.Logger
}

func NewPGXLogger(logger zerolog.Logger) *PGXLogger {
	return &PGXLogger{logger: logger.With().Str("component", "postgres").Logger()}
}

func (l *PGXLogger) Log(ctx context.Context, level tracelog.LogLevel, message string, data map[string]any) {
	safeData := make(map[string]any, len(data))
	for key, value := range data {
		if key == "args" {
			continue
		}
		safeData[key] = value
	}

	event := l.event(level).Fields(safeData).Str("db_event", message)
	if requestID, ok := ctx.Value(requestIDKey{}).(string); ok {
		event = event.Str("request_id", requestID)
	}
	event.Msg("database operation")
}

func (l *PGXLogger) event(level tracelog.LogLevel) *zerolog.Event {
	switch level {
	case tracelog.LogLevelTrace, tracelog.LogLevelDebug:
		return l.logger.Debug()
	case tracelog.LogLevelInfo:
		return l.logger.Info()
	case tracelog.LogLevelWarn:
		return l.logger.Warn()
	case tracelog.LogLevelError:
		return l.logger.Error()
	default:
		return l.logger.Info()
	}
}

type requestIDKey struct{}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}
