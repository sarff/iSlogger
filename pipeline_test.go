package iSlogger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type testContextKey struct{}

func newTestLogger(t *testing.T, config Config) *Logger {
	t.Helper()
	logger, err := New(config.WithLogDir(t.TempDir()).WithConsoleOutput(false))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	t.Cleanup(func() {
		if err := logger.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	})
	return logger
}

func readLogs(t *testing.T, logger *Logger) (string, string) {
	t.Helper()
	if err := logger.Flush(); err != nil {
		t.Fatalf("Flush() error: %v", err)
	}
	mainPath, errorPath := logger.GetCurrentLogPaths()
	mainData, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main log: %v", err)
	}
	errorData, err := os.ReadFile(errorPath)
	if err != nil {
		t.Fatalf("read error log: %v", err)
	}
	return string(mainData), string(errorData)
}

func TestLevelRoutingUsesRecordLevel(t *testing.T) {
	logger := newTestLogger(t, DefaultConfig().
		WithAppName("routing").
		WithLogLevel(slog.LevelDebug).
		WithoutBuffering())

	logger.Debug("debug")
	logger.Info("info", "payload", "level=ERROR")
	logger.Warn("warn")
	logger.Error("error")

	mainLog, errorLog := readLogs(t, logger)
	for _, message := range []string{"debug", "info", "warn", "error"} {
		if !strings.Contains(mainLog, "msg="+message) {
			t.Errorf("main log does not contain %q:\n%s", message, mainLog)
		}
	}
	if strings.Contains(errorLog, "msg=debug") || strings.Contains(errorLog, "msg=info") {
		t.Fatalf("error log contains a low-level record:\n%s", errorLog)
	}
	if strings.Count(errorLog, "msg=warn") != 1 || strings.Count(errorLog, "msg=error") != 1 {
		t.Fatalf("error log routing is incorrect:\n%s", errorLog)
	}
}

func TestWithAttributesParticipateInFiltering(t *testing.T) {
	logger := newTestLogger(t, DefaultConfig().
		WithAppName("bound-fields").
		WithoutBuffering().
		WithAttributeCondition("role", "admin").
		WithFieldMask("password", "***").
		WithFieldRedaction("secret").
		WithRegexFilter(`[0-9]{4}`, "####"))

	logger.With("role", "user").Info("denied")
	logger.With(
		"role", "admin",
		"password", "plain-text",
		"secret", "remove-me",
		"empty", "",
	).Info("allowed", slog.Group("nested", "card", "1234"))

	mainLog, _ := readLogs(t, logger)
	if strings.Contains(mainLog, "denied") {
		t.Fatalf("bound condition was not applied:\n%s", mainLog)
	}
	for _, forbidden := range []string{"plain-text", "remove-me", "1234"} {
		if strings.Contains(mainLog, forbidden) {
			t.Errorf("sensitive value %q leaked:\n%s", forbidden, mainLog)
		}
	}
	for _, expected := range []string{"msg=allowed", `password=***`, `empty=""`, "card=####"} {
		if !strings.Contains(mainLog, expected) {
			t.Errorf("filtered log does not contain %q:\n%s", expected, mainLog)
		}
	}
}

func TestConditionRunsBeforeRateLimit(t *testing.T) {
	logger := newTestLogger(t, DefaultConfig().
		WithAppName("condition-limit").
		WithoutBuffering().
		WithMessageContainsCondition("allowed").
		WithRateLimit(slog.LevelInfo, 1, time.Hour))

	logger.Info("denied")
	logger.Info("allowed first")
	logger.Info("allowed second")
	mainLog, _ := readLogs(t, logger)
	if !strings.Contains(mainLog, "allowed first") || strings.Contains(mainLog, "allowed second") {
		t.Fatalf("condition/rate-limit order is incorrect:\n%s", mainLog)
	}
}

func TestWarnRateLimitAppliedOnce(t *testing.T) {
	logger := newTestLogger(t, DefaultConfig().
		WithAppName("warn-limit").
		WithoutBuffering().
		WithRateLimit(slog.LevelWarn, 1, time.Hour))
	logger.Warn("first")
	logger.Warn("second")
	mainLog, errorLog := readLogs(t, logger)
	if strings.Count(mainLog, "level=WARN") != 1 || strings.Count(errorLog, "level=WARN") != 1 {
		t.Fatalf("WARN must consume one quota and reach both files:\nmain=%s\nerror=%s", mainLog, errorLog)
	}
}

