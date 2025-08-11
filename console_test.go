package iSlogger

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestConsoleOutput_Enabled(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	config := DefaultConfig().
		WithAppName("console-test").
		WithLogDir("test-logs").
		WithConsoleOutput(true).
		WithDebug(true)

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log a message
	testMessage := "Console output test message"
	logger.Info(testMessage)

	// Close the pipe writer and read output
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify message appears in console output
	if !strings.Contains(output, testMessage) {
		t.Errorf("Expected console output to contain %q, but got: %s", testMessage, output)
	}
}

func TestConsoleOutput_Disabled(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	config := DefaultConfig().
		WithAppName("console-test-disabled").
		WithLogDir("test-logs").
		WithConsoleOutput(false).
		WithDebug(true)

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log a message
	testMessage := "Console disabled test message"
	logger.Info(testMessage)

	// Close the pipe writer and read output
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify message does NOT appear in console output
	if strings.Contains(output, testMessage) {
		t.Errorf("Expected console output to NOT contain %q when console output is disabled, but got: %s", testMessage, output)
	}
}

func TestConsoleOutput_DefaultBehavior(t *testing.T) {
	config := DefaultConfig()

	// Verify console output is enabled by default
	if !config.ConsoleOutput {
		t.Errorf("Expected console output to be enabled by default, but it was disabled")
	}
}

func TestWithConsoleOutput(t *testing.T) {
	config := DefaultConfig()

	// Test enabling console output
	configEnabled := config.WithConsoleOutput(true)
	if !configEnabled.ConsoleOutput {
		t.Errorf("Expected console output to be enabled after WithConsoleOutput(true)")
	}

	// Test disabling console output
	configDisabled := config.WithConsoleOutput(false)
	if configDisabled.ConsoleOutput {
		t.Errorf("Expected console output to be disabled after WithConsoleOutput(false)")
	}

	// Verify original config is unchanged (immutable pattern)
	if !config.ConsoleOutput {
		t.Errorf("Expected original config to remain unchanged")
	}
}
