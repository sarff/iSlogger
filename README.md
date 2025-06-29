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
- **🔒 Field Filtering**: Mask or redact sensitive data (passwords, tokens, etc.)
- **🎯 Conditional Logging**: Log only when specific conditions are met
- **⚡ Rate Limiting**: Prevent log flooding with per-level rate limits
- **🛡️ Security**: Built-in protection for sensitive information

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

### Basic Configuration
| Option | Default | Description |
|--------|---------|-------------|
| `LogDir` | `"logs"` | Directory for log files |
| `AppName` | `"app"` | Application name (used in filenames) |
| `Debug` | `false` | Enable debug level logging |
| `RetentionDays` | `7` | Days to keep old log files |
| `JSONFormat` | `false` | Use JSON format instead of text |
| `AddSource` | `false` | Include source file and line info |
| `TimeFormat` | `RFC3339` | Custom time format |

### Filtering Configuration Methods
| Method | Description |
|--------|-------------|
| `WithFieldMask(key, mask)` | Mask field value with specified string |
| `WithFieldRedaction(key)` | Completely remove field from logs |
| `WithRegexFilter(pattern, replacement)` | Replace regex matches with replacement |
| `WithCondition(condition)` | Add custom logging condition |
| `WithLevelCondition(level)` | Only log at or above specified level |
| `WithMessageContainsCondition(text)` | Only log messages containing text |
| `WithAttributeCondition(key, value)` | Only log when attribute matches value |
| `WithTimeBasedCondition(start, end)` | Only log during specified hours |
| `WithRateLimit(level, count, period)` | Rate limit logs for specific level |

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

## 🔒 Field Filtering & Security

Protect sensitive information with built-in field filtering:

### Field Masking

```go
config := islogger.DefaultConfig().
    WithFieldMask("password", "***").
    WithFieldMask("api_key", "***HIDDEN***").
    WithFieldRedaction("internal_data") // Completely removes field

logger, _ := islogger.New(config)
logger.Info("User login", 
    "username", "john",
    "password", "secret123",        // Will be logged as "***"
    "api_key", "sk_live_1234567",   // Will be logged as "***HIDDEN***"
    "internal_data", "sensitive",   // Will be completely removed
)
```

### Regex Filtering

```go
config := islogger.DefaultConfig().
    WithRegexFilter(`\d{4}-\d{4}-\d{4}-\d{4}`, "****-****-****-****"). // Credit cards
    WithRegexFilter(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, "***@***.***") // Emails

logger, _ := islogger.New(config)
logger.Info("Payment", "message", "Card 1234-5678-9012-3456 for user@example.com")
// Logs: "Card ****-****-****-**** for ***@***.***"
```

## 🎯 Conditional Logging

Log only when specific conditions are met:

### Level-Based Conditions

```go
config := islogger.DefaultConfig().
    WithLevelCondition(slog.LevelWarn) // Only WARN and ERROR

logger, _ := islogger.New(config)
logger.Debug("Debug info")  // NOT logged
logger.Info("Info message") // NOT logged
logger.Warn("Warning")      // Logged
logger.Error("Error")       // Logged
```

### Content-Based Conditions

```go
config := islogger.DefaultConfig().
    WithMessageContainsCondition("important"). // Messages containing "important"
    WithAttributeCondition("user_type", "admin") // OR admin users only

logger, _ := islogger.New(config)
logger.Info("Regular operation")                                    // NOT logged
logger.Info("Important system notification")                       // Logged
logger.Info("User action", "user_type", "admin", "action", "delete") // Logged
```

### Time-Based Conditions

```go
config := islogger.DefaultConfig().
    WithTimeBasedCondition(9, 18) // Only during work hours (9 AM - 6 PM)

logger, _ := islogger.New(config)
// Logs only between 9:00 and 18:00
```

### Complex Conditions

```go
// Combine multiple conditions with AND/OR logic
config := islogger.DefaultConfig().
    WithCondition(islogger.AnyCondition(
        islogger.LevelCondition(slog.LevelWarn),           // WARN+ OR
        islogger.MessageContainsCondition("important"),    // contains "important" OR
        islogger.CombineConditions(                        // (admin AND action)
            islogger.AttributeCondition("user_type", "admin"),
            islogger.MessageContainsCondition("action"),
        ),
    ))
```

## ⚡ Rate Limiting

Prevent log flooding with per-level rate limits:

```go
config := islogger.DefaultConfig().
    WithRateLimit(slog.LevelDebug, 100, time.Minute). // Max 100 DEBUG/minute
    WithRateLimit(slog.LevelInfo, 500, time.Minute).  // Max 500 INFO/minute
    WithRateLimit(slog.LevelWarn, 50, time.Minute)    // Max 50 WARN/minute

logger, _ := islogger.New(config)

// Rapid logging - only first 100 DEBUG messages per minute will be logged
for i := 0; i < 1000; i++ {
    logger.Debug("Debug message", "count", i)
}
```

## 🏭 Production Configuration

Complete example for production environments:

```go
config := islogger.DefaultConfig().
    WithAppName("myapp").
    WithLogDir("/var/log/myapp").
    WithDebug(false).
    WithRetentionDays(30).
    WithJSONFormat(true).
    // Security: mask sensitive fields
    WithFieldMask("password", "***").
    WithFieldMask("api_key", "***").
    WithFieldMask("token", "***").
    WithFieldRedaction("internal").
    // Filter sensitive patterns
    WithRegexFilter(`(?i)password["\s]*[:=]["\s]*[^";\s]+`, "password: ***").
    WithRegexFilter(`\d{4}-\d{4}-\d{4}-\d{4}`, "****-****-****-****").
    WithRegexFilter(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, "***@***.***").
    // Production logging conditions
    WithCondition(islogger.AnyCondition(
        islogger.LevelCondition(slog.LevelWarn),        // All warnings/errors
        islogger.AttributeCondition("service", "payment"), // Payment service logs
        islogger.AttributeCondition("critical", "true"),    // Critical operations
    )).
    // Rate limiting
    WithRateLimit(slog.LevelInfo, 1000, time.Minute).
    WithRateLimit(slog.LevelDebug, 100, time.Minute)

logger, err := islogger.New(config)
if err != nil {
    panic(err)
}
defer logger.Close()
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
    WithTimeFormat("2006-01-02 15:04:05").
    WithAddSource(true).
    // Add filtering
    WithFieldMask("password", "***").
    WithRegexFilter(`\d{4}-\d{4}-\d{4}-\d{4}`, "****-****-****-****").
    WithLevelCondition(slog.LevelInfo).
    WithRateLimit(slog.LevelDebug, 100, time.Minute)
```

### Filtering Helper Functions

```go
// Condition helpers
LevelCondition(minLevel slog.Level) LogCondition
MessageContainsCondition(substring string) LogCondition
AttributeCondition(key, expectedValue string) LogCondition
TimeBasedCondition(startHour, endHour int) LogCondition
CombineConditions(conditions ...LogCondition) LogCondition  // AND logic
AnyCondition(conditions ...LogCondition) LogCondition       // OR logic

// Filter helpers
MaskFieldFilter(mask string) FieldFilter
RedactFieldFilter() FieldFilter
RegexMaskFilter(pattern, mask string) RegexFilter
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