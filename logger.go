package iSlogger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Logger writes structured records to daily files and optional console streams.
// Values returned by With share the underlying files and lifecycle.
type Logger struct {
	config  Config
	core    *loggerCore
	handler slog.Handler
	ctx     context.Context
}

type loggerCore struct {
	mu          sync.RWMutex
	cleanupMu   sync.Mutex
	config      Config
	baseDir     string
	level       slog.LevelVar
	sinks       *sinkSet
	currentDate string
	limiter     *rateLimiter
	nowFunc     func() time.Time
	stopCleanup chan struct{}
	stopOnce    sync.Once
	cleanupWG   sync.WaitGroup
	closed      bool
}

type sinkSet struct {
	mainFile     *os.File
	errorFile    *os.File
	mainBuffer   *bufferedWriter
	errorBuffer  *bufferedWriter
	mainHandler  slog.Handler
	errorHandler slog.Handler
	stdout       slog.Handler
	stderr       slog.Handler
}

type minimumLevel struct{}

func (minimumLevel) Level() slog.Level { return slog.Level(-1 << 30) }

// New validates config, opens the current daily files, and starts retention cleanup.
func New(config Config) (*Logger, error) {
	config = config.withDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid logger configuration: %w", err)
	}
	config.Filters = cloneFilterConfig(config.Filters)
	if err := os.MkdirAll(config.LogDir, 0o700); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	baseDir, err := filepath.Abs(config.LogDir)
	if err != nil {
		return nil, fmt.Errorf("resolve log directory: %w", err)
	}

	core := &loggerCore{
		config:      config,
		baseDir:     baseDir,
		limiter:     newRateLimiter(config.Filters.RateLimits),
		nowFunc:     time.Now,
		stopCleanup: make(chan struct{}),
	}
	core.level.Set(config.LogLevel)
	today := core.now().Format(time.DateOnly)
	core.sinks, err = newSinkSet(config, baseDir, today)
	if err != nil {
		return nil, err
	}
	core.currentDate = today

	logger := &Logger{config: config, core: core, ctx: context.Background()}
	logger.handler = newPipelineHandler(core)
	core.cleanupWG.Add(1)
	go core.cleanupLoop()
	return logger, nil
}

func newSinkSet(config Config, baseDir, date string) (*sinkSet, error) {
	mainPath := filepath.Join(baseDir, fmt.Sprintf("%s_%s.log", config.AppName, date))
	errorPath := filepath.Join(baseDir, fmt.Sprintf("%s_error_%s.log", config.AppName, date))
	mainFile, err := os.OpenFile(mainPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open main log file: %w", err)
	}
	errorFile, err := os.OpenFile(errorPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		_ = mainFile.Close()
		return nil, fmt.Errorf("open error log file: %w", err)
	}

	mainBuffer := newBufferedWriter(mainFile, config.BufferSize, config.FlushInterval)
	errorBuffer := newBufferedWriter(errorFile, config.BufferSize, config.FlushInterval)
	opts := &slog.HandlerOptions{
		AddSource: config.AddSource,
		Level:     minimumLevel{},
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				attr.Value = slog.StringValue(attr.Value.Time().Format(config.TimeFormat))
			}
			return attr
		},
	}
	createHandler := func(writer io.Writer) slog.Handler {
		if config.JSONFormat {
			return slog.NewJSONHandler(writer, opts)
		}
		return slog.NewTextHandler(writer, opts)
	}
	sinks := &sinkSet{
		mainFile:     mainFile,
		errorFile:    errorFile,
		mainBuffer:   mainBuffer,
		errorBuffer:  errorBuffer,
		mainHandler:  createHandler(mainBuffer),
		errorHandler: createHandler(errorBuffer),
	}
	if config.ConsoleOutput {
		sinks.stdout = createHandler(os.Stdout)
		sinks.stderr = createHandler(os.Stderr)
	}
	return sinks, nil
}

func (s *sinkSet) close() error {
	if s == nil {
		return nil
	}
	return errors.Join(s.mainBuffer.Close(), s.errorBuffer.Close(), s.mainFile.Close(), s.errorFile.Close())
}

func (c *loggerCore) now() time.Time {
	if c.nowFunc != nil {
		return c.nowFunc()
	}
	return time.Now()
}

func (c *loggerCore) enabled(level slog.Level) bool {
	c.mu.RLock()
	closed := c.closed
	c.mu.RUnlock()
	return !closed && level >= c.level.Level()
}

