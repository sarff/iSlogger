package iSlogger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// levelFilterWriter filters logs by level
type levelFilterWriter struct {
	writer   io.Writer
	maxLevel slog.Level // Maximum level to write (inclusive)
}

func (lfw *levelFilterWriter) Write(p []byte) (n int, err error) {
	logStr := string(p)

	if strings.Contains(logStr, "level=WARN") ||
		strings.Contains(logStr, "level=ERROR") ||
		strings.Contains(logStr, `"level":"WARN"`) ||
		strings.Contains(logStr, `"level":"ERROR"`) {
		// Don't write WARN/ERROR to info file
		return len(p), nil
	}

	// Write DEBUG/INFO to info file
	return lfw.writer.Write(p)
}

// Logger wraps slog.Logger with file rotation
type Logger struct {
	config      Config
	infoLogger  *slog.Logger
	errorLogger *slog.Logger
	infoFile    *os.File
	errorFile   *os.File
	currentDate string
	mu          sync.RWMutex
}

// New creates a new Logger instance
func New(config Config) (*Logger, error) {
	// Set defaults if empty
	if config.LogDir == "" {
		config.LogDir = "logs"
	}
	if config.AppName == "" {
		config.AppName = "app"
	}
	if config.RetentionDays <= 0 {
		config.RetentionDays = 7
	}
	if config.TimeFormat == "" {
		config.TimeFormat = time.RFC3339
	}

	// Create log directory
	if err := os.MkdirAll(config.LogDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	l := &Logger{
		config:      config,
		currentDate: time.Now().Format("2006-01-02"),
	}

	if err := l.initLoggers(); err != nil {
		return nil, err
	}

	// Start cleanup
	go l.startCleanupRoutine()

	return l, nil
}

// initLoggers initializes both info and error loggers
func (l *Logger) initLoggers() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Close existing files if open
	if l.infoFile != nil {
		l.infoFile.Close()
	}
	if l.errorFile != nil {
		l.errorFile.Close()
	}

	var err error
	today := time.Now().Format("2006-01-02")

	// Open info log file
	infoPath := filepath.Join(l.config.LogDir, fmt.Sprintf("%s_%s.log", l.config.AppName, today))
	l.infoFile, err = os.OpenFile(infoPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open info log file: %w", err)
	}

	// Open error log file
	errorPath := filepath.Join(l.config.LogDir, fmt.Sprintf("%s_error_%s.log", l.config.AppName, today))
	l.errorFile, err = os.OpenFile(errorPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open error log file: %w", err)
	}

	// Create multi-writers for console + file output
	infoFileWriter := &levelFilterWriter{
		writer:   l.infoFile,
		maxLevel: slog.LevelInfo, // Only DEBUG and INFO
	}

	infoWriter := io.MultiWriter(os.Stdout, infoFileWriter)
	errorWriter := io.MultiWriter(os.Stderr, l.errorFile)

	// slog options
	opts := &slog.HandlerOptions{
		AddSource: l.config.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Custom time format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format(l.config.TimeFormat)),
				}
			}
			return a
		},
	}

	// Set log level based on debug mode
	if l.config.Debug {
		opts.Level = slog.LevelDebug
	} else {
		opts.Level = slog.LevelWarn
	}

	// Create base handlers
	var infoHandler, errorHandler slog.Handler
	if l.config.JSONFormat {
		infoHandler = slog.NewJSONHandler(infoWriter, opts)
		errorHandler = slog.NewJSONHandler(errorWriter, opts)
	} else {
		infoHandler = slog.NewTextHandler(infoWriter, opts)
		errorHandler = slog.NewTextHandler(errorWriter, opts)
	}

	// Wrap with filtered handlers
	filteredInfoHandler := newFilteredHandler(infoHandler, l.config.Filters)
	filteredErrorHandler := newFilteredHandler(errorHandler, l.config.Filters)

	l.infoLogger = slog.New(filteredInfoHandler)
	l.errorLogger = slog.New(filteredErrorHandler)

	l.currentDate = today
	return nil
}

// checkDateRotation checks if we need to rotate log files
func (l *Logger) checkDateRotation() {
	today := time.Now().Format("2006-01-02")
	if today != l.currentDate {
		l.initLoggers() // This will handle the rotation
	}
}

// Debug logs debug level message
func (l *Logger) Debug(msg string, args ...any) {
	l.checkDateRotation()
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.infoLogger.Debug(msg, args...)
}

// Info logs info level message
func (l *Logger) Info(msg string, args ...any) {
	l.checkDateRotation()
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.infoLogger.Info(msg, args...)
}

// Warn logs warning level message
func (l *Logger) Warn(msg string, args ...any) {
	l.checkDateRotation()
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.infoLogger.Warn(msg, args...)
	l.errorLogger.Warn(msg, args...)
}

// Error logs error level message
func (l *Logger) Error(msg string, args ...any) {
	l.checkDateRotation()
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.infoLogger.Error(msg, args...)
	l.errorLogger.Error(msg, args...)
}

// With creates a logger with additional attributes
func (l *Logger) With(args ...any) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newLogger := &Logger{
		config:      l.config,
		infoFile:    l.infoFile,
		errorFile:   l.errorFile,
		currentDate: l.currentDate,
		infoLogger:  l.infoLogger.With(args...),
		errorLogger: l.errorLogger.With(args...),
	}
	return newLogger
}

// WithContext creates a logger with context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newLogger := &Logger{
		config:      l.config,
		infoFile:    l.infoFile,
		errorFile:   l.errorFile,
		currentDate: l.currentDate,
		infoLogger:  l.infoLogger.WithGroup("context"),
		errorLogger: l.errorLogger.WithGroup("context"),
	}
	return newLogger
}

// SetLevel changes the log level dynamically
func (l *Logger) SetLevel(level slog.Level) error {
	l.config.Debug = level == slog.LevelDebug
	return l.initLoggers()
}

// SetDebug enables or disables debug mode
func (l *Logger) SetDebug(debug bool) error {
	l.config.Debug = debug
	return l.initLoggers()
}

// Close closes the logger and its files
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errs []error
	if l.infoFile != nil {
		if err := l.infoFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if l.errorFile != nil {
		if err := l.errorFile.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing files: %v", errs)
	}
	return nil
}
