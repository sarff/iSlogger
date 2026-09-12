# iSlogger

`iSlogger` is a standard-library-only logging package built on `log/slog`. It
writes daily files, optionally mirrors records to the console, filters sensitive
attributes, rate-limits noisy levels, and removes expired log files.

## Installation

```bash
go get github.com/sarff/iSlogger
```

The package clause uses `iSlogger`. The examples below give it the conventional
lowercase import alias:

```go
import islogger "github.com/sarff/iSlogger"
```

## Quick start

```go
package main

import (
	"log/slog"

	islogger "github.com/sarff/iSlogger"
)

func main() {
	config := islogger.DefaultConfig().
		WithAppName("myapp").
		WithLogLevel(slog.LevelDebug)

	logger, err := islogger.New(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	logger.Info("application started", "version", "1.0.0")
	logger.Error("request failed", "error", "connection timeout")
}
```

`DefaultConfig` creates two buffered files in `logs/` and enables console
output:

- `myapp_YYYY-MM-DD.log` contains every record allowed by `LogLevel`, filters,
  conditions, and rate limits.
- `myapp_error_YYYY-MM-DD.log` contains the same allowed WARN and higher
  records.
- DEBUG and INFO go to stdout. WARN and higher go to stderr. Each console
  record is written once.

## Configuration

| Field | Default | Meaning |
| --- | --- | --- |
| `LogDir` | `logs` | Directory for daily files |
| `AppName` | `app` | Safe file-name prefix |
| `LogLevel` | `INFO` | Minimum enabled `slog.Level` |
| `RetentionDays` | `7` | Previous calendar days retained |
| `JSONFormat` | `false` | JSON instead of text output |
| `AddSource` | `false` | Add the caller's file and line |
| `TimeFormat` | `time.RFC3339` | Go timestamp layout |
| `ConsoleOutput` | `true` | Enable stdout/stderr output |
| `BufferSize` | `8192` | Flush threshold; `0` writes directly |
| `FlushInterval` | `5s` | Periodic flush; `0` disables it |
| `FlushOnLevel` | `ERROR` | Immediately flush at or above this level |

`New` validates the final configuration. Negative retention, buffer, or flush
values, unsafe application names, invalid regular expressions, nil filters, and
invalid rate limits return an error. `Config.Validate` performs the same check
without opening files.

Configuration builders use copy-on-write behavior. Extending a derived config
does not mutate maps or slices in its source config.

```go
config := islogger.DefaultConfig().
	WithLogDir("service-logs").
	WithAppName("payments").
	WithLogLevel(slog.LevelInfo).
	WithRetentionDays(30).
	WithJSONFormat(true).
	WithAddSource(true).
	WithBufferSize(16 * 1024).
	WithFlushInterval(2 * time.Second).
	WithFlushOnLevel(slog.LevelWarn)
```

Use `WithoutBuffering` for direct file writes. `WithBuffering` restores the
default 8 KiB, five-second, ERROR-flush settings.

## Structured and context logging

`With` returns a logger with bound attributes. It shares files, log level,
rotation, buffers, and lifecycle with its parent. Calling `Close` on any shared
handle closes that shared logger.

```go
requestLogger := logger.With(
	"request_id", "req-123",
	"method", "POST",
)
requestLogger.Info("request completed", "status", 201)
```

Pass contexts at the call site, following the standard `slog` API:

```go
logger.InfoContext(ctx, "request started", "request_id", requestID)
logger.ErrorContext(ctx, "request failed", "error", err)
```

The package also provides `DebugContext`, `WarnContext`, and global context
functions. A context is forwarded to the handler; arbitrary values are not
automatically converted into attributes. `WithContext` remains available for
v1 compatibility but is deprecated.

## Filtering and conditions

Field filters apply recursively to call-site attributes, grouped attributes,
and attributes bound with `With`.

```go
config := islogger.DefaultConfig().
	WithFieldMask("password", "***").
	WithFieldRedaction("internal").
	WithRegexFilter(`\d{4}-\d{4}-\d{4}-\d{4}`, "****-****-****-****")
```

Regular-expression filters modify string attribute values. They do not rewrite
the log message. For standalone construction, `NewRegexFilter` returns a
compilation error. `RegexMaskFilter` is retained as a deprecated panic-based
compatibility helper.

Conditions added directly to a config use AND semantics:

```go
config := islogger.DefaultConfig().
	WithLevelCondition(slog.LevelInfo).
	WithAttributeCondition("tenant", "acme")
```

Use `AnyCondition` for OR behavior and `CombineConditions` for an explicit AND
group:

```go
config := islogger.DefaultConfig().WithCondition(islogger.AnyCondition(
	islogger.LevelCondition(slog.LevelWarn),
	islogger.AttributeCondition("critical", "true"),
))
```

`WithTimeBasedCondition(start, end)` uses local time and the half-open interval
`[start, end)`. An interval such as `22, 6` crosses midnight; equal hours allow
the full day.

Rate limits apply to an exact `slog.Level` after conditions pass:

```go
config := islogger.DefaultConfig().
	WithRateLimit(slog.LevelDebug, 100, time.Minute)
```

## Global logger

```go
if err := islogger.Init(islogger.DefaultConfig()); err != nil {
	panic(err)
}
defer islogger.Close()

islogger.Info("application started")
islogger.WarnContext(ctx, "request is slow", "duration", elapsed)
```

`Init` creates the replacement before swapping it into the global slot. If
creation fails, the existing global logger remains active. Logging calls are
safe no-ops when no global logger is installed.

## File management and shutdown

Daily rotation happens on the first accepted record after the local date
changes. The cleanup worker starts with the logger, recognizes only exact dated
filenames for the configured application, and stops during `Close`.

```go
files, err := logger.GetLogFiles()
mainPath, errorPath := logger.GetCurrentLogPaths()

err = logger.Cleanup() // synchronous and reports filesystem errors
err = logger.Reopen()  // for an external file-rotation tool
err = logger.Flush()
err = logger.Close()   // safe to call repeatedly
```

`RotateNow` is deprecated because reopening a daily filename does not create a
second rotation on the same date; it now aliases `Reopen`. `CleanupNow` is a
deprecated wrapper around `Cleanup` that discards the returned error.

Always close an owned logger. `Close` stops cleanup and auto-flush goroutines,
flushes pending records, and closes both files.

## Development

The module requires Go 1.24.1 or newer and has no runtime dependencies.

```bash
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

See [TODO.md](TODO.md) for planned work.
