package iSlogger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// startCleanupRoutine starts the cleanup goroutine
func (l *Logger) startCleanupRoutine() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	l.performCleanup()

	for {
		select {
		case <-ticker.C:
			l.performCleanup()
		}
	}
}

// performCleanup removes old log files
func (l *Logger) performCleanup() {
	cutoffDate := time.Now().AddDate(0, 0, -l.config.RetentionDays)

	entries, err := os.ReadDir(l.config.LogDir)
	if err != nil {
		if l.errorLogger != nil {
			l.Error("Failed to read log directory", "error", err)
		}
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !l.isOurLogFile(entry.Name()) {
			continue
		}

		filePath := filepath.Join(l.config.LogDir, entry.Name())
		if l.shouldRemoveFile(entry, cutoffDate) {
			if err := os.Remove(filePath); err != nil {
				if l.errorLogger != nil {
					l.Error("Failed to remove old log file", "file", entry.Name(), "error", err)
				}
			} else {
				if l.infoLogger != nil {
					l.Info("Removed old log file", "file", entry.Name())
				}
			}
		}
	}
}

// isOurLogFile checks if the file belongs to this logger instance
func (l *Logger) isOurLogFile(filename string) bool {
	if !strings.HasPrefix(filename, l.config.AppName) {
		return false
	}

	if !strings.HasSuffix(filename, ".log") {
		return false
	}

	expectedPatterns := []string{
		l.config.AppName + "_",       // app_2024-01-01.log
		l.config.AppName + "_error_", // app_error_2024-01-01.log
	}

	for _, pattern := range expectedPatterns {
		if strings.HasPrefix(filename, pattern) {
			return true
		}
	}

	return false
}

// shouldRemoveFile determines if a file should be removed based on age
func (l *Logger) shouldRemoveFile(entry os.DirEntry, cutoffDate time.Time) bool {
	info, err := entry.Info()
	if err != nil {
		return false
	}

	return info.ModTime().Before(cutoffDate)
}

// CleanupNow performs immediate cleanup of old log files
func (l *Logger) CleanupNow() {
	go l.performCleanup()
}

// GetLogFiles returns list of current log files
func (l *Logger) GetLogFiles() ([]string, error) {
	entries, err := os.ReadDir(l.config.LogDir)
	if err != nil {
		return nil, err
	}

	var logFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if l.isOurLogFile(entry.Name()) {
			logFiles = append(logFiles, entry.Name())
		}
	}

	return logFiles, nil
}

// GetCurrentLogPaths returns paths to current log files
func (l *Logger) GetCurrentLogPaths() (infoPath, errorPath string) {
	today := time.Now().Format("2006-01-02")
	infoPath = filepath.Join(l.config.LogDir, fmt.Sprintf("%s_%s.log", l.config.AppName, today))
	errorPath = filepath.Join(l.config.LogDir, fmt.Sprintf("%s_error_%s.log", l.config.AppName, today))
	return
}

// RotateNow forces immediate log rotation
func (l *Logger) RotateNow() error {
	return l.initLoggers()
}
