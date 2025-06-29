package iSlogger

import (
	"log/slog"
	"regexp"
	"time"
)

type Config struct {
	LogDir        string // Directory for log files
	AppName       string // Application name for log file prefix
	Debug         bool   // Enable debug logging
	RetentionDays int    // Number of days to keep log files
	JSONFormat    bool   // Use JSON format instead of text
	AddSource     bool   // Add source file and line info
	TimeFormat    string // Custom time format

	// Filtering configuration
	Filters FilterConfig // Filtering and conditional logging configuration
}

func DefaultConfig() Config {
	return Config{
		LogDir:        "logs",
		AppName:       "app",
		Debug:         false,
		RetentionDays: 7,
		JSONFormat:    false,
		AddSource:     false,
		TimeFormat:    time.RFC3339, // "2006-01-02T15:04:05Z07:00"
		Filters:       DefaultFilterConfig(),
	}
}

// WithDebug enables debug logging
func (c Config) WithDebug(debug bool) Config {
	c.Debug = debug
	return c
}

// WithLogDir sets the log directory
func (c Config) WithLogDir(dir string) Config {
	c.LogDir = dir
	return c
}

// WithAppName sets the application name
func (c Config) WithAppName(name string) Config {
	c.AppName = name
	return c
}

// WithRetentionDays sets the retention period
func (c Config) WithRetentionDays(days int) Config {
	c.RetentionDays = days
	return c
}

// WithJSONFormat enables JSON format
func (c Config) WithJSONFormat(json bool) Config {
	c.JSONFormat = json
	return c
}

// WithTimeFormat sets custom time format
func (c Config) WithTimeFormat(format string) Config {
	c.TimeFormat = format
	return c
}

// WithAddSource enables Source
func (c Config) WithAddSource(source bool) Config {
	c.AddSource = source
	return c
}

// Filtering configuration methods

// WithCondition adds a conditional logging function
func (c Config) WithCondition(condition LogCondition) Config {
	c.Filters.Conditions = append(c.Filters.Conditions, condition)
	return c
}

// WithFieldFilter adds a field filter for a specific key
func (c Config) WithFieldFilter(key string, filter FieldFilter) Config {
	if c.Filters.FieldFilters == nil {
		c.Filters.FieldFilters = make(map[string]FieldFilter)
	}
	c.Filters.FieldFilters[key] = filter
	return c
}

// WithFieldMask masks a field with the given mask string
func (c Config) WithFieldMask(key string, mask string) Config {
	return c.WithFieldFilter(key, MaskFieldFilter(mask))
}

// WithFieldRedaction completely removes a field
func (c Config) WithFieldRedaction(key string) Config {
	return c.WithFieldFilter(key, RedactFieldFilter())
}

// WithRegexFilter adds a regex-based filter
func (c Config) WithRegexFilter(pattern string, replacement string) Config {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		// Skip invalid regex patterns
		return c
	}
	c.Filters.RegexFilters = append(c.Filters.RegexFilters, RegexFilter{
		Pattern:     regex,
		Replacement: replacement,
	})
	return c
}

// WithRateLimit adds rate limiting for a specific log level
func (c Config) WithRateLimit(level slog.Level, maxCount int, period time.Duration) Config {
	if c.Filters.RateLimits == nil {
		c.Filters.RateLimits = make(map[slog.Level]RateLimit)
	}
	c.Filters.RateLimits[level] = RateLimit{
		MaxCount: maxCount,
		Period:   period,
	}
	return c
}

// WithLevelCondition adds a minimum level condition
func (c Config) WithLevelCondition(minLevel slog.Level) Config {
	return c.WithCondition(LevelCondition(minLevel))
}

// WithMessageContainsCondition adds a message content condition
func (c Config) WithMessageContainsCondition(substring string) Config {
	return c.WithCondition(MessageContainsCondition(substring))
}

// WithAttributeCondition adds an attribute-based condition
func (c Config) WithAttributeCondition(key, value string) Config {
	return c.WithCondition(AttributeCondition(key, value))
}

// WithTimeBasedCondition adds a time-based condition
func (c Config) WithTimeBasedCondition(startHour, endHour int) Config {
	return c.WithCondition(TimeBasedCondition(startHour, endHour))
}
