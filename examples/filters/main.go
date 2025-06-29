package main

import (
	"log/slog"
	"time"

	"github.com/sarff/iSlogger"
)

func main() {
	// Example 1: Basic field masking and redaction
	basicFilteringExample()

	// Example 2: Conditional logging
	conditionalLoggingExample()

	// Example 3: Rate limiting
	rateLimitingExample()

	// Example 4: Complex filtering for production
	productionFilteringExample()
}

func basicFilteringExample() {
	println("\n=== Basic Filtering Example ===")

	config := iSlogger.DefaultConfig().
		WithAppName("filter-basic").
		WithLogDir("logs").
		WithDebug(true).
		// Mask sensitive fields
		WithFieldMask("password", "***").
		WithFieldMask("api_key", "***HIDDEN***").
		WithFieldRedaction("internal_data").
		// Filter credit card numbers
		WithRegexFilter(`\d{4}-\d{4}-\d{4}-\d{4}`, "****-****-****-****")

	logger, err := iSlogger.New(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// These fields will be filtered
	logger.Info("User registration",
		"username", "john_doe",
		"password", "secretpassword123",
		"api_key", "sk_live_1234567890abcdef",
		"internal_data", "should_not_appear",
		"payment_info", "Card: 1234-5678-9012-3456",
	)

	// Regular logging works normally
	logger.Info("User logged in", "user_id", 12345, "timestamp", time.Now())
}

func conditionalLoggingExample() {
	println("\n=== Conditional Logging Example ===")

	config := iSlogger.DefaultConfig().
		WithAppName("filter-condition").
		WithLogDir("logs").
		WithDebug(true).
		// Only log warnings/errors OR messages containing "important"
		WithCondition(iSlogger.AnyCondition(
			iSlogger.LevelCondition(slog.LevelWarn),
			iSlogger.MessageContainsCondition("important"),
		)).
		// Additional condition: only during work hours (9-18)
		WithTimeBasedCondition(9, 18)

	logger, err := iSlogger.New(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// These will be filtered based on conditions
	logger.Debug("Regular debug message")        // Not logged (below WARN and doesn't contain "important")
	logger.Info("Regular info message")          // Not logged (below WARN and doesn't contain "important")
	logger.Info("Important system notification") // Logged (contains "important")
	logger.Warn("Memory usage is high")          // Logged (WARN level)
	logger.Error("Database connection failed")   // Logged (ERROR level)

	// Time-based filtering will depend on current hour
	currentHour := time.Now().Hour()
	if currentHour >= 9 && currentHour <= 18 {
		println("Work hours - logs will be written")
	} else {
		println("Outside work hours - logs may be filtered")
	}
}

func rateLimitingExample() {
	println("\n=== Rate Limiting Example ===")

	config := iSlogger.DefaultConfig().
		WithAppName("filter-rate").
		WithLogDir("logs").
		WithDebug(true).
		// Limit DEBUG messages to 5 per minute
		WithRateLimit(slog.LevelDebug, 5, time.Minute).
		// Limit INFO messages to 20 per minute
		WithRateLimit(slog.LevelInfo, 20, time.Minute)

	logger, err := iSlogger.New(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// Rapid debug logging - only first 5 will be logged
	println("Sending 10 debug messages rapidly...")
	for i := 0; i < 10; i++ {
		logger.Debug("Debug message", "count", i, "timestamp", time.Now())
	}

	// WARN and ERROR are not rate limited
	logger.Warn("This warning will always be logged")
	logger.Error("This error will always be logged")
}

func productionFilteringExample() {
	println("\n=== Production Filtering Example ===")

	config := iSlogger.DefaultConfig().
		WithAppName("filter-production").
		WithLogDir("logs").
		WithDebug(false). // Production mode
		// Security: mask all sensitive fields
		WithFieldMask("password", "***").
		WithFieldMask("api_key", "***").
		WithFieldMask("secret", "***").
		WithFieldMask("token", "***").
		WithFieldRedaction("internal").
		// Filter common sensitive patterns
		WithRegexFilter(`(?i)password["\s]*[:=]["\s]*[^";\s]+`, "password: ***").
		WithRegexFilter(`(?i)api[_-]?key["\s]*[:=]["\s]*[^";\s]+`, "api_key: ***").
		WithRegexFilter(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, "***@***.***").
		WithRegexFilter(`\d{4}-\d{4}-\d{4}-\d{4}`, "****-****-****-****").
		// Production conditions: only important logs
		WithCondition(iSlogger.AnyCondition(
			iSlogger.LevelCondition(slog.LevelWarn),           // All warnings and errors
			iSlogger.AttributeCondition("service", "payment"), // All payment service logs
			iSlogger.AttributeCondition("critical", "true"),   // Critical operations
			iSlogger.CombineConditions( // User actions for admins only
				iSlogger.AttributeCondition("user_type", "admin"),
				iSlogger.MessageContainsCondition("action"),
			),
		)).
		// Rate limiting for production
		WithRateLimit(slog.LevelInfo, 100, time.Minute).
		WithRateLimit(slog.LevelDebug, 10, time.Minute)

	logger, err := iSlogger.New(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// These demonstrate production filtering
	logger.Debug("Debug info", "sql", "SELECT * FROM users") // Not logged (debug disabled)

	logger.Info("Regular operation") // Not logged (doesn't meet conditions)

	logger.Info("Payment processed", // Logged (service=payment)
		"service", "payment",
		"amount", 100.50,
		"card_number", "1234-5678-9012-3456", // Will be masked
		"user_email", "user@example.com", // Will be masked
	)

	logger.Info("Admin action", // Logged (admin + action)
		"user_type", "admin",
		"action", "delete_user",
		"target_user", 12345,
		"api_key", "sk_live_dangerous_key", // Will be masked
	)

	logger.Info("Critical system operation", // Logged (critical=true)
		"critical", "true",
		"operation", "database_backup",
		"internal", "sensitive_data", // Will be redacted
	)

	logger.Warn("High memory usage", // Logged (WARN level)
		"memory_percent", 85,
		"service", "web-server",
	)

	logger.Error("Service unavailable", // Logged (ERROR level)
		"service", "database",
		"error", "connection timeout",
		"password", "should_be_hidden", // Will be masked
	)
}
