package main

import (
	"log/slog"
	"time"

	"github.com/sarff/iSlogger"
)

func main() {
	// Initialize global logger with default configuration
	config := iSlogger.DefaultConfig().
		WithAppName("myapp").
		WithLogLevel(slog.LevelDebug).
		WithLogDir("logs")

	if err := iSlogger.Init(config); err != nil {
		panic(err)
	}
	defer iSlogger.Close()

	// Use global convenience functions
	iSlogger.Info("Application started", "version", "1.0.0")
	iSlogger.Debug("Debug information", "user_id", 123)
	iSlogger.Warn("This is a warning", "component", "database")
	iSlogger.Error("Something went wrong", "error", "connection timeout")

	// Create logger with context
	userLogger := iSlogger.With("user_id", 456, "session", "abc123")
	userLogger.Info("User logged in", "username", "john_doe")
	userLogger.Error("User action failed", "action", "purchase", "reason", "insufficient funds")

	// Demonstrate log level switching
	iSlogger.Info("Switching to WARN level...")
	iSlogger.SetLevel(slog.LevelWarn) // Only warnings and errors will be logged

	iSlogger.Debug("This won't appear") // Won't be logged
	iSlogger.Info("This won't appear")  // Won't be logged
	iSlogger.Warn("This will appear")   // Will be logged
	iSlogger.Error("This will appear")  // Will be logged

	// Switch back to debug level
	iSlogger.SetLevel(slog.LevelDebug)
	iSlogger.Debug("Debug level is back!")

	// Structured logging example
	iSlogger.Info("Request processed",
		"method", "POST",
		"path", "/api/users",
		"status", 201,
		"duration", time.Millisecond*150,
		"ip", "192.168.1.100",
		"user_agent", "MyApp/1.0",
	)

	time.Sleep(time.Second) // Allow time for async operations
}
