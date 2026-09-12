package iSlogger

import (
	"bytes"
	"io"
	"sync"
	"time"
)

// bufferedWriter provides buffered writing with automatic flushing.
type bufferedWriter struct {
	writer        io.Writer
	buffer        *bytes.Buffer
	mu            sync.Mutex
	size          int
	flushInterval time.Duration
	stopChan      chan struct{}
	stopOnce      sync.Once
	wg            sync.WaitGroup
	closed        bool
}

func newBufferedWriter(writer io.Writer, size int, flushInterval time.Duration) *bufferedWriter {
	bw := &bufferedWriter{
		writer:        writer,
		buffer:        bytes.NewBuffer(make([]byte, 0, max(size, 0))),
		size:          max(size, 0),
		flushInterval: flushInterval,
	}
	if bw.size > 0 && flushInterval > 0 {
		bw.stopChan = make(chan struct{})
		bw.wg.Add(1)
		go bw.autoFlush()
	}
	return bw
}

func (bw *bufferedWriter) Write(p []byte) (int, error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	if bw.closed {
		return 0, io.ErrClosedPipe
	}
	if bw.size == 0 {
		return bw.writer.Write(p)
	}

	n, err := bw.buffer.Write(p)
	if err != nil {
		return n, err
	}
	if bw.buffer.Len() >= bw.size {
		if err := bw.flushLocked(); err != nil {
			return n, err
		}
	}
	return n, nil
}

func (bw *bufferedWriter) Flush() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.flushLocked()
}

func (bw *bufferedWriter) flushLocked() error {
	if bw.buffer.Len() == 0 {
		return nil
	}
	data := bw.buffer.Bytes()
	n, err := bw.writer.Write(data)
	if n > 0 {
		bw.buffer.Next(n)
	}
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

func (bw *bufferedWriter) autoFlush() {
	defer bw.wg.Done()
	ticker := time.NewTicker(bw.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = bw.Flush()
		case <-bw.stopChan:
			return
		}
	}
}

func (bw *bufferedWriter) Close() error {
	bw.stopOnce.Do(func() {
		if bw.stopChan != nil {
			close(bw.stopChan)
		}
	})
	bw.wg.Wait()

	bw.mu.Lock()
	defer bw.mu.Unlock()
	if bw.closed {
		return nil
	}
	err := bw.flushLocked()
	bw.closed = true
	return err
}