func (c *loggerCore) route(ctx context.Context, record slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	date := c.now().Format(time.DateOnly)
	if date != c.currentDate {
		if err := c.reopenLocked(date); err != nil {
			return err
		}
	}

	var errs []error
	if err := c.sinks.mainHandler.Handle(ctx, record); err != nil {
		errs = append(errs, err)
	}
	if record.Level >= slog.LevelWarn {
		if err := c.sinks.errorHandler.Handle(ctx, record); err != nil {
			errs = append(errs, err)
		}
	}
	if c.sinks.stdout != nil {
		console := c.sinks.stdout
		if record.Level >= slog.LevelWarn {
			console = c.sinks.stderr
		}
		if err := console.Handle(ctx, record); err != nil {
			errs = append(errs, err)
		}
	}
	if c.config.BufferSize > 0 && record.Level >= c.config.FlushOnLevel {
		errs = append(errs, c.sinks.mainBuffer.Flush())
		if record.Level >= slog.LevelWarn {
			errs = append(errs, c.sinks.errorBuffer.Flush())
		}
	}
	return errors.Join(errs...)
}

func (c *loggerCore) reopenLocked(date string) error {
	newSinks, err := newSinkSet(c.config, c.baseDir, date)
	if err != nil {
		return err
	}
	old := c.sinks
	c.sinks = newSinks
	c.currentDate = date
	return old.close()
}

func (l *Logger) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	if l == nil || l.core == nil || !l.handler.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	record := slog.NewRecord(l.core.now(), level, msg, pcs[0])
	record.Add(args...)
	_ = l.handler.Handle(ctx, record)
}

// Debug logs at slog.LevelDebug.
func (l *Logger) Debug(msg string, args ...any) { l.log(l.context(), slog.LevelDebug, msg, args...) }

// Info logs at slog.LevelInfo.
func (l *Logger) Info(msg string, args ...any) { l.log(l.context(), slog.LevelInfo, msg, args...) }

// Warn logs at slog.LevelWarn.
func (l *Logger) Warn(msg string, args ...any) { l.log(l.context(), slog.LevelWarn, msg, args...) }

// Error logs at slog.LevelError.
func (l *Logger) Error(msg string, args ...any) { l.log(l.context(), slog.LevelError, msg, args...) }

// DebugContext logs at slog.LevelDebug with ctx.
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.log(nonNilContext(ctx), slog.LevelDebug, msg, args...)
}

// InfoContext logs at slog.LevelInfo with ctx.
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.log(nonNilContext(ctx), slog.LevelInfo, msg, args...)
}

// WarnContext logs at slog.LevelWarn with ctx.
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.log(nonNilContext(ctx), slog.LevelWarn, msg, args...)
}

// ErrorContext logs at slog.LevelError with ctx.
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.log(nonNilContext(ctx), slog.LevelError, msg, args...)
}

func (l *Logger) context() context.Context {
	if l == nil {
		return context.Background()
	}
	return nonNilContext(l.ctx)
}

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// With returns a logger with additional structured attributes.
func (l *Logger) With(args ...any) *Logger {
	if l == nil {
		return nil
	}
	record := slog.NewRecord(time.Time{}, slog.LevelInfo, "", 0)
	record.Add(args...)
	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool { attrs = append(attrs, attr); return true })
	return &Logger{config: l.config, core: l.core, handler: l.handler.WithAttrs(attrs), ctx: l.ctx}
}

// WithContext returns a logger that uses ctx for calls without an explicit context.
// Deprecated: pass context to DebugContext, InfoContext, WarnContext, or ErrorContext.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{config: l.config, core: l.core, handler: l.handler, ctx: nonNilContext(ctx)}
}

// SetLevel changes the shared minimum level without reopening files.
func (l *Logger) SetLevel(level slog.Level) error {
	if l == nil || l.core == nil {
		return nil
	}
	l.core.level.Set(level)
	l.core.mu.Lock()
	l.core.config.LogLevel = level
	l.core.mu.Unlock()
	l.config.LogLevel = level
	return nil
}

// Flush writes both pending file buffers.
func (l *Logger) Flush() error {
	if l == nil || l.core == nil {
		return nil
	}
	l.core.mu.RLock()
	defer l.core.mu.RUnlock()
	if l.core.closed {
		return nil
	}
	return errors.Join(l.core.sinks.mainBuffer.Flush(), l.core.sinks.errorBuffer.Flush())
}

// Close stops background work and closes shared files. It is safe to call repeatedly.
func (l *Logger) Close() error {
	if l == nil || l.core == nil {
		return nil
	}
	l.core.stopOnce.Do(func() { close(l.core.stopCleanup) })
	l.core.cleanupWG.Wait()
	l.core.mu.Lock()
	defer l.core.mu.Unlock()
	if l.core.closed {
		return nil
	}
	l.core.closed = true
	return l.core.sinks.close()
}

// Reopen closes and reopens the current daily files. It is useful with
// external file-rotation tools.
func (l *Logger) Reopen() error {
	if l == nil || l.core == nil {
		return nil
	}
	l.core.mu.Lock()
	defer l.core.mu.Unlock()
	if l.core.closed {
		return nil
	}
	return l.core.reopenLocked(l.core.now().Format(time.DateOnly))
}

// RotateNow is retained for compatibility.
// Deprecated: use Reopen; daily filenames cannot rotate twice on one date.
func (l *Logger) RotateNow() error { return l.Reopen() }
