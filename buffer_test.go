package iSlogger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestBufferedWriter_Write(t *testing.T) {
	buf := &bytes.Buffer{}
	bw := newBufferedWriter(buf, 100, 0, slog.LevelError)
	defer bw.Close()

	data := []byte("test message")
	n, err := bw.Write(data)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if n != len(data) {
		t.Fatalf("Expected %d bytes written, got %d", len(data), n)
	}

	// Data should be in buffer, not yet written to underlying writer
	if buf.Len() > 0 {
		t.Fatal("Data should not be written to underlying writer yet")
	}
}

func TestBufferedWriter_FlushOnSize(t *testing.T) {
	buf := &bytes.Buffer{}
	bw := newBufferedWriter(buf, 10, 0, slog.LevelError) // Small buffer
	defer bw.Close()

	data := []byte("this is a long message that exceeds buffer size")
	n, err := bw.Write(data)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if n != len(data) {
		t.Fatalf("Expected %d bytes written, got %d", len(data), n)
	}

	// Data should be flushed to underlying writer due to size
	if buf.Len() == 0 {
		t.Fatal("Data should be flushed to underlying writer")
	}
}

func TestBufferedWriter_FlushOnLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	bw := newBufferedWriter(buf, 1000, 0, slog.LevelWarn) // Large buffer, flush on WARN
	defer bw.Close()

	// Write INFO level - should not flush immediately
	infoData := []byte(`{"level":"INFO","msg":"info message"}`)
	bw.Write(infoData)
	if buf.Len() > 0 {
		t.Fatal("INFO message should not flush immediately")
	}

	// Write WARN level - should flush immediately
	warnData := []byte(`{"level":"WARN","msg":"warning message"}`)
	bw.Write(warnData)
	if buf.Len() == 0 {
		t.Fatal("WARN message should trigger immediate flush")
	}
}

func TestBufferedWriter_ManualFlush(t *testing.T) {
	buf := &bytes.Buffer{}
	bw := newBufferedWriter(buf, 1000, 0, slog.LevelError)
	defer bw.Close()

	data := []byte("test message")
	bw.Write(data)

	// Should not be flushed yet
	if buf.Len() > 0 {
		t.Fatal("Data should not be flushed yet")
	}

	// Manual flush
	err := bw.Flush()
	if err != nil {
		t.Fatalf("Expected no error on flush, got: %v", err)
	}

	// Should be flushed now
	if buf.Len() == 0 {
		t.Fatal("Data should be flushed after manual flush")
	}
	if !strings.Contains(buf.String(), "test message") {
		t.Fatal("Flushed data should contain original message")
	}
}

func TestBufferedWriter_AutoFlush(t *testing.T) {
	buf := &bytes.Buffer{}
	bw := newBufferedWriter(buf, 1000, 50*time.Millisecond, slog.LevelError)

	data := []byte("test message")
	bw.Write(data)

	// Wait for auto flush - longer wait to ensure flush happens
	time.Sleep(100 * time.Millisecond)

	// Close the writer to stop goroutine and flush remaining data
	err := bw.Close()
	if err != nil {
		t.Fatalf("Expected no error on close, got: %v", err)
	}

	// Now safely check the content
	bufContent := buf.String()
	if !strings.Contains(bufContent, "test message") {
		t.Fatal("Auto-flushed data should contain original message")
	}
}

func TestBufferedWriter_NoBuffering(t *testing.T) {
	buf := &bytes.Buffer{}
	bw := newBufferedWriter(buf, 0, 0, slog.LevelError) // No buffering
	defer bw.Close()

	data := []byte("test message")
	n, err := bw.Write(data)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if n != len(data) {
		t.Fatalf("Expected %d bytes written, got %d", len(data), n)
	}

	// Data should be written immediately when buffering is disabled
	if buf.Len() == 0 {
		t.Fatal("Data should be written immediately when buffering is disabled")
	}
	if !strings.Contains(buf.String(), "test message") {
		t.Fatal("Written data should contain original message")
	}
}

func TestBufferedWriter_Close(t *testing.T) {
	buf := &bytes.Buffer{}
	bw := newBufferedWriter(buf, 1000, 0, slog.LevelError)

	data := []byte("test message")
	bw.Write(data)

	// Should not be flushed yet
	if buf.Len() > 0 {
		t.Fatal("Data should not be flushed yet")
	}

	// Close should flush remaining data
	err := bw.Close()
	if err != nil {
		t.Fatalf("Expected no error on close, got: %v", err)
	}

	// Should be flushed now
	if buf.Len() == 0 {
		t.Fatal("Data should be flushed on close")
	}
	if !strings.Contains(buf.String(), "test message") {
		t.Fatal("Flushed data should contain original message")
	}
}
