package exprtrace

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSetCapturesCaller(t *testing.T) {
	m := New()
	m.Set("k", "v")

	e, ok := m.Entry("k")
	if !ok {
		t.Fatalf("entry not found")
	}
	if e.Value != "v" {
		t.Errorf("wrong value: %v", e.Value)
	}
	if !strings.HasSuffix(e.File, "map_test.go") {
		t.Errorf("file attribution wrong: %s", e.File)
	}
	if !strings.Contains(e.Function, "TestSetCapturesCaller") {
		t.Errorf("function attribution wrong: %s", e.Function)
	}
	// Sanity-check line number is near the Set call.
	_, _, line, _ := runtime.Caller(0)
	if e.Line <= 0 || e.Line >= line {
		t.Errorf("line %d not plausible (current %d)", e.Line, line)
	}
}

func TestSetFromCallerSkipsHelper(t *testing.T) {
	m := New()
	setViaHelper(m, "k", "v")

	e, _ := m.Entry("k")
	if !strings.Contains(e.Function, "TestSetFromCallerSkipsHelper") {
		t.Errorf("expected attribution to test func, got %s", e.Function)
	}
}

func setViaHelper(m *Map, k string, v any) {
	m.SetFromCaller(k, v, 1)
}

func TestMergePreservesProvenance(t *testing.T) {
	a := New()
	b := New()
	a.Set("a_key", 1)
	b.Set("b_key", 2)

	a.Merge(b)

	ea, _ := a.Entry("a_key")
	eb, _ := a.Entry("b_key")
	if !strings.Contains(ea.Function, "TestMergePreservesProvenance") {
		t.Errorf("a_key provenance lost: %s", ea.Function)
	}
	if !strings.Contains(eb.Function, "TestMergePreservesProvenance") {
		t.Errorf("b_key provenance lost: %s", eb.Function)
	}
}

func TestAsAnyMap(t *testing.T) {
	m := New()
	m.Set("a", "1")
	m.Set("b", 2)

	out := m.AsAnyMap()
	if out["a"] != "1" || out["b"] != 2 {
		t.Errorf("AsAnyMap wrong: %#v", out)
	}
	// Modifying the returned map must not affect the source.
	out["c"] = "leak"
	if _, ok := m.Get("c"); ok {
		t.Errorf("AsAnyMap leaked a back-reference")
	}
}

func TestDumpD2NoOpWithoutDir(t *testing.T) {
	m := New()
	m.Set("k", "v")
	path, err := m.DumpD2(DumpTarget{Expression: "k == 'v'"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Errorf("expected no-op when dir is empty, got %q", path)
	}
}

func TestDumpD2WritesFile(t *testing.T) {
	dir := t.TempDir()
	m := New()
	m.Set("workflow.name", "hello")
	m.Set("steps.A.outputs.result", "10")

	path, err := m.DumpD2(DumpTarget{
		Dir:        dir,
		Expression: "steps.A.outputs.result == '10'",
		CallerFile: "util/template/expression_template.go",
		CallerLine: 182,
		Label:      "test",
	})
	if err != nil {
		t.Fatalf("DumpD2: %v", err)
	}
	if path == "" {
		t.Fatalf("expected a file path")
	}
	if filepath.Dir(path) != dir {
		t.Errorf("wrong dir: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	out := string(data)

	for _, must := range []string{
		"expression : steps.A.outputs.result == '10'",
		"expr.Compile",
		"\"workflow.name\"",
		"\"steps.A.outputs.result\"",
		"env -> compile",
	} {
		if !strings.Contains(out, must) {
			t.Errorf("output missing %q\n--- output ---\n%s\n", must, out)
		}
	}
}

func TestRenderD2GroupsBySourceSite(t *testing.T) {
	m := New()
	// These two Sets share the same source site (this function + adjacent line)
	m.Set("a", 1)
	m.Set("b", 2)

	out := m.RenderD2(DumpTarget{
		Expression: "a + b",
		CallerFile: "test.go",
		CallerLine: 1,
	})
	// Both keys should have an edge from sources into env.
	if !strings.Contains(out, "sources.") {
		t.Errorf("expected sources.* node: %s", out)
	}
	if !strings.Contains(out, "-> env.\"a\": set") {
		t.Errorf("missing edge to a: %s", out)
	}
	if !strings.Contains(out, "-> env.\"b\": set") {
		t.Errorf("missing edge to b: %s", out)
	}
}

func TestCloneIsIndependent(t *testing.T) {
	m := New()
	m.Set("k", "v1")

	c := m.Clone()
	m.Set("k", "v2")

	if v, _ := c.Get("k"); v != "v1" {
		t.Errorf("clone mutated: %v", v)
	}
}
