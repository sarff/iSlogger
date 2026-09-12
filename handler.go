package iSlogger

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type pipelineHandler struct {
	core   *loggerCore
	bound  []slog.Attr
	groups []string
}

func newPipelineHandler(core *loggerCore) *pipelineHandler {
	return &pipelineHandler{core: core}
}

func (h *pipelineHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.core.enabled(level)
}

func (h *pipelineHandler) Handle(ctx context.Context, record slog.Record) error {
	attrs := append([]slog.Attr(nil), h.bound...)
	recordAttrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		recordAttrs = append(recordAttrs, attr)
		return true
	})
	attrs = append(attrs, nestAttrs(recordAttrs, h.groups)...)

	if !conditionsAllow(h.core.config.Filters.Conditions, record.Level, record.Message, attrs) {
		return nil
	}
	if !h.core.limiter.allow(record.Level, h.core.now()) {
		return nil
	}

	filtered := filterAttrs(attrs, h.core.config.Filters)
	clean := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	clean.AddAttrs(filtered...)
	return h.core.route(ctx, clean)
}

func (h *pipelineHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := &pipelineHandler{
		core:   h.core,
		bound:  append([]slog.Attr(nil), h.bound...),
		groups: append([]string(nil), h.groups...),
	}
	clone.bound = append(clone.bound, nestAttrs(attrs, clone.groups)...)
	return clone
}

func (h *pipelineHandler) WithGroup(name string) slog.Handler {
	clone := &pipelineHandler{
		core:   h.core,
		bound:  append([]slog.Attr(nil), h.bound...),
		groups: append([]string(nil), h.groups...),
	}
	if name != "" {
		clone.groups = append(clone.groups, name)
	}
	return clone
}

func nestAttrs(attrs []slog.Attr, groups []string) []slog.Attr {
	if len(attrs) == 0 || len(groups) == 0 {
		return attrs
	}
	nested := append([]slog.Attr(nil), attrs...)
	for i := len(groups) - 1; i >= 0; i-- {
		nested = []slog.Attr{{Key: groups[i], Value: slog.GroupValue(nested...)}}
	}
	return nested
}

func conditionsAllow(conditions []LogCondition, level slog.Level, msg string, attrs []slog.Attr) bool {
	for _, condition := range conditions {
		if !condition(level, msg, attrs) {
			return false
		}
	}
	return true
}

func filterAttrs(attrs []slog.Attr, config FilterConfig) []slog.Attr {
	filtered := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		attr.Value = attr.Value.Resolve()
		if attr.Value.Kind() == slog.KindGroup {
			children := filterAttrs(attr.Value.Group(), config)
			if len(children) == 0 {
				continue
			}
			attr.Value = slog.GroupValue(children...)
		}
		if filter, ok := config.FieldFilters[attr.Key]; ok {
			attr.Value = filter(attr.Key, attr.Value).Resolve()
		}
		if _, redacted := attr.Value.Any().(redactedValue); redacted {
			continue
		}
		if attr.Value.Kind() == slog.KindString {
			value := attr.Value.String()
			for _, filter := range config.RegexFilters {
				value = filter.Pattern.ReplaceAllString(value, filter.Replacement)
			}
			attr.Value = slog.StringValue(value)
		}
		filtered = append(filtered, attr)
	}
	return filtered
}

type rateLimiter struct {
	mu      sync.Mutex
	limits  map[slog.Level]RateLimit
	buckets map[slog.Level]rateBucket
}

type rateBucket struct {
	count int
	since time.Time
}

func newRateLimiter(limits map[slog.Level]RateLimit) *rateLimiter {
	return &rateLimiter{limits: limits, buckets: make(map[slog.Level]rateBucket)}
}

func (r *rateLimiter) allow(level slog.Level, now time.Time) bool {
	limit, ok := r.limits[level]
	if !ok {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	bucket := r.buckets[level]
	if bucket.since.IsZero() || now.Sub(bucket.since) >= limit.Period {
		bucket = rateBucket{since: now}
	}
	if bucket.count >= limit.MaxCount {
		r.buckets[level] = bucket
		return false
	}
	bucket.count++
	r.buckets[level] = bucket
	return true
}
