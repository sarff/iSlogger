package pkg

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/sarff/iSlogger"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{input: "debug", want: slog.LevelDebug},
		{input: " INFO ", want: slog.LevelInfo},
		{input: "Warn", want: slog.LevelWarn},
		{input: "error", want: slog.LevelError},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := parseLogLevel(test.input)
			if err != nil {
				t.Fatalf("parseLogLevel(%q) error: %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("parseLogLevel(%q) = %v, want %v", test.input, got, test.want)
			}
		})
	}
	if _, err := parseLogLevel("verbose"); err == nil {
		t.Fatal("parseLogLevel accepted an invalid level")
	}
}

func TestLoggingFallsBackToStandardLogger(t *testing.T) {
	if err := iSlogger.Close(); err != nil {
		t.Fatal(err)
	}
	previous := slog.Default()
	var output bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	Info("fallback-info", "request_id", 42)
	InfoContext(context.Background(), "fallback-context")
	for _, value := range []string{"fallback-info", "request_id=42", "fallback-context"} {
		if !strings.Contains(output.String(), value) {
			t.Errorf("fallback output does not contain %q: %s", value, output.String())
		}
	}
}

func TestGlobalLifecycle(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Cleanup(func() { _ = Close() })

	if err := Init(); err != nil {
		t.Fatalf("first Init() error: %v", err)
	}
	first := iSlogger.GetGlobalLogger()
	if first == nil {
		t.Fatal("Init did not install the global logger")
	}
	if err := Init(); err != nil {
		t.Fatalf("second Init() error: %v", err)
	}
	if second := iSlogger.GetGlobalLogger(); second == nil || second == first {
		t.Fatal("second Init did not replace the global logger")
	}
	if err := Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	if iSlogger.GetGlobalLogger() != nil {
		t.Fatal("Close did not clear the global logger")
	}
}
