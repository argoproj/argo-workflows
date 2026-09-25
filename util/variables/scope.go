package variables

import "maps"

// Scope holds variable values. Writes only via Key.Set.
type Scope struct {
	data    map[string]any
	skipped map[string]bool
}

func NewScope() *Scope {
	return &Scope{
		data:    map[string]any{},
		skipped: map[string]bool{},
	}
}

// AsAnyMap returns a snapshot. Nil-safe.
func (s *Scope) AsAnyMap() map[string]any {
	if s == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(s.data))
	maps.Copy(out, s.data)
	return out
}

// AsStringMap returns a snapshot of string-valued entries. Nil-safe.
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

func (s *Scope) IsSkipped(key string) bool {
	return s.skipped[key]
}

// Set writes through a Key
func (k *Key) Set(s *Scope, value any, args ...string) {
	s.data[k.Concretize(args...)] = value
}

// SetSkipped writes through a Key, marking the variable as skipped
func (k *Key) SetSkipped(s *Scope, value any, args ...string) {
	concrete := k.Concretize(args...)
	s.data[concrete] = value
	s.skipped[concrete] = true
}
