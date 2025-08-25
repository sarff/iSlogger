# iSlogger 📝

Production-ready Go logging package built on top of `slog` with automatic file rotation, dual output streams, and flexible configuration for enterprise applications.

## 🚀 Features

- **Dual Output Streams**: Separate files for general logs and errors
- **Automatic File Rotation**: Daily rotation with configurable retention
- **Flexible Log Levels**: Support for all slog levels (DEBUG, INFO, WARN, ERROR)
- **Structured Logging**: Built on Go's standard `slog` package
- **Thread-Safe**: Concurrent logging support with mutex protection
- **Zero Dependencies**: Uses only Go standard library
- **Flexible Configuration**: Builder pattern for easy setup
- **Global & Instance Loggers**: Use global functions or create instances
- **Context Support**: Context-aware logging for request tracking
- **🔒 Field Filtering**: Mask or redact sensitive data (passwords, tokens, etc.)
- **🎯 Conditional Logging**: Log only when specific conditions are met
- **⚡ Rate Limiting**: Prevent log flooding with per-level rate limits
- **🚀 Buffered Writes**: High-performance buffered I/O with intelligent flushing
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
    WithLogLevel(slog.LevelDebug).
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
| `LogLevel` | `INFO` | Minimum log level (DEBUG, INFO, WARN, ERROR) |
| `RetentionDays` | `7` | Days to keep old log files |
| `JSONFormat` | `false` | Use JSON format instead of text |
| `AddSource` | `false` | Include source file and line info |
| `TimeFormat` | `RFC3339` | Custom time format |
| `BufferSize` | `8192` | Buffer size in bytes (0 = no buffering) |
| `FlushInterval` | `5s` | Time interval for automatic buffer flushing |
| `FlushOnLevel` | `ERROR` | Minimum level that triggers immediate flush |

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

### Log Level Management

```go
// Start with WARN level (only warnings and errors)
logger.SetLevel(slog.LevelWarn)
logger.Debug("Won't be logged")
logger.Info("Won't be logged") 
logger.Warn("Will be logged")
logger.Error("Will be logged")

// Switch to DEBUG level (all messages)
logger.SetLevel(slog.LevelDebug)
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

## 🚀 Buffered Writes & Performance

Boost logging performance with intelligent buffering that reduces I/O operations while ensuring critical messages are never lost:

### Basic Buffering Configuration

```go
// Enable buffering with default settings (8KB buffer, 5s flush, ERROR flush)
config := islogger.DefaultConfig().WithBuffering()

// Or customize buffering parameters
config := islogger.DefaultConfig().
    WithBufferSize(16384).                        // 16KB buffer
    WithFlushInterval(10 * time.Second).          // Auto-flush every 10 seconds  
    WithFlushOnLevel(slog.LevelWarn)              // Immediately flush WARN+ levels

logger, _ := islogger.New(config)
defer logger.Close() // Automatically flushes buffers
```

### Disable Buffering for Real-time Logging

```go
// For applications requiring immediate log writes
config := islogger.DefaultConfig().WithoutBuffering()

logger, _ := islogger.New(config)
logger.Info("This writes immediately to disk") // No buffering
```

### Manual Buffer Control

```go
logger := islogger.New(islogger.DefaultConfig().WithBuffering())

// Log messages (buffered)
logger.Info("Processing request", "id", 1)
logger.Info("Processing request", "id", 2)
logger.Info("Processing request", "id", 3)

// Force flush at critical points
if err := logger.Flush(); err != nil {
    log.Printf("Failed to flush logs: %v", err)
}

// Critical messages flush immediately (based on FlushOnLevel)
logger.Error("Critical error occurred") // Flushes immediately
```

### High-Performance Web Server Example

```go
// Optimized for high-throughput applications
config := islogger.DefaultConfig().
    WithAppName("webserver").
    WithBufferSize(32768).                     // 32KB buffer for high volume
    WithFlushInterval(2 * time.Second).        // Frequent auto-flush
    WithFlushOnLevel(slog.LevelWarn).          // Immediate flush for warnings+
    WithRateLimit(slog.LevelDebug, 1000, time.Minute) // Prevent debug spam

logger, _ := islogger.New(config)
defer logger.Close()

// Handle thousands of requests with minimal I/O
http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    
    // These are buffered for performance
    logger.Info("Request started", "path", r.URL.Path, "method", r.Method)
    
    // ... process request ...
    
    if err != nil {
        // This flushes immediately due to FlushOnLevel=WARN
        logger.Error("Request failed", "error", err, "duration", time.Since(start))
        http.Error(w, "Internal Server Error", 500)
        return
    }
    
    // This is buffered
    logger.Info("Request completed", "duration", time.Since(start))
})

// Periodic flush for long-running operations  
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        logger.Flush() // Ensure logs are written regularly
    }
}()
```

### Buffering Configuration Methods

| Method | Description |
|--------|-------------|
| `WithBuffering()` | Enable buffering with defaults (8KB, 5s, ERROR flush) |
| `WithoutBuffering()` | Disable buffering for real-time logging |
| `WithBufferSize(bytes)` | Set custom buffer size (0 = no buffering) |
| `WithFlushInterval(duration)` | Set automatic flush interval |
| `WithFlushOnLevel(level)` | Set minimum level for immediate flush |

### Buffer Flushing Strategies

1. **Size-based**: Buffer flushes when full (prevents memory overflow)
2. **Time-based**: Automatic flush at configured intervals (prevents stale logs)  
3. **Level-based**: Immediate flush for high-priority messages (ensures critical logs)
4. **Manual**: Explicit control with `Flush()` method (for critical sections)
5. **Shutdown**: Automatic flush on `Close()` (prevents data loss)

### Performance Benefits

- **Reduced I/O**: Batch multiple log entries into single disk writes
- **Lower Latency**: Non-blocking writes for most log levels  
- **Higher Throughput**: Handle thousands of logs per second efficiently
- **Intelligent Flushing**: Critical messages bypass buffering for reliability
- **Memory Efficient**: Fixed-size buffers prevent memory growth

## 🏭 Production Configuration

Complete example for production environments:

```go
config := islogger.DefaultConfig().
    WithAppName("myapp").
    WithLogDir("/var/log/myapp").
    WithLogLevel(slog.LevelWarn).
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
    WithRateLimit(slog.LevelDebug, 100, time.Minute).
    // High-performance buffering
    WithBufferSize(16384).                     // 16KB buffer for production
    WithFlushInterval(3 * time.Second).        // Quick flush for responsiveness
    WithFlushOnLevel(slog.LevelWarn)           // Immediate flush for warnings+

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
SetLevel(level slog.Level) error
Flush() error
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
SetLevel(level slog.Level) error
Flush() error
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
    WithLogLevel(slog.LevelDebug).
    WithRetentionDays(30).
    WithJSONFormat(true).
    WithTimeFormat("2006-01-02 15:04:05").
    WithAddSource(true).
    // Add filtering
    WithFieldMask("password", "***").
    WithRegexFilter(`\d{4}-\d{4}-\d{4}-\d{4}`, "****-****-****-****").
    WithLevelCondition(slog.LevelInfo).
    WithRateLimit(slog.LevelDebug, 100, time.Minute).
    // Add buffering
    WithBufferSize(16384).
    WithFlushInterval(5 * time.Second).
    WithFlushOnLevel(slog.LevelWarn)
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

- **Debug**: Detailed information for debugging (when LogLevel is DEBUG)
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
## 🚀 Roadmap & Tasks

See [TODO.md](TODO.md) for current tasks and progress tracking.

Made with ❤️ in Ukraine 🇺🇦