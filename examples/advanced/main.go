package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sarff/iSlogger"
)

func main() {
	// Advanced configuration with JSON format and custom time
	config := iSlogger.DefaultConfig().
		WithAppName("advanced-app").
		WithLogLevel(slog.LevelDebug).
		WithLogDir("advanced-logs").
		WithRetentionDays(14).
		WithJSONFormat(true).
		WithTimeFormat("2006-01-02 15:04:05")

	// Create multiple logger instances
	logger1, err := iSlogger.New(config)
	if err != nil {
		panic(err)
	}
	defer logger1.Close()

	// Different configuration for second logger
	config2 := config.
		WithAppName("service-2").
		WithJSONFormat(false) // Text format

	logger2, err := iSlogger.New(config2)
	if err != nil {
		panic(err)
	}
	defer logger2.Close()

	// Demonstrate different loggers
	logger1.Info("Logger 1 started", "format", "JSON")
	logger2.Info("Logger 2 started", "format", "Text")

	// Context-aware logging
	ctx := context.WithValue(context.Background(), "request_id", "req-12345")
	ctxLogger := logger1.WithContext(ctx)
	ctxLogger.Info("Processing request", "operation", "user_creation")

	// Chained context building
	sessionLogger := logger1.
		With("session_id", "sess-98765").
		With("user_id", 42).
		With("ip", "192.168.1.200")

	sessionLogger.Info("User session started")
	sessionLogger.Warn("Suspicious activity detected", "reason", "multiple_failed_logins")

	// Dynamic level switching
	logger1.Info("Current level: Debug")
	logger1.SetLevel(slog.LevelError) // Only errors will be logged
	logger1.Info("This won't be logged")
	logger1.Error("This will be logged", "critical", true)

	// Reset to debug
	logger1.SetLevel(slog.LevelDebug)
	logger1.Info("Back to debug level")

	// File management operations
	files, err := logger1.GetLogFiles()
	if err != nil {
		logger1.Error("Failed to get log files", "error", err)
	} else {
		logger1.Info("Current log files", "count", len(files), "files", files)
	}

	// Get current log paths
	infoPath, errorPath := logger1.GetCurrentLogPaths()
	logger1.Info("Log file paths",
		"info_log", infoPath,
		"error_log", errorPath,
	)

	// Force rotation (useful for testing)
	logger1.Info("Forcing log rotation...")
	if err := logger1.RotateNow(); err != nil {
		logger1.Error("Failed to rotate logs", "error", err)
	} else {
		logger1.Info("Log rotation completed")
	}

	// Manual cleanup trigger
	logger1.Info("Triggering cleanup...")
	logger1.CleanupNow()

	// Performance logging example
	start := time.Now()
	simulateWork()
	duration := time.Since(start)

	logger1.Info("Operation completed",
		"operation", "simulate_work",
		"duration_ms", duration.Milliseconds(),
		"duration_str", duration.String(),
		"success", true,
	)

	// Error handling with stack context
	if err := simulateError(); err != nil {
		logger1.Error("Operation failed",
			"operation", "simulate_error",
			"error", err.Error(),
			"error_type", fmt.Sprintf("%T", err),
			"retry_recommended", true,
		)
	}

	// Batch logging for high-throughput scenarios
	logger1.Info("Starting batch operations...")
	for i := 0; i < 10; i++ {
		batchLogger := logger1.With("batch_id", fmt.Sprintf("batch-%d", i))
		batchLogger.Debug("Processing batch item", "item", i)

		if i%3 == 0 {
			batchLogger.Warn("Slow processing detected", "item", i, "threshold_ms", 100)
		}
	}
	logger1.Info("Batch operations completed")

	time.Sleep(2 * time.Second) // Allow async operations to complete
}

func simulateWork() {
	time.Sleep(50 * time.Millisecond)
}

func simulateError() error {
	return fmt.Errorf("simulated database connection error")
}
