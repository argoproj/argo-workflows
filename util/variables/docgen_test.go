package variables_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	v "github.com/argoproj/argo-workflows/v4/util/variables"
	_ "github.com/argoproj/argo-workflows/v4/util/variables/keys"
)

// -write flag lets you regenerate docs/variable-flow/variables.md in-place.
//
//	go test -run TestGenerateMarkdown -write ./util/variables/
var writeDocs = flag.Bool("write", false, "write docs/variable-flow/variables.md")

// TestGenerateMarkdown sanity-checks the doc output and optionally writes it.
func TestGenerateMarkdown(t *testing.T) {
	md := v.GenerateMarkdown()
	if len(md) < 500 {
		t.Fatalf("doc looks suspiciously short: %d bytes", len(md))
	}
	for _, must := range []string{
		"# Workflow variables catalog",
		"## 1. Alphabetical index",
		"## 2. Grouped by Kind",
		"## 3. Matrix by TemplateKind",
		"## 4. Grouped by LifecyclePhase",
		"workflow.name",
		"workflow.parameters.<name>",
		"steps.<name>.outputs.result",
		"tasks.<name>.outputs.result",
		"item.<key>",
		"retries.lastExitCode",
		"pod.name",
	} {
		if !containsSubstr(md, must) {
			t.Errorf("generated doc missing %q", must)
		}
	}

	if !*writeDocs {
		return
	}
	target := findRepoRoot(t)
	out := filepath.Join(target, "docs", "variable-flow", "variables.md")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(out, []byte(md), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Logf("wrote %s (%d bytes)", out, len(md))
}

func containsSubstr(s, needle string) bool {
	for i := 0; i+len(needle) <= len(s); i++ {
		if s[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// findRepoRoot walks up from the test's working dir until it finds a go.mod.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	d, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			t.Fatalf("could not find go.mod upward of test dir")
		}
		d = parent
	}
}
