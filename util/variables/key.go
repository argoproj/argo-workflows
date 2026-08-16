package variables

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Key is the handle for a single variable. Obtainable only via Define.
type Key struct {
	template    string
	params      []string
	kind        Kind
	valueType   string
	appliesTo   []TemplateKind
	phases      []LifecyclePhase
	description string
}

// Spec holds a Key's metadata. Passed to Define.
type Spec struct {
	Template    string
	Kind        Kind
	ValueType   string
	AppliesTo   []TemplateKind
	Phases      []LifecyclePhase
	Description string
}

var (
	catalogMu sync.RWMutex
	catalog   []*Key
	byKey     = map[string]*Key{}
)

// Define registers a variable and returns its handle. Panics on duplicate or
// empty Template — these are programmer errors.
func Define(spec Spec) *Key {
	if spec.Template == "" {
		panic("variables: Define with empty Template")
	}
	k := &Key{
		template:    spec.Template,
		params:      placeholders(spec.Template),
		kind:        spec.Kind,
		valueType:   spec.ValueType,
		appliesTo:   spec.AppliesTo,
		phases:      spec.Phases,
		description: spec.Description,
	}
	catalogMu.Lock()
	defer catalogMu.Unlock()
	if _, dup := byKey[k.template]; dup {
		panic(fmt.Sprintf("variables: duplicate Define: %q", k.template))
	}
	byKey[k.template] = k
	catalog = append(catalog, k)
	return k
}

// Template returns the key template (e.g. "steps.<name>.id"). Used by tests.
func (k *Key) Template() string { return k.template }

// Concretize substitutes placeholders in Template with the given args.
func (k *Key) Concretize(args ...string) string {
	if len(args) != len(k.params) {
		panic(fmt.Sprintf("variables: %q expects %d args, got %d", k.template, len(k.params), len(args)))
	}
	out := k.template
	for i, name := range k.params {
		out = strings.Replace(out, "<"+name+">", args[i], 1)
	}
	return out
}

func placeholders(t string) []string {
	var out []string
	for {
		lt := strings.Index(t, "<")
		if lt < 0 {
			return out
		}
		t = t[lt+1:]
		gt := strings.Index(t, ">")
		if gt < 0 {
			return out
		}
		out = append(out, t[:gt])
		t = t[gt+1:]
	}
}

// All returns a snapshot of every registered Key, sorted by Template.
func All() []*Key {
	catalogMu.RLock()
	defer catalogMu.RUnlock()
	out := append([]*Key(nil), catalog...)
	sort.Slice(out, func(i, j int) bool { return out[i].template < out[j].template })
	return out
}
