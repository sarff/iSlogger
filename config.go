package iSlogger

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Config controls file output, formatting, buffering, and filtering.
type Config struct {
	LogDir        string
	AppName       string
	LogLevel      slog.Level
	RetentionDays int
	JSONFormat    bool
	AddSource     bool
	TimeFormat    string
	ConsoleOutput bool

	BufferSize    int
	FlushInterval time.Duration
	FlushOnLevel  slog.Level

	Filters FilterConfig

	validationErr error
}

// DefaultConfig returns a production-ready configuration with console output
// and buffered daily files in the logs directory.
func DefaultConfig() Config {
	return Config{
		LogDir:        "logs",
		AppName:       "app",
		LogLevel:      slog.LevelInfo,
		RetentionDays: 7,
		TimeFormat:    time.RFC3339,
		ConsoleOutput: true,
		BufferSize:    8192,
		FlushInterval: 5 * time.Second,
		FlushOnLevel:  slog.LevelError,
		Filters:       DefaultFilterConfig(),
	}
}

func (c Config) withDefaults() Config {
	if c.LogDir == "" {
		c.LogDir = "logs"
	}
	if c.AppName == "" {
		c.AppName = "app"
	}
	if c.RetentionDays == 0 {
		c.RetentionDays = 7
	}
	if c.TimeFormat == "" {
		c.TimeFormat = time.RFC3339
	}
	return c
}

// Validate checks configuration values without creating a logger.
func (c Config) Validate() error {
	var errs []error
	if c.validationErr != nil {
		errs = append(errs, c.validationErr)
	}
	if c.RetentionDays < 0 {
		errs = append(errs, fmt.Errorf("retention days must not be negative"))
	}
	if c.BufferSize < 0 {
		errs = append(errs, fmt.Errorf("buffer size must not be negative"))
	}
	if c.FlushInterval < 0 {
		errs = append(errs, fmt.Errorf("flush interval must not be negative"))
	}
	if c.AppName == "." || c.AppName == ".." || strings.ContainsAny(c.AppName, "/\\\x00") {
		errs = append(errs, fmt.Errorf("app name must be a file-name component"))
	}
	for i, condition := range c.Filters.Conditions {
		if condition == nil {
			errs = append(errs, fmt.Errorf("condition %d is nil", i))
		}
	}
	for key, filter := range c.Filters.FieldFilters {
		if filter == nil {
			errs = append(errs, fmt.Errorf("field filter %q is nil", key))
		}
	}
	for i, filter := range c.Filters.RegexFilters {
		if filter.Pattern == nil {
			errs = append(errs, fmt.Errorf("regex filter %d has a nil pattern", i))
		}
	}
	for level, limit := range c.Filters.RateLimits {
		if limit.MaxCount <= 0 || limit.Period <= 0 {
			errs = append(errs, fmt.Errorf("rate limit for %s requires a positive count and period", level))
		}
	}
	return errors.Join(errs...)
}

// WithLogLevel sets the minimum enabled level.
func (c Config) WithLogLevel(level slog.Level) Config { c.LogLevel = level; return c }

// WithLogDir sets the directory used for daily files.
func (c Config) WithLogDir(dir string) Config { c.LogDir = dir; return c }

// WithAppName sets the file-name prefix.
func (c Config) WithAppName(name string) Config { c.AppName = name; return c }

// WithRetentionDays sets the number of previous calendar days to retain.
func (c Config) WithRetentionDays(days int) Config { c.RetentionDays = days; return c }

// WithJSONFormat selects JSON output when enabled and text output otherwise.
func (c Config) WithJSONFormat(json bool) Config { c.JSONFormat = json; return c }

// WithTimeFormat sets the Go layout used for record timestamps.
func (c Config) WithTimeFormat(format string) Config { c.TimeFormat = format; return c }

// WithAddSource includes the caller's file and line when enabled.
func (c Config) WithAddSource(source bool) Config { c.AddSource = source; return c }

// WithConsoleOutput enables or disables stdout/stderr output.
func (c Config) WithConsoleOutput(enabled bool) Config {
	c.ConsoleOutput = enabled
	return c
}

