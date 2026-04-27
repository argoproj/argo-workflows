package variables

import (
	"maps"
	"sort"
)

// Scope holds variable values. The internal map is unexported, and the only
// method that writes into it is Key.Set — so the only way to populate a
// Scope is to go through a registered Key, which enforces the catalog
// contract at the type-system level.
//
// Reading out of a Scope is flexible: AsAnyMap / AsStringMap produce
// snapshots for consumption by template.Replace and similar legacy
// map-based APIs. Mutating those snapshots does not write back.
type Scope struct {
	data map[string]any
}

// NewScope returns an empty Scope.
func NewScope() *Scope { return &Scope{data: map[string]any{}} }

// set is package-private; the only caller is Key.Set. This is the single
// write path for a Scope.
func (s *Scope) set(key string, value any) {
	s.data[key] = value
}

// get is package-private; the only caller is Key.Get.
// Nil-safe: a zero-value/nil Scope returns (nil, false).
func (s *Scope) get(key string) (any, bool) {
	if s == nil {
		return nil, false
	}
	v, ok := s.data[key]
	return v, ok
}

// AsAnyMap returns a snapshot of the scope as map[string]any, suitable for
// passing to expr.Compile / expr.Run. Mutations to the returned map do not
// affect the Scope. Nil-safe.
func (s *Scope) AsAnyMap() map[string]any {
	if s == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(s.data))
	maps.Copy(out, s.data)
	return out
}

// AsStringMap returns a snapshot of the string-valued entries. Non-string
// values are skipped. Intended for bridging into common.Parameters-style
// APIs that expect map[string]string. Nil-safe.
func (s *Scope) AsStringMap() map[string]string {
	out := map[string]string{}
	if s == nil {
		return out
	}
	for k, v := range s.data {
		if str, ok := v.(string); ok {
			out[k] = str
		}
	}
	return out
}

// Len returns the number of entries currently in the scope. Nil-safe.
func (s *Scope) Len() int {
	if s == nil {
		return 0
	}
	return len(s.data)
}

// Keys returns every concrete key currently in the scope, sorted.
// Useful for tests and debug dumps. Nil-safe.
func (s *Scope) Keys() []string {
	if s == nil {
		return nil
	}
	out := make([]string, 0, len(s.data))
	for k := range s.data {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Set writes a value through a Key. This is the ONLY way for caller code
// to write into a Scope — because Scope.set is unexported, Key.Set is the
// single public write path, and a Key cannot be constructed outside this
// package.
func (k *Key) Set(s *Scope, value any, args ...string) {
	s.set(k.Concretize(args...), value)
}

// Get reads a value through a Key.
func (k *Key) Get(s *Scope, args ...string) (any, bool) {
	return s.get(k.Concretize(args...))
}

// Has reports whether a value for this Key (with given args) is currently
// set in the scope.
func (k *Key) Has(s *Scope, args ...string) bool {
	_, ok := s.get(k.Concretize(args...))
	return ok
}
