package iSlogger

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"
)

// filteredHandler wraps slog.Handler and applies filtering logic
type filteredHandler struct {
	handler slog.Handler
	config  FilterConfig
}

// newFilteredHandler creates a new filtered handler
func newFilteredHandler(handler slog.Handler, config FilterConfig) *filteredHandler {
	return &filteredHandler{
		handler: handler,
		config:  config,
	}
}

// Enabled checks if the handler is enabled for the given level
func (h *filteredHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle processes the log record with filtering
func (h *filteredHandler) Handle(ctx context.Context, record slog.Record) error {
	// Apply rate limiting first
	if !h.checkRateLimit(record.Level) {
		return nil // Skip if rate limited
	}

	// Extract attributes for condition checking
	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	// Apply conditions
	if !h.shouldLog(record.Level, record.Message, attrs) {
		return nil // Skip if conditions not met
	}

	// Apply field filters
	filteredAttrs := h.applyFieldFilters(attrs)

	// Create new record with filtered attributes
	newRecord := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	for _, attr := range filteredAttrs {
		newRecord.AddAttrs(attr)
	}

	return h.handler.Handle(ctx, newRecord)
}

// WithAttrs creates a new handler with additional attributes
func (h *filteredHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &filteredHandler{
		handler: h.handler.WithAttrs(attrs),
		config:  h.config,
	}
}

// WithGroup creates a new handler with a group
func (h *filteredHandler) WithGroup(name string) slog.Handler {
	return &filteredHandler{
		handler: h.handler.WithGroup(name),
		config:  h.config,
	}
}

// shouldLog checks if the log entry should be written based on conditions
func (h *filteredHandler) shouldLog(level slog.Level, msg string, attrs []slog.Attr) bool {
	// If no conditions are set, log everything
	if len(h.config.Conditions) == 0 {
		return true
	}

	// All conditions must pass (AND logic)
	for _, condition := range h.config.Conditions {
		if !condition(level, msg, attrs) {
			return false
		}
	}
	return true
}

// applyFieldFilters applies field filters to attributes
func (h *filteredHandler) applyFieldFilters(attrs []slog.Attr) []slog.Attr {
	if len(h.config.FieldFilters) == 0 && len(h.config.RegexFilters) == 0 {
		return attrs
	}

	filtered := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		filteredAttr := h.applyFiltersToAttr(attr)
		if filteredAttr.Value.String() != "" { // Only include non-empty values
			filtered = append(filtered, filteredAttr)
		}
	}
	return filtered
}

// applyFiltersToAttr applies filters to a single attribute
func (h *filteredHandler) applyFiltersToAttr(attr slog.Attr) slog.Attr {
	// Apply field-specific filters
	if filter, exists := h.config.FieldFilters[attr.Key]; exists {
		attr.Value = filter(attr.Key, attr.Value)
	}

	// Apply regex filters to string values
	if attr.Value.Kind() == slog.KindString {
		strVal := attr.Value.String()
		for _, regexFilter := range h.config.RegexFilters {
			strVal = regexFilter.Pattern.ReplaceAllString(strVal, regexFilter.Replacement)
		}
		attr.Value = slog.StringValue(strVal)
	}

	return attr
}

// checkRateLimit checks if the log entry should be rate limited
func (h *filteredHandler) checkRateLimit(level slog.Level) bool {
	rateLimitPtr, exists := h.config.RateLimits[level]
	if !exists {
		return true // No rate limit set, allow
	}

	// Make a copy to work with
	rateLimit := rateLimitPtr
	now := time.Now()

	// Check if we need to reset the counter
	if now.Sub(rateLimit.lastReset) >= rateLimit.Period {
		atomic.StoreInt64(&rateLimit.counter, 0)
		rateLimit.lastReset = now
		// Update the config map
		h.config.RateLimits[level] = rateLimit
	}

	// Check if we're under the limit
	current := atomic.AddInt64(&rateLimit.counter, 1)
	if current <= int64(rateLimit.MaxCount) {
		// Update the config map
		h.config.RateLimits[level] = rateLimit
		return true
	}

	return false // Rate limited
}
