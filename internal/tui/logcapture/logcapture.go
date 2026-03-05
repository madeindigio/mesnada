package logcapture

import (
	"io"
	"log"
	"strings"
	"sync"
)

const defaultMaxLines = 500

// Buffer captures log output into a ring buffer of lines.
// It implements io.Writer so it can be used as log.SetOutput target.
// Thread-safe for concurrent writes and reads.
type Buffer struct {
	mu       sync.RWMutex
	lines    []string
	maxLines int
	original io.Writer
}

// New creates a Buffer that stores up to maxLines log lines.
// If maxLines <= 0, defaultMaxLines is used.
func New(maxLines int) *Buffer {
	if maxLines <= 0 {
		maxLines = defaultMaxLines
	}
	return &Buffer{
		lines:    make([]string, 0, maxLines),
		maxLines: maxLines,
	}
}

// Install redirects the standard log package output to this buffer.
// The original writer is saved and can be restored with Restore().
func (b *Buffer) Install() {
	b.mu.Lock()
	b.original = log.Writer()
	b.mu.Unlock()
	log.SetOutput(b)
}

// Restore returns the standard log output to its original writer.
func (b *Buffer) Restore() {
	b.mu.RLock()
	orig := b.original
	b.mu.RUnlock()
	if orig != nil {
		log.SetOutput(orig)
	}
}

// Write implements io.Writer. Called by the log package.
func (b *Buffer) Write(p []byte) (n int, err error) {
	text := string(p)
	// Split into individual lines, preserving each log entry
	newLines := strings.Split(strings.TrimRight(text, "\n"), "\n")

	b.mu.Lock()
	for _, line := range newLines {
		if line == "" {
			continue
		}
		b.lines = append(b.lines, line)
	}
	// Trim to maxLines (ring buffer behavior)
	if len(b.lines) > b.maxLines {
		b.lines = b.lines[len(b.lines)-b.maxLines:]
	}
	b.mu.Unlock()

	return len(p), nil
}

// Lines returns a copy of the current log lines.
func (b *Buffer) Lines() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	cp := make([]string, len(b.lines))
	copy(cp, b.lines)
	return cp
}

// Len returns the number of stored lines.
func (b *Buffer) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.lines)
}

// Content returns all log lines joined with newlines.
func (b *Buffer) Content() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return strings.Join(b.lines, "\n")
}
