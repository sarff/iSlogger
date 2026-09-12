package iSlogger

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (c *loggerCore) cleanupLoop() {
	defer c.cleanupWG.Done()
	_ = c.cleanup()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

func (c *loggerCore) cleanup() error {
	c.cleanupMu.Lock()
	defer c.cleanupMu.Unlock()

	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil
	}
	config := c.config
	baseDir := c.baseDir
	activeDate := c.currentDate
	now := c.now()
	c.mu.RUnlock()

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("read log directory: %w", err)
	}
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -config.RetentionDays)
	var errs []error
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		date, ok := logFileDate(config.AppName, entry.Name())
		if !ok || date == activeDate {
			continue
		}
		parsed, err := time.ParseInLocation(time.DateOnly, date, now.Location())
		if err != nil || !parsed.Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(baseDir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("remove %s: %w", entry.Name(), err))
		}
	}
	return errors.Join(errs...)
}

func logFileDate(appName, filename string) (string, bool) {
	if !strings.HasSuffix(filename, ".log") {
		return "", false
	}
	stem := strings.TrimSuffix(filename, ".log")
	prefixes := []string{appName + "_error_", appName + "_"}
	for _, prefix := range prefixes {
		if !strings.HasPrefix(stem, prefix) {
			continue
		}
		date := strings.TrimPrefix(stem, prefix)
		if _, err := time.Parse(time.DateOnly, date); err == nil {
			return date, true
		}
	}
	return "", false
}

func (l *Logger) isOurLogFile(filename string) bool {
	_, ok := logFileDate(l.config.AppName, filename)
	return ok
}

// Cleanup removes expired log files synchronously.
func (l *Logger) Cleanup() error {
	if l == nil || l.core == nil {
		return nil
	}
	return l.core.cleanup()
}

// CleanupNow is retained for compatibility.
// Deprecated: use Cleanup to receive any filesystem error.
func (l *Logger) CleanupNow() { _ = l.Cleanup() }

// GetLogFiles returns strictly named log files for this application.
func (l *Logger) GetLogFiles() ([]string, error) {
	if l == nil || l.core == nil {
		return nil, nil
	}
	entries, err := os.ReadDir(l.core.baseDir)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && l.isOurLogFile(entry.Name()) {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

// GetCurrentLogPaths returns the configured paths for the open daily files.
func (l *Logger) GetCurrentLogPaths() (infoPath, errorPath string) {
	if l == nil || l.core == nil {
		return "", ""
	}
	l.core.mu.RLock()
	date := l.core.currentDate
	l.core.mu.RUnlock()
	return filepath.Join(l.config.LogDir, fmt.Sprintf("%s_%s.log", l.config.AppName, date)),
		filepath.Join(l.config.LogDir, fmt.Sprintf("%s_error_%s.log", l.config.AppName, date))
}
