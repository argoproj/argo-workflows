package logging

import (
	"io"
	"sync"
)

// TestHook is a hook implementation for testing that captures log messages
type TestHook struct {
	mu      sync.RWMutex
	entries []TestEntry
}

// TestEntry represents a captured log entry
type TestEntry struct {
	Level Level
	Msg   string
}

// NewTestHook creates a new test hook
func NewTestHook() *TestHook {
	return &TestHook{
		entries: make([]TestEntry, 0),
	}
}

// Levels returns the levels this hook should fire on
func (h *TestHook) Levels() []Level {
	return []Level{Trace, Debug, Info, Warn, Error, Fatal, Print, Panic}
}

// Fire is called when a log event is fired
func (h *TestHook) Fire(level Level, msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, TestEntry{
		Level: level,
		Msg:   msg,
	})
}

// LastEntry returns the last captured log entry
func (h *TestHook) LastEntry() *TestEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.entries) == 0 {
		return nil
	}
	return &h.entries[len(h.entries)-1]
}

// AllEntries returns all captured log entries
func (h *TestHook) AllEntries() []TestEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	entries := make([]TestEntry, len(h.entries))
	copy(entries, h.entries)
	return entries
}

// Reset clears all captured entries
func (h *TestHook) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = make([]TestEntry, 0)
}

// NewTestLogger creates a logger that doesn't output to stdout for testing
func NewTestLogger(logLevel Level, format LogType, hooks ...Hook) Logger {
	return NewSlogLoggerCustom(logLevel, format, io.Discard, hooks...)
}
