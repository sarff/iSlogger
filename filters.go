package iSlogger

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

// LogCondition decides whether a record is accepted.
type LogCondition func(level slog.Level, msg string, attrs []slog.Attr) bool

// FieldFilter transforms an attribute value.
type FieldFilter func(key string, value slog.Value) slog.Value

// FilterConfig groups conditional, field, regex, and rate-limit settings.
type FilterConfig struct {
	Conditions   []LogCondition
	FieldFilters map[string]FieldFilter
	RegexFilters []RegexFilter
	RateLimits   map[slog.Level]RateLimit
}

// RegexFilter replaces matches in every string attribute value.
type RegexFilter struct {
	Pattern     *regexp.Regexp
	Replacement string
}

// RateLimit defines a fixed-window limit for an exact slog level.
type RateLimit struct {
	MaxCount int
	Period   time.Duration
}

// DefaultFilterConfig returns filtering with no restrictions.
func DefaultFilterConfig() FilterConfig {
	return FilterConfig{
		Conditions:   []LogCondition{},
		FieldFilters: make(map[string]FieldFilter),
		RegexFilters: []RegexFilter{},
		RateLimits:   make(map[slog.Level]RateLimit),
	}
}

// MaskFieldFilter returns a filter that replaces a value with mask.
func MaskFieldFilter(mask string) FieldFilter {
	return func(string, slog.Value) slog.Value { return slog.StringValue(mask) }
}

type redactedValue struct{}

func (redactedValue) String() string { return "" }

// RedactFieldFilter returns a filter that removes an attribute.
func RedactFieldFilter() FieldFilter {
	return func(string, slog.Value) slog.Value { return slog.AnyValue(redactedValue{}) }
}

// NewRegexFilter compiles a safe regex filter.
func NewRegexFilter(pattern, replacement string) (RegexFilter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return RegexFilter{}, fmt.Errorf("compile regex filter: %w", err)
	}
	return RegexFilter{Pattern: re, Replacement: replacement}, nil
}

// RegexMaskFilter creates a regex filter that masks matching patterns.
// Deprecated: use NewRegexFilter to handle invalid patterns without a panic.
func RegexMaskFilter(pattern, mask string) RegexFilter {
	filter, err := NewRegexFilter(pattern, mask)
	if err != nil {
		panic(err)
	}
	return filter
}

// LevelCondition accepts records at or above minLevel.
func LevelCondition(minLevel slog.Level) LogCondition {
	return func(level slog.Level, _ string, _ []slog.Attr) bool { return level >= minLevel }
}

// MessageContainsCondition accepts messages containing substring.
func MessageContainsCondition(substring string) LogCondition {
	return func(_ slog.Level, msg string, _ []slog.Attr) bool { return strings.Contains(msg, substring) }
}

// AttributeCondition accepts records with a matching attribute.
func AttributeCondition(key, expectedValue string) LogCondition {
	return func(_ slog.Level, _ string, attrs []slog.Attr) bool {
		return walkAttrs(attrs, func(attr slog.Attr) bool {
			return attr.Key == key && attr.Value.Resolve().String() == expectedValue
		})
	}
}

// TimeBasedCondition allows [startHour, endHour). An interval that crosses
// midnight is supported; equal hours mean the whole day.
func TimeBasedCondition(startHour, endHour int) LogCondition {
	return func(_ slog.Level, _ string, _ []slog.Attr) bool {
		hour := time.Now().Hour()
		if startHour == endHour {
			return true
		}
		if startHour < endHour {
			return hour >= startHour && hour < endHour
		}
		return hour >= startHour || hour < endHour
	}
}

// CombineConditions combines conditions with AND semantics.
func CombineConditions(conditions ...LogCondition) LogCondition {
	return func(level slog.Level, msg string, attrs []slog.Attr) bool {
		for _, condition := range conditions {
			if condition == nil || !condition(level, msg, attrs) {
				return false
			}
		}
		return true
	}
}

// AnyCondition combines conditions with OR semantics.
func AnyCondition(conditions ...LogCondition) LogCondition {
	return func(level slog.Level, msg string, attrs []slog.Attr) bool {
		for _, condition := range conditions {
			if condition != nil && condition(level, msg, attrs) {
				return true
			}
		}
		return false
	}
}

func walkAttrs(attrs []slog.Attr, match func(slog.Attr) bool) bool {
	for _, attr := range attrs {
		attr.Value = attr.Value.Resolve()
		if match(attr) {
			return true
		}
		if attr.Value.Kind() == slog.KindGroup && walkAttrs(attr.Value.Group(), match) {
			return true
		}
	}
	return false
}
