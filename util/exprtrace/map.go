// Package exprtrace provides a string-keyed map that records where each entry
// was written (file, line, function, optional stack), and a d2 dumper that
// visualises the map contents alongside the call site of an expr.Compile.
//
// The Map is intended as a drop-in replacement for map[string]any in code paths
// that feed expr.Compile / expr.Run, so we can see which piece of the
// controller contributed each variable at evaluation time.
package exprtrace

import (
	"maps"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"sync"
)

// Entry is a single key's value plus the provenance of its last Set.
type Entry struct {
	Value    any
	File     string
	Line     int
	Function string
	// Stack is only populated when ARGO_EXPR_TRACE_STACK=1.
	Stack string
}

// Map is a string-keyed value store that records where each entry was last set.
// The zero value is not usable — construct with New().
type Map struct {
	mu           sync.RWMutex
	data         map[string]Entry
	captureStack bool
}

// New returns an empty Map. If ARGO_EXPR_TRACE_STACK=1 is set in the
// environment, every Set will also capture a full goroutine stack.
func New() *Map {
	return &Map{
		data:         make(map[string]Entry),
		captureStack: os.Getenv("ARGO_EXPR_TRACE_STACK") == "1",
	}
}

// Enabled returns true if tracing output is configured (ARGO_EXPR_TRACE_DIR
// is set). Call-sites should guard FromAnyMap/DumpD2 with this so we do not
// allocate a traced Map on the hot path when tracing is disabled.
func Enabled() bool {
	return os.Getenv("ARGO_EXPR_TRACE_DIR") != ""
}

// FromAnyMap lifts a plain map[string]any into a Map. Every entry is
// attributed to the caller of FromAnyMap — use this at the last mile
// (e.g. just before expr.Compile) when you do not have a tracked Map
// through the whole call chain. The resulting Map still visualises every
// key's presence; only the provenance is coarse.
func FromAnyMap(m map[string]any) *Map {
	out := New()
	pc, file, line, ok := runtime.Caller(1)
	var fname string
	if ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			fname = fn.Name()
		}
	}
	for k, v := range m {
		out.data[k] = Entry{
			Value:    v,
			File:     file,
			Line:     line,
			Function: fname,
		}
	}
	return out
}

// Set records key=value with caller info captured via runtime.Caller(1).
// Use SetFromCaller from helper functions so the attribution skips past the
// helper and names the real caller.
//
// Panics if m is nil; Set is a write so callers must have constructed a Map.
func (m *Map) Set(key string, value any) {
	m.setAt(key, value, 2)
}

// SetFromCaller is like Set but attributes the entry to the caller skip
// frames up the stack. skip=0 behaves like Set.
//
// Panics if m is nil.
func (m *Map) SetFromCaller(key string, value any, skip int) {
	m.setAt(key, value, skip+2)
}

func (m *Map) setAt(key string, value any, skip int) {
	pc, file, line, ok := runtime.Caller(skip)
	var fname string
	if ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			fname = fn.Name()
		}
	}
	var stack string
	if m.captureStack {
		stack = string(debug.Stack())
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = Entry{
		Value:    value,
		File:     file,
		Line:     line,
		Function: fname,
		Stack:    stack,
	}
}

// Get returns the value (without provenance). Safe on nil receiver.
func (m *Map) Get(key string) (any, bool) {
	if m == nil {
		return nil, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.data[key]
	if !ok {
		return nil, false
	}
	return e.Value, true
}

// Entry returns the full Entry (with provenance) for key. Safe on nil receiver.
func (m *Map) Entry(key string) (Entry, bool) {
	if m == nil {
		return Entry{}, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.data[key]
	return e, ok
}

// Delete removes a key. Panics if m is nil (would be a bug — can't delete from nothing).
func (m *Map) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

// Len returns the number of entries. Safe on nil receiver.
func (m *Map) Len() int {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

// Keys returns a sorted slice of all keys. Safe on nil receiver.
func (m *Map) Keys() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Entries returns a shallow copy of the provenance-bearing entries.
// Safe on nil receiver (returns empty map, so it's safe to range).
func (m *Map) Entries() map[string]Entry {
	if m == nil {
		return map[string]Entry{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]Entry, len(m.data))
	maps.Copy(out, m.data)
	return out
}

// AsAnyMap returns a copy of the values as map[string]any, suitable for
// passing to expr.Compile / expr.Run. Safe on nil receiver.
func (m *Map) AsAnyMap() map[string]any {
	if m == nil {
		return map[string]any{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]any, len(m.data))
	for k, v := range m.data {
		out[k] = v.Value
	}
	return out
}

// Merge copies entries from other into m, preserving other's provenance.
// Entries already present in m are overwritten.
func (m *Map) Merge(other *Map) {
	if other == nil {
		return
	}
	other.mu.RLock()
	src := make(map[string]Entry, len(other.data))
	maps.Copy(src, other.data)
	other.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	maps.Copy(m.data, src)
}

// Clone returns a deep copy of m. Safe on nil receiver (returns a fresh empty Map).
func (m *Map) Clone() *Map {
	out := New()
	if m == nil {
		return out
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	maps.Copy(out.data, m.data)
	return out
}
