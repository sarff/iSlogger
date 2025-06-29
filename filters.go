package iSlogger

import (
	"log/slog"
	"regexp"
	"strings"
	"time"
)

// LogCondition defines a function that determines whether a log entry should be written
type LogCondition func(level slog.Level, msg string, attrs []slog.Attr) bool

// FieldFilter defines a function that filters/modifies field values
type FieldFilter func(key string, value slog.Value) slog.Value

// FilterConfig holds all filtering configuration
type FilterConfig struct {
	// Conditional logging
	Conditions []LogCondition

	// Field filters
	FieldFilters map[string]FieldFilter
	RegexFilters []RegexFilter

	// Rate limiting
	RateLimits map[slog.Level]RateLimit
}

// RegexFilter defines a regex-based field filter
type RegexFilter struct {
	Pattern     *regexp.Regexp
	Replacement string
}

// RateLimit defines rate limiting configuration
type RateLimit struct {
	MaxCount  int           // Maximum number of logs per period
	Period    time.Duration // Time period for rate limiting
	counter   int64         // Internal counter
	lastReset time.Time     // Internal last reset time
}

// DefaultFilterConfig returns default filter configuration
func DefaultFilterConfig() FilterConfig {
	return FilterConfig{
		Conditions:   []LogCondition{},
		FieldFilters: make(map[string]FieldFilter),
		RegexFilters: []RegexFilter{},
		RateLimits:   make(map[slog.Level]RateLimit),
	}
}

// Common field filters

// MaskFieldFilter masks a field with the given mask
func MaskFieldFilter(mask string) FieldFilter {
	return func(key string, value slog.Value) slog.Value {
		return slog.StringValue(mask)
	}
}

// RedactFieldFilter completely removes the field by setting it to empty
func RedactFieldFilter() FieldFilter {
	return func(key string, value slog.Value) slog.Value {
		return slog.StringValue("")
	}
}

// RegexMaskFilter creates a regex filter that masks matching patterns
func RegexMaskFilter(pattern string, mask string) RegexFilter {
	return RegexFilter{
		Pattern:     regexp.MustCompile(pattern),
		Replacement: mask,
	}
}

// Common conditions

// LevelCondition creates a condition that only allows logs at or above specified level
func LevelCondition(minLevel slog.Level) LogCondition {
	return func(level slog.Level, msg string, attrs []slog.Attr) bool {
		return level >= minLevel
	}
}

// MessageContainsCondition creates a condition based on message content
func MessageContainsCondition(substring string) LogCondition {
	return func(level slog.Level, msg string, attrs []slog.Attr) bool {
		return strings.Contains(msg, substring)
	}
}

// AttributeCondition creates a condition based on attribute values
func AttributeCondition(key string, expectedValue string) LogCondition {
	return func(level slog.Level, msg string, attrs []slog.Attr) bool {
		for _, attr := range attrs {
			if attr.Key == key && attr.Value.String() == expectedValue {
				return true
			}
		}
		return false
	}
}

// TimeBasedCondition creates a condition based on time of day
func TimeBasedCondition(startHour, endHour int) LogCondition {
	return func(level slog.Level, msg string, attrs []slog.Attr) bool {
		hour := time.Now().Hour()
		return hour >= startHour && hour <= endHour
	}
}

// CombineConditions combines multiple conditions with AND logic
func CombineConditions(conditions ...LogCondition) LogCondition {
	return func(level slog.Level, msg string, attrs []slog.Attr) bool {
		for _, condition := range conditions {
			if !condition(level, msg, attrs) {
				return false
			}
		}
		return true
	}
}

// AnyCondition combines multiple conditions with OR logic
func AnyCondition(conditions ...LogCondition) LogCondition {
	return func(level slog.Level, msg string, attrs []slog.Attr) bool {
		for _, condition := range conditions {
			if condition(level, msg, attrs) {
				return true
			}
		}
		return false
	}
}
