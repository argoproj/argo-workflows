package variables

import (
	"github.com/argoproj/argo-workflows/v4/util/exprtrace"
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
	data *exprtrace.Map
}

// NewScope returns an empty Scope.
func NewScope() *Scope { return &Scope{data: exprtrace.New()} }

// set is package-private; the only caller is Key.Set. This is the single
// write path for a Scope.
func (s *Scope) set(key string, value any, skip int) {
	s.data.SetFromCaller(key, value, skip+1)
}

// get is package-private; the only caller is Key.Get.
func (s *Scope) get(key string) (any, bool) {
	return s.data.Get(key)
}

// AsAnyMap returns a snapshot of the scope as map[string]any, suitable for
// passing to expr.Compile / expr.Run. Mutations to the returned map do not
// affect the Scope.
func (s *Scope) AsAnyMap() map[string]any { return s.data.AsAnyMap() }

// AsStringMap returns a snapshot of the string-valued entries. Non-string
// values are skipped. Intended for bridging into common.Parameters-style
// APIs that expect map[string]string.
func (s *Scope) AsStringMap() map[string]string {
	out := map[string]string{}
	for k, v := range s.data.AsAnyMap() {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

// Len returns the number of entries currently in the scope.
func (s *Scope) Len() int { return s.data.Len() }

// Keys returns every concrete key currently in the scope, sorted.
// Useful for tests and debug dumps.
func (s *Scope) Keys() []string { return s.data.Keys() }

// Provenance returns a snapshot of entry provenance (file, line, function).
// Useful for exprtrace-style d2 dumps.
func (s *Scope) Provenance() map[string]exprtrace.Entry { return s.data.Entries() }

// Set writes a value through a Key. This is the ONLY way for caller code
// to write into a Scope — because Scope.set is unexported, Key.Set is the
// single public write path, and a Key cannot be constructed outside this
// package.
func (k *Key) Set(s *Scope, value any, args ...string) {
	s.set(k.Concretize(args...), value, 1)
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