func TestRateLimitConcurrent(t *testing.T) {
	logger := newTestLogger(t, DefaultConfig().
		WithAppName("concurrent-limit").
		WithoutBuffering().
		WithRateLimit(slog.LevelInfo, 10, time.Hour))
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("limited")
		}()
	}
	wg.Wait()
	mainLog, _ := readLogs(t, logger)
	if count := strings.Count(mainLog, "msg=limited"); count != 10 {
		t.Fatalf("got %d records, want 10", count)
	}
}

func TestConfigBuildersDoNotMutateSource(t *testing.T) {
	base := DefaultConfig()
	derived := base.WithFieldMask("password", "***").WithRateLimit(slog.LevelInfo, 1, time.Minute)
	if len(base.Filters.FieldFilters) != 0 || len(base.Filters.RateLimits) != 0 {
		t.Fatal("builder mutated its source config")
	}
	if len(derived.Filters.FieldFilters) != 1 || len(derived.Filters.RateLimits) != 1 {
		t.Fatal("derived config is missing filters")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{"negative retention", DefaultConfig().WithRetentionDays(-1)},
		{"negative buffer", DefaultConfig().WithBufferSize(-1)},
		{"negative interval", DefaultConfig().WithFlushInterval(-time.Second)},
		{"unsafe app name", DefaultConfig().WithAppName("../escape")},
		{"invalid regex", DefaultConfig().WithRegexFilter("[", "x")},
		{"invalid rate", DefaultConfig().WithRateLimit(slog.LevelInfo, 0, time.Second)},
		{"invalid hours", DefaultConfig().WithTimeBasedCondition(24, 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.config.Validate(); err == nil {
				t.Fatal("Validate() returned nil")
			}
		})
	}
}

func TestFailedInitKeepsExistingGlobalLogger(t *testing.T) {
	logger := newTestLogger(t, DefaultConfig().WithAppName("existing-global").WithoutBuffering())
	SetGlobalLogger(logger)
	t.Cleanup(func() { _ = Close() })
	if err := Init(DefaultConfig().WithAppName("../invalid")); err == nil {
		t.Fatal("Init() returned nil for invalid config")
	}
	if got := GetGlobalLogger(); got != logger {
		t.Fatal("failed Init replaced the existing global logger")
	}
	Info("still-active")
	mainLog, _ := readLogs(t, logger)
	if !strings.Contains(mainLog, "still-active") {
		t.Fatal("existing global logger stopped after failed Init")
	}
}

func TestSourcePointsToCaller(t *testing.T) {
	logger := newTestLogger(t, DefaultConfig().
		WithAppName("source").
		WithAddSource(true).
		WithoutBuffering())
	logger.Info("instance-source")
	mainLog, _ := readLogs(t, logger)
	if !strings.Contains(mainLog, "pipeline_test.go") || strings.Contains(mainLog, "logger.go") {
		t.Fatalf("instance source is incorrect: %s", mainLog)
	}

	SetGlobalLogger(logger)
	t.Cleanup(func() { _ = Close() })
	InfoContext(context.Background(), "global-source")
	mainLog, _ = readLogs(t, logger)
	line := mainLog[strings.LastIndex(mainLog, "time="):]
	if !strings.Contains(line, "pipeline_test.go") || strings.Contains(line, "global.go") {
		t.Fatalf("global source is incorrect: %s", line)
	}
}

func TestDerivedLoggerSharesLifecycle(t *testing.T) {
	logger := newTestLogger(t, DefaultConfig().WithAppName("shared").WithoutBuffering())
	derived := logger.With("component", "worker")
	if err := derived.Reopen(); err != nil {
		t.Fatalf("derived Reopen() error: %v", err)
	}
	logger.Info("root")
	derived.Info("child")
	mainLog, _ := readLogs(t, logger)
	if !strings.Contains(mainLog, "msg=root") || !strings.Contains(mainLog, "msg=child component=worker") {
		t.Fatalf("derived logger does not share live sinks:\n%s", mainLog)
	}
}

