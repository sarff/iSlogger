package iSlogger

import (
	"context"
	"log/slog"
	"sync"
)

var (
	defaultLogger *Logger
	globalMu      sync.RWMutex
)

// Init installs a newly created global logger without discarding a working one on failure.
func Init(config Config) error {
	logger, err := New(config)
	if err != nil {
		return err
	}
	globalMu.Lock()
	old := defaultLogger
	defaultLogger = logger
	globalMu.Unlock()
	if old != nil && old != logger {
		return old.Close()
	}
	return nil
}

// InitDefault initializes the global logger with DefaultConfig.
func InitDefault() error { return Init(DefaultConfig()) }

// GetGlobalLogger returns the installed global logger, or nil when uninitialized.
func GetGlobalLogger() *Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return defaultLogger
}

// SetGlobalLogger replaces and closes the previous global logger.
func SetGlobalLogger(logger *Logger) {
	globalMu.Lock()
	old := defaultLogger
	if old == logger {
		globalMu.Unlock()
		return
	}
	defaultLogger = logger
	globalMu.Unlock()
	if old != nil {
		_ = old.Close()
	}
}

func globalLogger() *Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return defaultLogger
}

// Debug logs through the global logger at slog.LevelDebug.
func Debug(msg string, args ...any) {
	if l := globalLogger(); l != nil {
		l.log(l.context(), slog.LevelDebug, msg, args...)
	}
}

// Info logs through the global logger at slog.LevelInfo.
func Info(msg string, args ...any) {
	if l := globalLogger(); l != nil {
		l.log(l.context(), slog.LevelInfo, msg, args...)
	}
}

// Warn logs through the global logger at slog.LevelWarn.
func Warn(msg string, args ...any) {
	if l := globalLogger(); l != nil {
		l.log(l.context(), slog.LevelWarn, msg, args...)
	}
}

// Error logs through the global logger at slog.LevelError.
func Error(msg string, args ...any) {
	if l := globalLogger(); l != nil {
		l.log(l.context(), slog.LevelError, msg, args...)
	}
}

// DebugContext logs through the global logger with ctx.
func DebugContext(ctx context.Context, msg string, args ...any) {
	if l := globalLogger(); l != nil {
		l.log(nonNilContext(ctx), slog.LevelDebug, msg, args...)
	}
}

// InfoContext logs through the global logger with ctx.
func InfoContext(ctx context.Context, msg string, args ...any) {
	if l := globalLogger(); l != nil {
		l.log(nonNilContext(ctx), slog.LevelInfo, msg, args...)
	}
}

// WarnContext logs through the global logger with ctx.
func WarnContext(ctx context.Context, msg string, args ...any) {
	if l := globalLogger(); l != nil {
		l.log(nonNilContext(ctx), slog.LevelWarn, msg, args...)
	}
}

// ErrorContext logs through the global logger with ctx.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	if l := globalLogger(); l != nil {
		l.log(nonNilContext(ctx), slog.LevelError, msg, args...)
	}
}

// With returns a global logger handle with additional attributes.
func With(args ...any) *Logger {
	if l := globalLogger(); l != nil {
		return l.With(args...)
	}
	return nil
}

// WithContext is retained for compatibility.
// Deprecated: use the level-specific context functions.
func WithContext(ctx context.Context) *Logger {
	if l := globalLogger(); l != nil {
		return l.WithContext(ctx)
	}
	return nil
}

// SetLevel changes the global logger's minimum level.
func SetLevel(level slog.Level) error {
	if l := globalLogger(); l != nil {
		return l.SetLevel(level)
	}
	return nil
}

// Flush writes pending global logger buffers.
func Flush() error {
	if l := globalLogger(); l != nil {
		return l.Flush()
	}
	return nil
}

// Reopen reopens the global logger's current daily files.
func Reopen() error {
	if l := globalLogger(); l != nil {
		return l.Reopen()
	}
	return nil
}

// Close removes and closes the global logger.
func Close() error {
	globalMu.Lock()
	logger := defaultLogger
	defaultLogger = nil
	globalMu.Unlock()
	if logger != nil {
		return logger.Close()
	}
	return nil
}

// Cleanup synchronously removes expired files for the global logger.
func Cleanup() error {
	if l := globalLogger(); l != nil {
		return l.Cleanup()
	}
	return nil
}

// CleanupNow is retained for compatibility.
// Deprecated: use Cleanup to receive filesystem errors.
func CleanupNow() { _ = Cleanup() }

// GetLogFiles returns files belonging to the global logger.
func GetLogFiles() ([]string, error) {
	if l := globalLogger(); l != nil {
		return l.GetLogFiles()
	}
	return nil, nil
}
