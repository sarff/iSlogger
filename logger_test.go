package iSlogger

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test").
		WithLogDir("test-logs").
		WithLogLevel(slog.LevelDebug)

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs")

	if logger.config.AppName != "test" {
		t.Errorf("Expected app name 'test', got '%s'", logger.config.AppName)
	}

	if logger.config.LogLevel != slog.LevelDebug {
		t.Error("Expected log level to be DEBUG")
	}
}

func TestLogLevels(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-levels").
		WithLogDir("test-logs-levels").
		WithLogLevel(slog.LevelDebug)

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-levels")

	logger.Debug("Debug message", "key", "value")
	logger.Info("Info message", "key", "value")
	logger.Warn("Warning message", "key", "value")
	logger.Error("Error message", "key", "value")

	today := time.Now().Format("2006-01-02")
	infoPath := filepath.Join("test-logs-levels", "test-levels_"+today+".log")
	errorPath := filepath.Join("test-logs-levels", "test-levels_error_"+today+".log")

	if _, err := os.Stat(infoPath); os.IsNotExist(err) {
		t.Error("Info log file was not created")
	}

	if _, err := os.Stat(errorPath); os.IsNotExist(err) {
		t.Error("Error log file was not created")
	}
}

func TestLogLevelChange(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-level").
		WithLogDir("test-logs-level").
		WithLogLevel(slog.LevelWarn) // Start with WARN level

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-level")

	if logger.config.LogLevel != slog.LevelWarn {
		t.Error("Expected log level to be WARN initially")
	}

	err = logger.SetLevel(slog.LevelDebug)
	if err != nil {
		t.Errorf("Failed to change log level: %v", err)
	}

	if logger.config.LogLevel != slog.LevelDebug {
		t.Error("Expected log level to be DEBUG after SetLevel(DEBUG)")
	}
}

func TestWith(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-with").
		WithLogDir("test-logs-with").
		WithLogLevel(slog.LevelDebug)

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-with")

	contextLogger := logger.With("user_id", 123, "session", "abc")
	contextLogger.Info("Test message with context")

	logger.Info("Original logger message")
}

func TestGlobalLogger(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-global").
		WithLogDir("test-logs-global").
		WithLogLevel(slog.LevelDebug)

	err := Init(config)
	if err != nil {
		t.Fatalf("Failed to initialize global logger: %v", err)
	}
	defer Close()
	defer os.RemoveAll("test-logs-global")

	Debug("Global debug message")
	Info("Global info message")
	Warn("Global warning message")
	Error("Global error message")

	contextLogger := With("global_key", "global_value")
	if contextLogger == nil {
		t.Error("Expected non-nil logger from global With()")
	}
}

func TestConfigBuilder(t *testing.T) {
	config := DefaultConfig().
		WithAppName("builder-test").
		WithLogDir("builder-logs").
		WithLogLevel(slog.LevelDebug).
		WithRetentionDays(14).
		WithJSONFormat(true).
		WithTimeFormat("2006-01-02 15:04:05").
		WithAddSource(true)

	if config.AppName != "builder-test" {
		t.Errorf("Expected app name 'builder-test', got '%s'", config.AppName)
	}

	if config.LogDir != "builder-logs" {
		t.Errorf("Expected log dir 'builder-logs', got '%s'", config.LogDir)
	}

	if config.LogLevel != slog.LevelDebug {
		t.Error("Expected log level to be DEBUG")
	}

	if config.RetentionDays != 14 {
		t.Errorf("Expected retention days 14, got %d", config.RetentionDays)
	}

	if !config.JSONFormat {
		t.Error("Expected JSON format to be enabled")
	}

	if config.TimeFormat != "2006-01-02 15:04:05" {
		t.Errorf("Expected custom time format, got '%s'", config.TimeFormat)
	}

	if !config.AddSource {
		t.Error("Expected add-source to be disabled")
	}
}

