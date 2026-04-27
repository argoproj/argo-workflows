package variables

import (
	"fmt"
	"strings"
	"sync"
)

// Key is the handle for a single variable. A Key is obtainable only via
// Define, which guarantees the variable is registered in the catalog.
// Outside this package there is no way to synthesise a Key, so there is
// no way to write into a Scope without going through the catalog.
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
	// Template is the key as users see it, with placeholders in angle
	// brackets. Examples:
	//   "workflow.name"                       (no placeholders)
	//   "steps.<name>.id"                     (one)
	//   "steps.<name>.outputs.parameters.<p>" (two, order matters)
	Template string
	// Kind categorises the variable.
	Kind Kind
	// ValueType is a rough Go type description: "string", "json", "wfv1.Artifact".
	ValueType string
	// AppliesTo lists the TemplateKinds in whose scope this variable appears.
	AppliesTo []TemplateKind
	// Phases lists the lifecycle moments at which this variable is in scope.
	Phases []LifecyclePhase
	// Description is a one-line user-facing explanation.
	Description string
}

var (
	catalogMu sync.RWMutex
	catalog   []*Key
	byKey     = map[string]*Key{}
)

// Define registers a variable and returns its handle. Must be called at
// package init time (typically from a package-level var declaration).
// Panics on duplicate Template or empty Template — these are programmer
// errors, not runtime conditions.
func Define(spec Spec) *Key {
	if spec.Template == "" {
		panic("variables: Define with empty Template")
	}
	k := &Key{
		template:    spec.Template,
		params:      parsePlaceholders(spec.Template),
		kind:        spec.Kind,
		valueType:   spec.ValueType,
		appliesTo:   append([]TemplateKind(nil), spec.AppliesTo...),
		phases:      append([]LifecyclePhase(nil), spec.Phases...),
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

// Template returns the key template (e.g. "steps.<name>.id").
func (k *Key) Template() string { return k.template }

// Kind returns the variable's Kind.
func (k *Key) Kind() Kind { return k.kind }

// ValueType returns the rough type of this variable's values.
func (k *Key) ValueType() string { return k.valueType }

// AppliesTo lists template kinds in whose scope this variable appears.
func (k *Key) AppliesTo() []TemplateKind { return append([]TemplateKind(nil), k.appliesTo...) }

// Phases lists lifecycle moments at which this variable is in scope.
func (k *Key) Phases() []LifecyclePhase { return append([]LifecyclePhase(nil), k.phases...) }

// Description is a one-line user-facing explanation.
func (k *Key) Description() string { return k.description }

// Placeholders returns the placeholder names in Template, in order of
// appearance. For "steps.<name>.outputs.parameters.<p>" returns ["name","p"].
func (k *Key) Placeholders() []string { return append([]string(nil), k.params...) }

// Concretize substitutes placeholders in Template with the given args.
// Panics if the number of args does not match the number of placeholders.
func (k *Key) Concretize(args ...string) string {
	if len(args) != len(k.params) {
		panic(fmt.Sprintf(
			"variables: %q expects %d placeholder arg(s), got %d",
			k.template, len(k.params), len(args),
		))
	}
	out := k.template
	for i, name := range k.params {
		out = strings.Replace(out, "<"+name+">", args[i], 1)
	}
	return out
}

// parsePlaceholders extracts placeholder names from a Template like
// "steps.<name>.outputs.parameters.<p>" → ["name", "p"].
func parsePlaceholders(template string) []string {
	var out []string
	rest := template
	for {
		lt := strings.Index(rest, "<")
		if lt < 0 {
			return out
		}
		rest = rest[lt+1:]
		gt := strings.Index(rest, ">")
		if gt < 0 {
			return out
		}
		out = append(out, rest[:gt])
		rest = rest[gt+1:]
	}
}

// All returns a snapshot of every registered Key, sorted by Template.
// Build the catalog / docs from this.
func All() []*Key {
	catalogMu.RLock()
	defer catalogMu.RUnlock()
	out := append([]*Key(nil), catalog...)
	// Sort in a stable way — by Template.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1].template > out[j].template; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}
