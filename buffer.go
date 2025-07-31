package iSlogger

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// bufferedWriter provides buffered writing with automatic flushing
type bufferedWriter struct {
	writer        io.Writer
	buffer        *bytes.Buffer
	mu            sync.Mutex
	size          int
	flushInterval time.Duration
	flushOnLevel  slog.Level
	stopChan      chan struct{}
	once          sync.Once
}

// newBufferedWriter creates a new buffered writer
func newBufferedWriter(writer io.Writer, size int, flushInterval time.Duration, flushOnLevel slog.Level) *bufferedWriter {
	if size <= 0 {
		// If buffering is disabled, return a pass-through writer
		return &bufferedWriter{
			writer: writer,
			buffer: &bytes.Buffer{},
			size:   0,
		}
	}

	bw := &bufferedWriter{
		writer:        writer,
		buffer:        bytes.NewBuffer(make([]byte, 0, size)),
		size:          size,
		flushInterval: flushInterval,
		flushOnLevel:  flushOnLevel,
		stopChan:      make(chan struct{}),
	}

	// Start automatic flushing goroutine if interval is set
	if flushInterval > 0 {
		go bw.autoFlush()
	}

	return bw
}

// Write writes data to the buffer and flushes if necessary
func (bw *bufferedWriter) Write(p []byte) (n int, err error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	// If buffering is disabled, write directly
	if bw.size == 0 {
		return bw.writer.Write(p)
	}

	// Check if this is a high-priority log that should be flushed immediately
	shouldFlushImmediately := bw.shouldFlushImmediately(p)

	// Write to buffer
	n, err = bw.buffer.Write(p)
	if err != nil {
		return n, err
	}

	// Flush if buffer is full, or if this is a high-priority log
	if bw.buffer.Len() >= bw.size || shouldFlushImmediately {
		if flushErr := bw.flushLocked(); flushErr != nil {
			return n, flushErr
		}
	}

	return n, nil
}

// shouldFlushImmediately checks if the log entry should trigger immediate flush
func (bw *bufferedWriter) shouldFlushImmediately(p []byte) bool {
	logStr := string(p)

	// Check for high-priority levels based on flushOnLevel
	switch bw.flushOnLevel {
	case slog.LevelDebug:
		return true // Flush on any level
	case slog.LevelInfo:
		return strings.Contains(logStr, "level=INFO") ||
			strings.Contains(logStr, "level=WARN") ||
			strings.Contains(logStr, "level=ERROR") ||
			strings.Contains(logStr, `"level":"INFO"`) ||
			strings.Contains(logStr, `"level":"WARN"`) ||
			strings.Contains(logStr, `"level":"ERROR"`)
	case slog.LevelWarn:
		return strings.Contains(logStr, "level=WARN") ||
			strings.Contains(logStr, "level=ERROR") ||
			strings.Contains(logStr, `"level":"WARN"`) ||
			strings.Contains(logStr, `"level":"ERROR"`)
	case slog.LevelError:
		return strings.Contains(logStr, "level=ERROR") ||
			strings.Contains(logStr, `"level":"ERROR"`)
	}

	return false
}

// Flush flushes the buffer to the underlying writer
func (bw *bufferedWriter) Flush() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.flushLocked()
}

// flushLocked flushes the buffer without acquiring the lock (must be called with lock held)
func (bw *bufferedWriter) flushLocked() error {
	if bw.buffer.Len() == 0 {
		return nil
	}

	_, err := bw.writer.Write(bw.buffer.Bytes())
	if err != nil {
		return err
	}

	bw.buffer.Reset()
	return nil
}

// autoFlush periodically flushes the buffer
func (bw *bufferedWriter) autoFlush() {
	ticker := time.NewTicker(bw.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bw.Flush()
		case <-bw.stopChan:
			return
		}
	}
}

// Close stops the auto-flush goroutine and flushes remaining data
func (bw *bufferedWriter) Close() error {
	bw.once.Do(func() {
		if bw.stopChan != nil {
			close(bw.stopChan)
		}
	})

	// Final flush
	return bw.Flush()
}

// Sync is an alias for Flush to match io interfaces
func (bw *bufferedWriter) Sync() error {
	return bw.Flush()
}
