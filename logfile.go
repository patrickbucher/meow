package meow

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LogFile writes log messags to a file.
type LogFile struct {
	Sink *bufio.Writer
	file *os.File
}

// NewLogFile creates a log file under the given path.
func NewLogFile(path string) (*LogFile, error) {
	logFile, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("open log file %s: %w", path, err)
	}
	return &LogFile{bufio.NewWriter(logFile), logFile}, nil
}

// Write writes the given data to the underlying sink and flushes it.
func (l LogFile) Write(data []byte) (int, error) {
	n, err := l.Sink.Write(data)
	if err != nil {
		return 0, fmt.Errorf("write to sink: %w", err)
	}
	if err := l.Sink.Flush(); err != nil {
		return 0, fmt.Errorf("flush sink: %w", err)
	}
	return n, nil
}

// WriteString writes the given string using the sink's Write() function.
func (l LogFile) WriteLine(s string) (int, error) {
	return l.Write([]byte(strings.TrimSpace(s) + "\n"))
}

// Close closes the underlying file.
func (l LogFile) Close() error {
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", l.file.Name(), err)
	}
	return nil
}
