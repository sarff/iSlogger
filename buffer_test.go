package iSlogger

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBufferedWriter_Write(t *testing.T) {
	buf := &bytes.Buffer{}
	bw := newBufferedWriter(buf, 100, 0)
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
	bw := newBufferedWriter(buf, 10, 0) // Small buffer
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

func TestBufferedWriter_DoesNotInspectLogText(t *testing.T) {
	buf := &bytes.Buffer{}
	bw := newBufferedWriter(buf, 1000, 0)
	defer bw.Close()

	warnData := []byte(`{"level":"WARN","msg":"warning message"}`)
	bw.Write(warnData)
	if buf.Len() > 0 {
		t.Fatal("buffer must not infer flush policy from formatted bytes")
	}
}

func TestBufferedWriter_ManualFlush(t *testing.T) {
	buf := &bytes.Buffer{}
	bw := newBufferedWriter(buf, 1000, 0)
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
	buf := &notifyingBuffer{wrote: make(chan struct{})}
	bw := newBufferedWriter(buf, 1000, 50*time.Millisecond)

	data := []byte("test message")
	bw.Write(data)

	select {
	case <-buf.wrote:
	case <-time.After(time.Second):
		t.Fatal("automatic flush did not run")
	}

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

type notifyingBuffer struct {
	bytes.Buffer
	wrote chan struct{}
	once  sync.Once
}

func (b *notifyingBuffer) Write(p []byte) (int, error) {
	n, err := b.Buffer.Write(p)
	b.once.Do(func() { close(b.wrote) })
	return n, err
}

func TestBufferedWriter_NoBuffering(t *testing.T) {
	buf := &bytes.Buffer{}
	bw := newBufferedWriter(buf, 0, 0) // No buffering
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
	bw := newBufferedWriter(buf, 1000, 0)

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