func TestFileRotation(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-rotation").
		WithLogDir("test-logs-rotation").
		WithLogLevel(slog.LevelDebug)

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-rotation")

	logger.Info("Before rotation")

	err = logger.RotateNow()
	if err != nil {
		t.Errorf("Failed to rotate logs: %v", err)
	}

	logger.Info("After rotation")

	// Check that files exist
	files, err := logger.GetLogFiles()
	if err != nil {
		t.Errorf("Failed to get log files: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected at least one log file")
	}
}

func TestLogFileNaming(t *testing.T) {
	config := DefaultConfig().
		WithAppName("naming-test").
		WithLogDir("test-logs-naming")

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-naming")

	logger.Info("Test message")

	files, err := logger.GetLogFiles()
	if err != nil {
		t.Errorf("Failed to get log files: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	expectedInfo := "naming-test_" + today + ".log"
	expectedError := "naming-test_error_" + today + ".log"

	var foundInfo, foundError bool
	for _, file := range files {
		if file == expectedInfo {
			foundInfo = true
		}
		if file == expectedError {
			foundError = true
		}
	}

	if !foundInfo {
		t.Errorf("Expected to find info log file '%s', got files: %v", expectedInfo, files)
	}

	if !foundError {
		t.Errorf("Expected to find error log file '%s', got files: %v", expectedError, files)
	}
}

func TestCleanup(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-cleanup").
		WithLogDir("test-logs-cleanup").
		WithRetentionDays(1) // Keep only 1 day

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-cleanup")

	oldDate := time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	oldFile := filepath.Join("test-logs-cleanup", "test-cleanup_"+oldDate+".log")

	file, err := os.Create(oldFile)
	if err != nil {
		t.Fatalf("Failed to create old test file: %v", err)
	}
	file.Close()

	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	os.Chtimes(oldFile, twoDaysAgo, twoDaysAgo)

	logger.CleanupNow()

	time.Sleep(100 * time.Millisecond)

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("Expected old log file to be removed")
	}
}

func TestIsOurLogFile(t *testing.T) {
	config := DefaultConfig().WithAppName("myapp")
	logger := &Logger{config: config}

	tests := []struct {
		filename string
		expected bool
	}{
		{"myapp_2024-01-01.log", true},
		{"myapp_error_2024-01-01.log", true},
		{"otherapp_2024-01-01.log", false},
		{"myapp.txt", false},
		{"random.log", false},
		{"myapp_", false},
	}

	for _, test := range tests {
		result := logger.isOurLogFile(test.filename)
		if result != test.expected {
			t.Errorf("isOurLogFile(%s) = %v, expected %v", test.filename, result, test.expected)
		}
	}
}

func BenchmarkLogging(b *testing.B) {
	config := DefaultConfig().
		WithAppName("bench").
		WithLogDir("bench-logs").
		WithLogLevel(slog.LevelDebug)

	logger, err := New(config)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("bench-logs")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Benchmark message", "iteration", b.N, "timestamp", time.Now())
		}
	})
}

func TestLogger_BufferedWrites(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "islogger_buffer_test")
	defer os.RemoveAll(tempDir)

	config := DefaultConfig().
		WithLogDir(tempDir).
		WithAppName("buffer_test").
		WithLogLevel(slog.LevelDebug). // Enable debug to see INFO messages
		WithBufferSize(1024).
		WithFlushInterval(100 * time.Millisecond).
		WithFlushOnLevel(slog.LevelError)

	l, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer l.Close()

	// Write some logs
	l.Info("This is an info message")
	l.Debug("This is a debug message")
	l.Warn("This is a warning message")

	// Check that files exist but may not have content yet (buffered)
	infoFile := filepath.Join(tempDir, "buffer_test_"+time.Now().Format("2006-01-02")+".log")
	errorFile := filepath.Join(tempDir, "buffer_test_error_"+time.Now().Format("2006-01-02")+".log")

	// Files should exist
	if _, err := os.Stat(infoFile); os.IsNotExist(err) {
		t.Fatal("Info log file should exist")
	}
	if _, err := os.Stat(errorFile); os.IsNotExist(err) {
		t.Fatal("Error log file should exist")
	}

	// Manual flush
	err = l.Flush()
	if err != nil {
		t.Fatalf("Failed to flush logger: %v", err)
	}

	// Now files should have content
	infoContent, err := os.ReadFile(infoFile)
	if err != nil {
		t.Fatalf("Failed to read info file: %v", err)
	}
	if !strings.Contains(string(infoContent), "This is an info message") {
		t.Fatal("Info file should contain info message")
	}

	errorContent, err := os.ReadFile(errorFile)
	if err != nil {
		t.Fatalf("Failed to read error file: %v", err)
	}
	if !strings.Contains(string(errorContent), "This is a warning message") {
		t.Fatal("Error file should contain warning message")
	}
}

