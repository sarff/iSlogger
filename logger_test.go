package iSlogger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test").
		WithLogDir("test-logs").
		WithDebug(true)

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs")

	if logger.config.AppName != "test" {
		t.Errorf("Expected app name 'test', got '%s'", logger.config.AppName)
	}

	if !logger.config.Debug {
		t.Error("Expected debug mode to be enabled")
	}
}

func TestLogLevels(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-levels").
		WithLogDir("test-logs-levels").
		WithDebug(true)

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

func TestDebugMode(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-debug").
		WithLogDir("test-logs-debug").
		WithDebug(false) // Start in production mode

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-debug")

	if logger.config.Debug {
		t.Error("Expected debug mode to be disabled initially")
	}

	err = logger.SetDebug(true)
	if err != nil {
		t.Errorf("Failed to enable debug mode: %v", err)
	}

	if !logger.config.Debug {
		t.Error("Expected debug mode to be enabled after SetDebug(true)")
	}
}

func TestWith(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-with").
		WithLogDir("test-logs-with").
		WithDebug(true)

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
		WithDebug(true)

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
		WithDebug(true).
		WithRetentionDays(14).
		WithJSONFormat(true).
		WithTimeFormat("2006-01-02 15:04:05")

	if config.AppName != "builder-test" {
		t.Errorf("Expected app name 'builder-test', got '%s'", config.AppName)
	}

	if config.LogDir != "builder-logs" {
		t.Errorf("Expected log dir 'builder-logs', got '%s'", config.LogDir)
	}

	if !config.Debug {
		t.Error("Expected debug mode to be enabled")
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
}

func TestFileRotation(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-rotation").
		WithLogDir("test-logs-rotation").
		WithDebug(true)

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
		WithDebug(true)

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
