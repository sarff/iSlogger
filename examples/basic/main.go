package main

import (
	"time"
)

func main() {
	// Initialize global logger with default configuration
	config := islogger.DefaultConfig().
		WithAppName("myapp").
		WithDebug(true).
		WithLogDir("logs")

	if err := islogger.Init(config); err != nil {
		panic(err)
	}
	defer islogger.Close()

	// Use global convenience functions
	islogger.Info("Application started", "version", "1.0.0")
	islogger.Debug("Debug information", "user_id", 123)
	islogger.Warn("This is a warning", "component", "database")
	islogger.Error("Something went wrong", "error", "connection timeout")

	// Create logger with context
	userLogger := islogger.With("user_id", 456, "session", "abc123")
	userLogger.Info("User logged in", "username", "john_doe")
	userLogger.Error("User action failed", "action", "purchase", "reason", "insufficient funds")

	// Demonstrate debug mode switching
	islogger.Info("Switching to production mode...")
	islogger.SetDebug(false) // Only warnings and errors will be logged

	islogger.Debug("This won't appear") // Won't be logged
	islogger.Info("This won't appear")  // Won't be logged
	islogger.Warn("This will appear")   // Will be logged
	islogger.Error("This will appear")  // Will be logged

	// Switch back to debug mode
	islogger.SetDebug(true)
	islogger.Debug("Debug mode is back!")

	// Structured logging example
	islogger.Info("Request processed",
		"method", "POST",
		"path", "/api/users",
		"status", 201,
		"duration", time.Millisecond*150,
		"ip", "192.168.1.100",
		"user_agent", "MyApp/1.0",
	)

	time.Sleep(time.Second) // Allow time for async operations
}