func TestContextAndGlobalManagementAPI(t *testing.T) {
	logger := newTestLogger(t, DefaultConfig().
		WithAppName("management-api").
		WithLogLevel(slog.LevelDebug).
		WithoutBuffering())
	ctx := context.WithValue(context.Background(), testContextKey{}, "value")
	logger.DebugContext(ctx, "instance-debug")
	logger.InfoContext(ctx, "instance-info")
	logger.WarnContext(ctx, "instance-warn")
	logger.ErrorContext(ctx, "instance-error")
	logger.WithContext(ctx).Info("bound-context")

	SetGlobalLogger(logger)
	t.Cleanup(func() { _ = Close() })
	DebugContext(ctx, "global-debug")
	WarnContext(ctx, "global-warn")
	ErrorContext(ctx, "global-error")
	WithContext(ctx).Info("global-bound-context")
	if err := SetLevel(slog.LevelWarn); err != nil {
		t.Fatal(err)
	}
	Info("filtered-after-level-change")
	if err := Flush(); err != nil {
		t.Fatal(err)
	}
	if err := Reopen(); err != nil {
		t.Fatal(err)
	}
	if err := Cleanup(); err != nil {
		t.Fatal(err)
	}
	CleanupNow()
	if files, err := GetLogFiles(); err != nil || len(files) != 2 {
		t.Fatalf("GetLogFiles() = %v, %v", files, err)
	}
	mainLog, _ := readLogs(t, logger)
	for _, message := range []string{
		"instance-debug", "instance-info", "instance-warn", "instance-error",
		"bound-context", "global-debug", "global-warn", "global-error", "global-bound-context",
	} {
		if !strings.Contains(mainLog, message) {
			t.Errorf("main log does not contain %q", message)
		}
	}
	if strings.Contains(mainLog, "filtered-after-level-change") {
		t.Fatal("SetLevel did not update the shared level")
	}
}

func TestZeroConfigDefaultsAndCompatibilityHelpers(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(Config{LogDir: dir, ConsoleOutput: false})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logger.Close() })
	if logger.config.AppName != "app" || logger.config.RetentionDays != 7 || logger.config.TimeFormat != time.RFC3339 {
		t.Fatalf("defaults were not applied: %+v", logger.config)
	}
	if err := logger.RotateNow(); err != nil {
		t.Fatal(err)
	}
	if filter := RegexMaskFilter(`[0-9]+`, "#"); filter.Pattern.ReplaceAllString("123", filter.Replacement) != "#" {
		t.Fatal("RegexMaskFilter compatibility wrapper failed")
	}
	buffered := DefaultConfig().WithoutBuffering().WithBuffering()
	if buffered.BufferSize != 8192 || buffered.FlushInterval != 5*time.Second {
		t.Fatal("WithBuffering did not restore defaults")
	}
}

func TestPipelineHandlerGroup(t *testing.T) {
	logger := newTestLogger(t, DefaultConfig().WithAppName("groups").WithoutBuffering())
	grouped := slog.New(logger.handler.WithGroup("request")).With("id", 42)
	grouped.Info("grouped", "empty", "")
	mainLog, _ := readLogs(t, logger)
	if !strings.Contains(mainLog, "request.id=42") || !strings.Contains(mainLog, `request.empty=""`) {
		t.Fatalf("group attributes were not preserved: %s", mainLog)
	}
}

func TestCleanupUsesExactDatedNames(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(DefaultConfig().WithLogDir(dir).WithAppName("cleanup").WithRetentionDays(1).WithConsoleOutput(false))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logger.Close() })
	old := filepath.Join(dir, "cleanup_2000-01-01.log")
	unrelated := filepath.Join(dir, "cleanup_notes.log")
	if err := os.WriteFile(old, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unrelated, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := logger.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error: %v", err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("expired dated log was not removed")
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("unrelated log was removed: %v", err)
	}
}

func TestCleanupConcurrentCalls(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(DefaultConfig().WithLogDir(dir).WithAppName("concurrent-cleanup").WithRetentionDays(1).WithConsoleOutput(false))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logger.Close() })

	old := filepath.Join(dir, "concurrent-cleanup_2000-01-01.log")
	unrelated := filepath.Join(dir, "concurrent-cleanup_notes.log")
	if err := os.WriteFile(old, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unrelated, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	const cleanupCalls = 16
	start := make(chan struct{})
	errs := make(chan error, cleanupCalls)
	var wg sync.WaitGroup
	for range cleanupCalls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- logger.Cleanup()
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent Cleanup() error: %v", err)
		}
	}

	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("expired dated log was not removed")
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("unrelated log was removed: %v", err)
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) / 2, nil }

func TestBufferedWriterPreservesRemainderAfterShortWrite(t *testing.T) {
	bw := newBufferedWriter(shortWriter{}, 100, 0)
	if _, err := bw.Write([]byte("abcdefgh")); err != nil {
		t.Fatal(err)
	}
	if err := bw.Flush(); err != io.ErrShortWrite {
		t.Fatalf("Flush() error = %v, want %v", err, io.ErrShortWrite)
	}
	if got := bw.buffer.String(); got != "efgh" {
		t.Fatalf("remaining buffer = %q, want %q", got, "efgh")
	}
}
