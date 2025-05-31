package iSlogger

import "time"

type Config struct {
	LogDir        string // Directory for log files
	AppName       string // Application name for log file prefix
	Debug         bool   // Enable debug logging
	RetentionDays int    // Number of days to keep log files
	JSONFormat    bool   // Use JSON format instead of text
	AddSource     bool   // Add source file and line info
	TimeFormat    string // Custom time format
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
