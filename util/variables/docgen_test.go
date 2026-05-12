package variables_test

import (
	"flag"
	"os"
	"strings"
	"testing"

	v "github.com/argoproj/argo-workflows/v4/util/variables"
	_ "github.com/argoproj/argo-workflows/v4/util/variables/keys"
)

// -write regenerates docs/variable-flow/variables.md in-place:
//
//	go test -run TestGenerateMarkdown ./util/variables/ -args -write
var writeDocs = flag.Bool("write", false, "write docs/variable-flow/variables.md")

func TestGenerateMarkdown(t *testing.T) {
	md := v.GenerateMarkdown()
	for _, must := range []string{
		"# Workflow variables catalog",
		"workflow.name", "workflow.parameters.<name>",
		"steps.<name>.outputs.result", "item.<key>", "pod.name",
	} {
		if !strings.Contains(md, must) {
			t.Errorf("generated doc missing %q", must)
		}
	}
	if !*writeDocs {
		return
	}
	if err := os.WriteFile("../../docs/variable-flow/variables.md", []byte(md), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
