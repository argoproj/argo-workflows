package variables_test

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	v "github.com/argoproj/argo-workflows/v4/util/variables"
	// Trigger Define() for the canonical catalog.
	_ "github.com/argoproj/argo-workflows/v4/util/variables/keys"
)

// TestKeyHandle_Opaque encodes the correctness-by-construction invariant at
// the type-system level: outside this package, the Key type has no
// exported constructor and its fields are unexported. The only way to get
// a Key is via Define (or a pre-declared handle).
//
// This test is as much documentation as verification: using reflection, we
// check that Key has no exported fields. If a future contributor adds one,
// the test fails and forces a review.
func TestKeyHandle_Opaque(t *testing.T) {
	var k v.Key
	typ := reflect.TypeOf(k)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.IsExported() {
			t.Errorf("Key.%s is exported — breaks correctness-by-construction", f.Name)
		}
	}
}

// TestScope_Opaque is the same check for Scope. If Scope exposes an
// exported map or write helper, arbitrary keys can be written and the
// contract is broken.
func TestScope_Opaque(t *testing.T) {
	var s v.Scope
	typ := reflect.TypeOf(s)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.IsExported() {
			t.Errorf("Scope.%s is exported — breaks correctness-by-construction", f.Name)
		}
	}
}

// TestDefine_RegistersInCatalog confirms that calling Define makes the Key
// appear in All(). This closes the loop: if you call Define, the variable
// is catalogued; if you don't, you have no handle to write it.
func TestDefine_RegistersInCatalog(t *testing.T) {
	k := v.Define(v.Spec{
		Template: "test.registers-in-catalog",
		Kind:     v.KindGlobal,
	})
	found := false
	for _, entry := range v.All() {
		if entry == k {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Define did not register %q in All()", k.Template())
	}
}

// TestDefine_DuplicatesPanic ensures that two packages can't accidentally
// declare the same variable. If a duplicate slips in, init panics and
// the binary refuses to start.
func TestDefine_DuplicatesPanic(t *testing.T) {
	_ = v.Define(v.Spec{Template: "test.dup.1", Kind: v.KindGlobal})
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on duplicate Define")
		} else if !strings.Contains(r.(string), "duplicate Define") {
			t.Errorf("expected duplicate-Define panic, got %v", r)
		}
	}()
	_ = v.Define(v.Spec{Template: "test.dup.1", Kind: v.KindGlobal})
}

// TestDefine_EmptyTemplatePanics — empty templates are programmer errors.
func TestDefine_EmptyTemplatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on empty template")
		}
	}()
	_ = v.Define(v.Spec{Template: "", Kind: v.KindGlobal})
}

// TestConcretize_PlaceholderSubstitution walks the templated-key case.
func TestConcretize_PlaceholderSubstitution(t *testing.T) {
	k := v.Define(v.Spec{
		Template: "test.concretize.<a>.<b>",
		Kind:     v.KindGlobal,
	})
	if got := k.Concretize("foo", "bar"); got != "test.concretize.foo.bar" {
		t.Errorf("wrong: %q", got)
	}
	// Wrong arity panics.
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on wrong arity")
		}
	}()
	_ = k.Concretize("only-one")
}

// TestKeySet_IsOnlyWritePath shows end-to-end: a Scope starts empty, the
// only way to populate it is Key.Set, and the value round-trips via
// Key.Get.
func TestKeySet_IsOnlyWritePath(t *testing.T) {
	s := v.NewScope()
	if s.Len() != 0 {
		t.Fatalf("new scope should be empty, got %d", s.Len())
	}

	k := v.Define(v.Spec{
		Template: "test.key-set.<name>",
		Kind:     v.KindGlobal,
	})
	k.Set(s, "value1", "alice")
	k.Set(s, "value2", "bob")

	got, ok := k.Get(s, "alice")
	if !ok || got != "value1" {
		t.Errorf("alice: got %v ok=%v", got, ok)
	}
	if !k.Has(s, "bob") {
		t.Errorf("bob should be present")
	}
	if s.Len() != 2 {
		t.Errorf("expected 2 entries, got %d", s.Len())
	}
}

// TestCatalog_ContainsRegisteredKeys verifies the canonical keys package
// has populated the catalog on import.
func TestCatalog_ContainsRegisteredKeys(t *testing.T) {
	all := v.All()
	names := make([]string, 0, len(all))
	for _, k := range all {
		names = append(names, k.Template())
	}
	sort.Strings(names)

	// Spot check that a few globals are present.
	want := []string{
		"workflow.name",
		"workflow.namespace",
		"workflow.parameters.<name>",
		"workflow.status",
		"workflow.failures",
	}
	for _, w := range want {
		if sort.SearchStrings(names, w); !contains(names, w) {
			t.Errorf("catalog missing %q", w)
		}
	}
}

// TestScope_AsAnyMapIsASnapshot — mutating the returned map must not affect
// the Scope; there is no write-back path.
func TestScope_AsAnyMapIsASnapshot(t *testing.T) {
	k := v.Define(v.Spec{Template: "test.snapshot", Kind: v.KindGlobal})
	s := v.NewScope()
	k.Set(s, "live")

	snap := s.AsAnyMap()
	snap["test.snapshot"] = "tampered"
	snap["injected"] = "bypass"

	got, _ := k.Get(s)
	if got != "live" {
		t.Errorf("snapshot write leaked back into scope: %v", got)
	}
	if _, ok := s.AsAnyMap()["injected"]; ok {
		t.Errorf("injected key appeared in scope via snapshot")
	}
}

func contains(sorted []string, s string) bool {
	i := sort.SearchStrings(sorted, s)
	return i < len(sorted) && sorted[i] == s
}