// WithCondition adds a condition; all added conditions must pass.
func (c Config) WithCondition(condition LogCondition) Config {
	c.Filters = cloneFilterConfig(c.Filters)
	c.Filters.Conditions = append(c.Filters.Conditions, condition)
	return c
}

// WithFieldFilter adds a filter for an attribute key.
func (c Config) WithFieldFilter(key string, filter FieldFilter) Config {
	c.Filters = cloneFilterConfig(c.Filters)
	c.Filters.FieldFilters[key] = filter
	return c
}

// WithFieldMask replaces the value of key with mask.
func (c Config) WithFieldMask(key, mask string) Config {
	return c.WithFieldFilter(key, MaskFieldFilter(mask))
}

// WithFieldRedaction removes attributes with key.
func (c Config) WithFieldRedaction(key string) Config {
	return c.WithFieldFilter(key, RedactFieldFilter())
}

// WithRegexFilter replaces pattern matches in string attribute values.
func (c Config) WithRegexFilter(pattern, replacement string) Config {
	c.Filters = cloneFilterConfig(c.Filters)
	filter, err := NewRegexFilter(pattern, replacement)
	if err != nil {
		c.validationErr = errors.Join(c.validationErr, err)
		return c
	}
	c.Filters.RegexFilters = append(c.Filters.RegexFilters, filter)
	return c
}

// WithRateLimit limits accepted records at an exact level in each period.
func (c Config) WithRateLimit(level slog.Level, maxCount int, period time.Duration) Config {
	c.Filters = cloneFilterConfig(c.Filters)
	c.Filters.RateLimits[level] = RateLimit{MaxCount: maxCount, Period: period}
	return c
}

// WithLevelCondition requires records to be at or above level.
func (c Config) WithLevelCondition(level slog.Level) Config {
	return c.WithCondition(LevelCondition(level))
}

// WithMessageContainsCondition requires the message to contain substring.
func (c Config) WithMessageContainsCondition(substring string) Config {
	return c.WithCondition(MessageContainsCondition(substring))
}

// WithAttributeCondition requires a matching attribute, including bound and grouped attributes.
func (c Config) WithAttributeCondition(key, value string) Config {
	return c.WithCondition(AttributeCondition(key, value))
}

// WithTimeBasedCondition allows records in the local-time interval [startHour, endHour).
func (c Config) WithTimeBasedCondition(startHour, endHour int) Config {
	if startHour < 0 || startHour > 23 || endHour < 0 || endHour > 23 {
		c.validationErr = errors.Join(c.validationErr, fmt.Errorf("time-based condition hours must be between 0 and 23"))
		return c
	}
	return c.WithCondition(TimeBasedCondition(startHour, endHour))
}

// WithBufferSize sets the buffer threshold in bytes; zero disables buffering.
func (c Config) WithBufferSize(size int) Config { c.BufferSize = size; return c }

// WithFlushInterval sets the periodic buffer flush interval; zero disables it.
func (c Config) WithFlushInterval(interval time.Duration) Config {
	c.FlushInterval = interval
	return c
}

// WithFlushOnLevel flushes file buffers after a record at or above level.
func (c Config) WithFlushOnLevel(level slog.Level) Config { c.FlushOnLevel = level; return c }

// WithoutBuffering writes each record directly to its files.
func (c Config) WithoutBuffering() Config { c.BufferSize = 0; return c }

// WithBuffering restores the default buffering settings.
func (c Config) WithBuffering() Config {
	c.BufferSize = 8192
	c.FlushInterval = 5 * time.Second
	c.FlushOnLevel = slog.LevelError
	return c
}

func cloneFilterConfig(src FilterConfig) FilterConfig {
	dst := FilterConfig{
		Conditions:   append([]LogCondition(nil), src.Conditions...),
		RegexFilters: append([]RegexFilter(nil), src.RegexFilters...),
		FieldFilters: make(map[string]FieldFilter, len(src.FieldFilters)),
		RateLimits:   make(map[slog.Level]RateLimit, len(src.RateLimits)),
	}
	for key, filter := range src.FieldFilters {
		dst.FieldFilters[key] = filter
	}
	for level, limit := range src.RateLimits {
		dst.RateLimits[level] = limit
	}
	return dst
}
