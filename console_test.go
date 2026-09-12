package iSlogger

import (
	"bytes"
	"log/slog"
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
		WithLogDir(t.TempDir()).
		WithConsoleOutput(true).
		WithLogLevel(slog.LevelDebug)

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
		WithLogDir(t.TempDir()).
		WithConsoleOutput(false).
		WithLogLevel(slog.LevelDebug)

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

func TestConsoleOutput_RoutesEachRecordOnce(t *testing.T) {
	oldStdout, oldStderr := os.Stdout, os.Stderr
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = stdoutW, stderrW
	defer func() { os.Stdout, os.Stderr = oldStdout, oldStderr }()

	logger, err := New(DefaultConfig().
		WithLogDir(t.TempDir()).
		WithAppName("console-routing").
		WithConsoleOutput(true).
		WithoutBuffering())
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("stdout-record")
	logger.Warn("stderr-record")
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}
	_ = stdoutW.Close()
	_ = stderrW.Close()

	var stdout, stderr bytes.Buffer
	_, _ = stdout.ReadFrom(stdoutR)
	_, _ = stderr.ReadFrom(stderrR)
	if strings.Count(stdout.String(), "stdout-record") != 1 || strings.Contains(stdout.String(), "stderr-record") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
	if strings.Count(stderr.String(), "stderr-record") != 1 || strings.Contains(stderr.String(), "stdout-record") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
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