func TestLogger_BufferedWritesWithoutBuffering(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "islogger_nobuffer_test")
	defer os.RemoveAll(tempDir)

	config := DefaultConfig().
		WithLogDir(tempDir).
		WithAppName("nobuffer_test").
		WithLogLevel(slog.LevelDebug). // Enable debug to see INFO messages
		WithoutBuffering()             // Disable buffering

	l, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer l.Close()

	// Write some logs
	l.Info("This is an info message")
	l.Warn("This is a warning message")

	// Files should have content immediately (no buffering)
	infoFile := filepath.Join(tempDir, "nobuffer_test_"+time.Now().Format("2006-01-02")+".log")
	errorFile := filepath.Join(tempDir, "nobuffer_test_error_"+time.Now().Format("2006-01-02")+".log")

	infoContent, err := os.ReadFile(infoFile)
	if err != nil {
		t.Fatalf("Failed to read info file: %v", err)
	}
	if !strings.Contains(string(infoContent), "This is an info message") {
		t.Fatal("Info file should immediately contain info message")
	}

	errorContent, err := os.ReadFile(errorFile)
	if err != nil {
		t.Fatalf("Failed to read error file: %v", err)
	}
	if !strings.Contains(string(errorContent), "This is a warning message") {
		t.Fatal("Error file should immediately contain warning message")
	}
}

func TestLogger_BufferedWritesAutoFlush(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "islogger_autoflush_test")
	defer os.RemoveAll(tempDir)

	config := DefaultConfig().
		WithLogDir(tempDir).
		WithAppName("autoflush_test").
		WithLogLevel(slog.LevelDebug). // Enable debug to see INFO messages
		WithBufferSize(1024).
		WithFlushInterval(50 * time.Millisecond) // Very short interval

	l, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer l.Close()

	// Write a log
	l.Info("This is an auto-flush test message")

	infoFile := filepath.Join(tempDir, "autoflush_test_"+time.Now().Format("2006-01-02")+".log")

	// Wait for auto-flush
	time.Sleep(100 * time.Millisecond)

	// File should have content due to auto-flush
	infoContent, err := os.ReadFile(infoFile)
	if err != nil {
		t.Fatalf("Failed to read info file: %v", err)
	}
	if !strings.Contains(string(infoContent), "This is an auto-flush test message") {
		t.Fatal("Info file should contain auto-flushed message")
	}
}

func TestLogger_BufferedWritesImmediateFlushOnError(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "islogger_errorflush_test")
	defer os.RemoveAll(tempDir)

	config := DefaultConfig().
		WithLogDir(tempDir).
		WithAppName("errorflush_test").
		WithBufferSize(1024).
		WithFlushOnLevel(slog.LevelError) // Flush immediately on errors

	l, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer l.Close()

	// Write an error log
	l.Error("This is an error message")

	errorFile := filepath.Join(tempDir, "errorflush_test_error_"+time.Now().Format("2006-01-02")+".log")

	// File should have content immediately due to error level flush
	errorContent, err := os.ReadFile(errorFile)
	if err != nil {
		t.Fatalf("Failed to read error file: %v", err)
	}
	if !strings.Contains(string(errorContent), "This is an error message") {
		t.Fatal("Error file should immediately contain error message")
	}
}
