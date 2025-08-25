package iSlogger

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestFieldMasking(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-mask").
		WithLogDir("test-logs-mask").
		WithLogLevel(slog.LevelDebug).
		WithFieldMask("password", "***").
		WithFieldMask("credit_card", "****-****-****-****")

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-mask")

	logger.Info("User login", "username", "john", "password", "secret123", "credit_card", "1234-5678-9012-3456")

	// Test should complete without errors
	// In real test, we would check the log file contents
}

func TestFieldRedaction(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-redact").
		WithLogDir("test-logs-redact").
		WithLogLevel(slog.LevelDebug).
		WithFieldRedaction("sensitive_data")

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-redact")

	logger.Info("Processing", "user_id", 123, "sensitive_data", "should_not_appear")

	// Test should complete without errors
}

func TestRegexFilter(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-regex").
		WithLogDir("test-logs-regex").
		WithLogLevel(slog.LevelDebug).
		WithRegexFilter(`\d{4}-\d{4}-\d{4}-\d{4}`, "****-****-****-****").                    // Credit card pattern
		WithRegexFilter(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, "***@***.***") // Email pattern

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-regex")

	logger.Info("Payment processed", "message", "Card 1234-5678-9012-3456 charged for user@example.com")

	// Test should complete without errors
}

func TestConditionalLogging(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-condition").
		WithLogDir("test-logs-condition").
		WithLogLevel(slog.LevelDebug).
		WithLevelCondition(slog.LevelWarn).       // Only WARN and above
		WithMessageContainsCondition("important") // OR contains "important"

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-condition")

	logger.Debug("Regular debug")  // Should NOT be logged
	logger.Info("Regular info")    // Should NOT be logged
	logger.Info("Important info")  // Should be logged (contains "important")
	logger.Warn("Warning message") // Should be logged (WARN level)
	logger.Error("Error message")  // Should be logged (ERROR level)

	// Test should complete without errors
}

func TestAttributeCondition(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-attr").
		WithLogDir("test-logs-attr").
		WithLogLevel(slog.LevelDebug).
		WithAttributeCondition("user_type", "admin") // Only admin users

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-attr")

	logger.Info("User action", "user_id", 123, "user_type", "regular") // Should NOT be logged
	logger.Info("Admin action", "user_id", 456, "user_type", "admin")  // Should be logged

	// Test should complete without errors
}

func TestTimeBasedCondition(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-time").
		WithLogDir("test-logs-time").
		WithLogLevel(slog.LevelDebug).
		WithTimeBasedCondition(9, 17) // Only during work hours (9-17)

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-time")

	currentHour := time.Now().Hour()
	logger.Info("Time-based log", "hour", currentHour)

	// Test should complete without errors
	// Whether it's logged depends on current time
}

func TestRateLimit(t *testing.T) {
	config := DefaultConfig().
		WithAppName("test-rate").
		WithLogDir("test-logs-rate").
		WithLogLevel(slog.LevelDebug).
		WithRateLimit(slog.LevelDebug, 3, time.Minute) // Max 3 DEBUG per minute

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-rate")

	// Send 5 debug messages, only first 3 should be logged
	for i := 0; i < 5; i++ {
		logger.Debug("Debug message", "count", i)
	}

	// Test should complete without errors
}

func TestCombinedFilters(t *testing.T) {
	// Test multiple filters working together
	config := DefaultConfig().
		WithAppName("test-combined").
		WithLogDir("test-logs-combined").
		WithLogLevel(slog.LevelDebug).
		WithFieldMask("password", "***").
		WithRegexFilter(`\d{4}-\d{4}-\d{4}-\d{4}`, "****-****-****-****").
		WithLevelCondition(slog.LevelInfo). // Only INFO and above
		WithRateLimit(slog.LevelInfo, 10, time.Minute)

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	defer os.RemoveAll("test-logs-combined")

	logger.Debug("Debug message")                                                  // Should NOT be logged (below INFO)
	logger.Info("User login", "password", "secret", "card", "1234-5678-9012-3456") // Should be logged with filtering
	logger.Warn("Warning message")                                                 // Should be logged
	logger.Error("Error message")                                                  // Should be logged

	// Test should complete without errors
}

func TestConditionHelpers(t *testing.T) {
	// Test the condition helper functions
	levelCond := LevelCondition(slog.LevelWarn)
	if !levelCond(slog.LevelError, "test", nil) {
		t.Error("Level condition should allow ERROR when minimum is WARN")
	}
	if levelCond(slog.LevelInfo, "test", nil) {
		t.Error("Level condition should not allow INFO when minimum is WARN")
	}

	msgCond := MessageContainsCondition("important")
	if !msgCond(slog.LevelInfo, "This is important", nil) {
		t.Error("Message condition should match when substring is present")
	}
	if msgCond(slog.LevelInfo, "Regular message", nil) {
		t.Error("Message condition should not match when substring is absent")
	}

	attrs := []slog.Attr{
		slog.String("user_type", "admin"),
		slog.Int("user_id", 123),
	}
	attrCond := AttributeCondition("user_type", "admin")
	if !attrCond(slog.LevelInfo, "test", attrs) {
		t.Error("Attribute condition should match when attribute value matches")
	}

	attrCond2 := AttributeCondition("user_type", "regular")
	if attrCond2(slog.LevelInfo, "test", attrs) {
		t.Error("Attribute condition should not match when attribute value differs")
	}
}

func TestCombineConditions(t *testing.T) {
	// Test AND logic
	levelCond := LevelCondition(slog.LevelInfo)
	msgCond := MessageContainsCondition("important")
	combined := CombineConditions(levelCond, msgCond)

	if !combined(slog.LevelInfo, "important message", nil) {
		t.Error("Combined condition should pass when both conditions are met")
	}
	if combined(slog.LevelDebug, "important message", nil) {
		t.Error("Combined condition should fail when first condition fails")
	}
	if combined(slog.LevelInfo, "regular message", nil) {
		t.Error("Combined condition should fail when second condition fails")
	}
}

func TestAnyCondition(t *testing.T) {
	// Test OR logic
	levelCond := LevelCondition(slog.LevelError)
	msgCond := MessageContainsCondition("important")
	any := AnyCondition(levelCond, msgCond)

	if !any(slog.LevelError, "regular message", nil) {
		t.Error("Any condition should pass when first condition is met")
	}
	if !any(slog.LevelInfo, "important message", nil) {
		t.Error("Any condition should pass when second condition is met")
	}
	if any(slog.LevelInfo, "regular message", nil) {
		t.Error("Any condition should fail when no conditions are met")
	}
}

func TestMaskFieldFilter(t *testing.T) {
	filter := MaskFieldFilter("***")
	result := filter("password", slog.StringValue("secret123"))
	if result.String() != "***" {
		t.Errorf("Expected '***', got '%s'", result.String())
	}
}

func TestRedactFieldFilter(t *testing.T) {
	filter := RedactFieldFilter()
	result := filter("sensitive", slog.StringValue("data"))
	if result.String() != "" {
		t.Errorf("Expected empty string, got '%s'", result.String())
	}
}
