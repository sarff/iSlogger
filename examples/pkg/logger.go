package pkg

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/sarff/iSlogger"
)

// Init configures and installs the process-wide logger.
func Init() error {
	config := iSlogger.DefaultConfig().
		WithAppName("Logger").
		WithLogDir("logs").
		WithRetentionDays(7).
		WithTimeFormat("2006-01-02 15:04:05").
		WithConsoleOutput(true).
		WithLogLevel(slog.LevelInfo).
		WithJSONFormat(true).
		WithoutBuffering()

	return iSlogger.Init(config)
}

func parseLogLevel(value string) (slog.Level, error) {
	var level slog.Level
	err := level.UnmarshalText([]byte(strings.TrimSpace(value)))
	return level, err
}

// SetLevel updates the minimum level of the initialized logger.
func SetLevel(value string) error {
	level, err := parseLogLevel(value)
	if err != nil {
		return err
	}
	return iSlogger.SetLevel(level)
}

// Debug writes a debug record or falls back to the standard logger.
func Debug(msg string, args ...any) {
	if logger := iSlogger.GetGlobalLogger(); logger != nil {
		logger.Debug(msg, args...)
		return
	}
	slog.Debug(msg, args...)
}

// Info writes an info record or falls back to the standard logger.
func Info(msg string, args ...any) {
	if logger := iSlogger.GetGlobalLogger(); logger != nil {
		logger.Info(msg, args...)
		return
	}
	slog.Info(msg, args...)
}

// Warn writes a warning record or falls back to the standard logger.
func Warn(msg string, args ...any) {
	if logger := iSlogger.GetGlobalLogger(); logger != nil {
		logger.Warn(msg, args...)
		return
	}
	slog.Warn(msg, args...)
}

// Error writes an error record or falls back to the standard logger.
func Error(msg string, args ...any) {
	if logger := iSlogger.GetGlobalLogger(); logger != nil {
		logger.Error(msg, args...)
		return
	}
	slog.Error(msg, args...)
}

// DebugContext writes a contextual debug record.
func DebugContext(ctx context.Context, msg string, args ...any) {
	ctx = nonNilContext(ctx)
	if logger := iSlogger.GetGlobalLogger(); logger != nil {
		logger.DebugContext(ctx, msg, args...)
		return
	}
	slog.DebugContext(ctx, msg, args...)
}

// InfoContext writes a contextual info record.
func InfoContext(ctx context.Context, msg string, args ...any) {
	ctx = nonNilContext(ctx)
	if logger := iSlogger.GetGlobalLogger(); logger != nil {
		logger.InfoContext(ctx, msg, args...)
		return
	}
	slog.InfoContext(ctx, msg, args...)
}

// WarnContext writes a contextual warning record.
func WarnContext(ctx context.Context, msg string, args ...any) {
	ctx = nonNilContext(ctx)
	if logger := iSlogger.GetGlobalLogger(); logger != nil {
		logger.WarnContext(ctx, msg, args...)
		return
	}
	slog.WarnContext(ctx, msg, args...)
}

// ErrorContext writes a contextual error record.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	ctx = nonNilContext(ctx)
	if logger := iSlogger.GetGlobalLogger(); logger != nil {
		logger.ErrorContext(ctx, msg, args...)
		return
	}
	slog.ErrorContext(ctx, msg, args...)
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// Flush writes pending buffered records.
func Flush() error { return iSlogger.Flush() }

// Reopen reopens the current log files.
func Reopen() error { return iSlogger.Reopen() }

// Cleanup removes expired log files synchronously.
func Cleanup() error { return iSlogger.Cleanup() }

// Fatal writes a fatal record, closes the logger, and exits with status 1.
func Fatal(msg string, args ...any) {
	Error("FATAL: "+msg, args...)
	if err := Close(); err != nil {
		slog.Error("failed to close logger", "err", err)
	}
	os.Exit(1)
}

// Close removes and closes the global logger.
func Close() error { return iSlogger.Close() }
