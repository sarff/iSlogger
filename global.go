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

// Init initializes with a predefined config
func Init(config Config) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if defaultLogger != nil {
		defaultLogger.Close()
	}

	var err error
	defaultLogger, err = New(config)
	return err
}

// InitDefault initializes the default config
func InitDefault() error {
	return Init(DefaultConfig())
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return defaultLogger
}

// SetGlobalLogger sets a custom logger as the global instance
func SetGlobalLogger(logger *Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()

	if defaultLogger != nil {
		defaultLogger.Close()
	}
	defaultLogger = logger
}

// Debug logs a debug message using the global logger
func Debug(msg string, args ...any) {
	globalMu.RLock()
	logger := defaultLogger
	globalMu.RUnlock()

	if logger != nil {
		logger.Debug(msg, args...)
	}
}

// Info logs an info message using the global logger
func Info(msg string, args ...any) {
	globalMu.RLock()
	logger := defaultLogger
	globalMu.RUnlock()

	if logger != nil {
		logger.Info(msg, args...)
	}
}

// Warn logs a warning message using the global logger
func Warn(msg string, args ...any) {
	globalMu.RLock()
	logger := defaultLogger
	globalMu.RUnlock()

	if logger != nil {
		logger.Warn(msg, args...)
	}
}

// Error logs an error message using the global logger
func Error(msg string, args ...any) {
	globalMu.RLock()
	logger := defaultLogger
	globalMu.RUnlock()

	if logger != nil {
		logger.Error(msg, args...)
	}
}

// With creates a logger with additional attributes using the global logger
func With(args ...any) *Logger {
	globalMu.RLock()
	logger := defaultLogger
	globalMu.RUnlock()

	if logger != nil {
		return logger.With(args...)
	}
	return nil
}

// WithContext creates a logger with context using the global logger
func WithContext(ctx context.Context) *Logger {
	globalMu.RLock()
	logger := defaultLogger
	globalMu.RUnlock()

	if logger != nil {
		return logger.WithContext(ctx)
	}
	return nil
}

// SetLevel changes the log level of the global logger
func SetLevel(level slog.Level) error {
	globalMu.RLock()
	logger := defaultLogger
	globalMu.RUnlock()

	if logger != nil {
		return logger.SetLevel(level)
	}
	return nil
}

// Flush flushes all buffers of the global logger
func Flush() error {
	globalMu.RLock()
	logger := defaultLogger
	globalMu.RUnlock()

	if logger != nil {
		return logger.Flush()
	}
	return nil
}

// Close closes the global logger
func Close() error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if defaultLogger != nil {
		err := defaultLogger.Close()
		defaultLogger = nil
		return err
	}
	return nil
}

// CleanupNow performs immediate cleanup using the global logger
func CleanupNow() {
	globalMu.RLock()
	logger := defaultLogger
	globalMu.RUnlock()

	if logger != nil {
		logger.CleanupNow()
	}
}

// GetLogFiles returns list of log files using the global logger
func GetLogFiles() ([]string, error) {
	globalMu.RLock()
	logger := defaultLogger
	globalMu.RUnlock()

	if logger != nil {
		return logger.GetLogFiles()
	}
	return nil, nil
}
