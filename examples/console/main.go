package main

import (
	"log/slog"
	"time"

	"github.com/sarff/iSlogger"
)

func main() {
	// Example 1: Default behavior (console output enabled)
	config1 := iSlogger.DefaultConfig().
		WithAppName("console-app").
		WithLogLevel(slog.LevelDebug).
		WithLogDir("logs")

	logger1, err := iSlogger.New(config1)
	if err != nil {
		panic(err)
	}
	defer logger1.Close()

	logger1.Info("This message appears in both console and file (default behavior)")
	logger1.Warn("Warning: appears in console and both files")
	logger1.Error("Error: appears in console and both files")

	// Example 2: Disable console output (file only)
	config2 := iSlogger.DefaultConfig().
		WithAppName("file-only-app").
		WithLogLevel(slog.LevelDebug).
		WithLogDir("logs").
		WithConsoleOutput(false)

	logger2, err := iSlogger.New(config2)
	if err != nil {
		panic(err)
	}
	defer logger2.Close()

	logger2.Info("This message only appears in files, not in console")
	logger2.Warn("File-only warning message")
	logger2.Error("File-only error message")

	// Example 3: Explicitly enable console output
	config3 := iSlogger.DefaultConfig().
		WithAppName("explicit-console").
		WithLogLevel(slog.LevelDebug).
		WithLogDir("logs").
		WithConsoleOutput(true)

	logger3, err := iSlogger.New(config3)
	if err != nil {
		panic(err)
	}
	defer logger3.Close()

	logger3.Info("Explicitly enabled console output")
	logger3.Debug("Debug message with console output")

	// Example 4: JSON format with console output
	config4 := iSlogger.DefaultConfig().
		WithAppName("json-console").
		WithLogLevel(slog.LevelDebug).
		WithLogDir("logs").
		WithJSONFormat(true).
		WithConsoleOutput(true)

	logger4, err := iSlogger.New(config4)
	if err != nil {
		panic(err)
	}
	defer logger4.Close()

	logger4.Info("JSON formatted message in both console and file",
		"user_id", 123,
		"action", "login",
		"timestamp", time.Now(),
	)

	time.Sleep(time.Second) // Allow time for async operations
}
