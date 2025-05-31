# iSlogger 📝

Production-ready Go logging package built on top of `slog` with automatic file rotation, dual output streams, and flexible configuration for enterprise applications.

## 🚀 Features

- **Dual Output Streams**: Separate files for general logs and errors
- **Automatic File Rotation**: Daily rotation with configurable retention
- **Debug/Production Modes**: Easy switching between detailed and minimal logging
- **Structured Logging**: Built on Go's standard `slog` package
- **Thread-Safe**: Concurrent logging support with mutex protection
- **Zero Dependencies**: Uses only Go standard library
- **Flexible Configuration**: Builder pattern for easy setup
- **Global & Instance Loggers**: Use global functions or create instances
- **Context Support**: Context-aware logging for request tracking

## 📦 Installation

```bash
go get github.com/sarff/iSlogger
```

## 🔧 Quick Start

### Basic Usage

```go
package main

import (
    "github.com/sarff/iSlogger"
)

func main() {
    // Initialize with default configuration
    if err := islogger.Init(islogger.DefaultConfig()); err != nil {
        panic(err)
    }
    defer islogger.Close()

    // Use global logging functions
    islogger.Info("Application started", "version", "1.0.0")
    islogger.Error("Something went wrong", "error", "connection failed")
}
```

### Advanced Configuration

```go
config := islogger.DefaultConfig().
    WithAppName("myapp").
    WithLogDir("logs").
    WithDebug(true).
    WithRetentionDays(14).
    WithJSONFormat(true).
    WithTimeFormat("2006-01-02 15:04:05")

logger, err := islogger.New(config)
if err != nil {
    panic(err)
}
defer logger.Close()
```

## 📋 Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `LogDir` | `"logs"` | Directory for log files |
| `AppName` | `"app"` | Application name (used in filenames) |
| `Debug` | `false` | Enable debug level logging |
| `RetentionDays` | `7` | Days to keep old log files |
| `JSONFormat` | `false` | Use JSON format instead of text |
| `AddSource` | `true` | Include source file and line info |
| `TimeFormat` | `RFC3339` | Custom time format |

## 📁 File Structure

The logger creates two types of files daily:

- `{AppName}_{YYYY-MM-DD}.log` - All log messages
- `{AppName}_error_{YYYY-MM-DD}.log` - Only warnings and errors

Example files:
```
logs/
├── myapp_2024-01-15.log       # All logs
├── myapp_error_2024-01-15.log # Errors only
├── myapp_2024-01-14.log       # Previous day
└── myapp_error_2024-01-14.log
```

## 🎯 Usage Examples

### Debug Mode Switching

```go
// Start in production mode (only warnings and errors)
logger.SetDebug(false)
logger.Debug("Won't be logged")
logger.Error("Will be logged")

// Switch to debug mode
logger.SetDebug(true)
logger.Debug("Now this will be logged")
```

### Structured Logging

```go
logger.Info("Request processed",
    "method", "POST",
    "path", "/api/users",
    "status", 201,
    "duration", time.Millisecond*150,
    "user_id", 12345,
)
```

### Context Logging

```go
// Create logger with context
userLogger := logger.With("user_id", 123, "session", "abc")
userLogger.Info("User action", "action", "login")

// Chain contexts
requestLogger := userLogger.With("request_id", "req-456")
requestLogger.Error("Request failed", "error", "timeout")
```

### Web Application Example

```go
func loggingMiddleware(logger *islogger.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            requestLogger := logger.With(
                "method", r.Method,
                "path", r.URL.Path,
                "request_id", generateRequestID(),
            )
            
            requestLogger.Info("Request started")
            next.ServeHTTP(w, r)
            
            requestLogger.Info("Request completed",
                "duration", time.Since(start),
            )
        })
    }
}
```

## 🧪 Testing

Run the test suite:

```bash
go test -v
```

Run benchmarks:

```bash
go test -bench=.
```

## 🔧 API Reference

### Global Functions

```go
// Initialize global logger
Init(config Config) error
InitDefault() error

// Logging functions
Debug(msg string, args ...any)
Info(msg string, args ...any)
Warn(msg string, args ...any)
Error(msg string, args ...any)

// Context functions
With(args ...any) *Logger
WithContext(ctx context.Context) *Logger

// Control functions
SetDebug(debug bool) error
SetLevel(level slog.Level) error
Close() error
```

### Logger Methods

```go
// Create new logger
New(config Config) (*Logger, error)

// Logging methods
Debug(msg string, args ...any)
Info(msg string, args ...any)
Warn(msg string, args ...any)
Error(msg string, args ...any)

// Context methods
With(args ...any) *Logger
WithContext(ctx context.Context) *Logger

// Management methods
SetDebug(debug bool) error
SetLevel(level slog.Level) error
RotateNow() error
CleanupNow()
GetLogFiles() ([]string, error)
GetCurrentLogPaths() (infoPath, errorPath string)
Close() error
```

### Configuration Builder

```go
config := DefaultConfig().
    WithAppName("myapp").
    WithLogDir("custom-logs").
    WithDebug(true).
    WithRetentionDays(30).
    WithJSONFormat(true).
    WithTimeFormat("2006-01-02 15:04:05")
```

## 🎨 Log Levels

- **Debug**: Detailed information for debugging (only in debug mode)
- **Info**: General information about application flow
- **Warn**: Warning messages (logged to both files)
- **Error**: Error messages (logged to both files)

## 🔄 File Rotation

- **Automatic**: New files created daily at midnight
- **Manual**: Force rotation with `RotateNow()`
- **Cleanup**: Old files automatically removed after retention period

## 🛡️ Thread Safety

iSlogger is fully thread-safe and supports concurrent logging from multiple goroutines.

## 📈 Performance

- Minimal overhead with efficient file I/O
- Asynchronous cleanup operations
- Optimized for high-throughput applications

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

Made with ❤️ in Ukraine 🇺🇦