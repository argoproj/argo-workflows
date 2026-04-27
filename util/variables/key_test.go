package variables_test

import (
	"sort"
	"strings"
	"testing"

	v "github.com/argoproj/argo-workflows/v4/util/variables"
	_ "github.com/argoproj/argo-workflows/v4/util/variables/keys"
)

func TestDefine_DuplicatesPanic(t *testing.T) {
	_ = v.Define(v.Spec{Template: "test.dup.1", Kind: v.KindGlobal})
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(r.(string), "duplicate Define") {
			t.Errorf("expected duplicate-Define panic, got %v", r)
		}
	}()
	_ = v.Define(v.Spec{Template: "test.dup.1", Kind: v.KindGlobal})
}

func TestDefine_EmptyTemplatePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("expected panic on empty template")
		}
	}()
	_ = v.Define(v.Spec{Template: "", Kind: v.KindGlobal})
}

func TestConcretize(t *testing.T) {
	k := v.Define(v.Spec{Template: "test.concretize.<a>.<b>", Kind: v.KindGlobal})
	if got := k.Concretize("foo", "bar"); got != "test.concretize.foo.bar" {
		t.Errorf("wrong: %q", got)
	}
	defer func() {
		if recover() == nil {
			t.Errorf("expected panic on wrong arity")
		}
	}()
	_ = k.Concretize("only-one")
}

func TestKeySet_AndSnapshot(t *testing.T) {
	k := v.Define(v.Spec{Template: "test.set.<n>", Kind: v.KindGlobal})
	s := v.NewScope()
	k.Set(s, "live", "alice")

	snap := s.AsAnyMap()
	snap["test.set.alice"] = "tampered"
	snap["injected"] = "bypass"

	if v := s.AsAnyMap()["test.set.alice"]; v != "live" {
		t.Errorf("snapshot leaked back: %v", v)
	}
	if _, ok := s.AsAnyMap()["injected"]; ok {
		t.Errorf("injected key appeared")
	}
}

func TestCatalog_ContainsRegisteredKeys(t *testing.T) {
	names := make([]string, 0)
	for _, k := range v.All() {
		names = append(names, k.Template())
	}
	sort.Strings(names)
	for _, want := range []string{
		"workflow.name", "workflow.namespace",
		"workflow.parameters.<name>", "workflow.status", "workflow.failures",
	} {
		i := sort.SearchStrings(names, want)
		if i >= len(names) || names[i] != want {
			t.Errorf("catalog missing %q", want)
		}
	}
}
