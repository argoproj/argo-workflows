package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateExamples(t *testing.T) {
	failures, err := ValidateArgoYamlRecursively(".", []string{"testvolume.yaml", "simple-parameters-configmap.yaml", "memoize-simple.yaml"})
	if err != nil {
		t.Errorf("There was an error: %s", err)
	}
	if len(failures) > 0 {
		fails := []string{}
		for path, fail := range failures {
			fails = append(fails, fmt.Sprintf("Validation failed - %s: %s", path, strings.Join(fail, "\n")))
		}
		t.Errorf("There were validation failures:\n%s", strings.Join(fails, "\n"))
	}
}

func TestValidateArgoYamlRecursivelyReportsFailures(t *testing.T) {
	dir := t.TempDir()
	writeYaml := func(name, content string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	badType := writeYaml("bad-type.yaml", ""+
		"kind: Workflow\n"+
		"apiVersion: argoproj.io/v1alpha1\n"+
		"metadata:\n"+
		"  generateName: x-\n"+
		"spec:\n"+
		"  entrypoint: 123\n")

	missingMetadata := writeYaml("missing-metadata.yaml", ""+
		"kind: Workflow\n"+
		"apiVersion: argoproj.io/v1alpha1\n"+
		"spec:\n"+
		"  entrypoint: x\n")

	unknownKind := writeYaml("unknown-kind.yaml", ""+
		"kind: NotAKind\n"+
		"apiVersion: argoproj.io/v1alpha1\n")

	intOrStringPort := writeYaml("intorstring-port.yaml", ""+
		"kind: Workflow\n"+
		"apiVersion: argoproj.io/v1alpha1\n"+
		"metadata:\n"+
		"  generateName: x-\n"+
		"spec:\n"+
		"  entrypoint: x\n"+
		"  templates:\n"+
		"  - name: x\n"+
		"    container:\n"+
		"      image: alpine\n"+
		"    livenessProbe:\n"+
		"      httpGet:\n"+
		"        port: 80\n"+
		"        path: /\n")

	listRoot := writeYaml("list-root.yaml", "- not-a-workflow\n")
	empty := writeYaml("empty.yaml", "")

	failures, err := ValidateArgoYamlRecursively(dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if got := failures[badType]; len(got) != 1 || !strings.Contains(got[0], "in /spec/entrypoint") {
		t.Errorf("bad-type.yaml: expected a single /spec/entrypoint error, got %v", got)
	}
	if got := failures[missingMetadata]; len(got) != 1 || !strings.Contains(got[0], "in (root)") {
		t.Errorf("missing-metadata.yaml: expected a single root-level error, got %v", got)
	}
	if got := failures[unknownKind]; len(got) != 1 || got[0] != `unknown kind "NotAKind"` {
		t.Errorf(`unknown-kind.yaml: expected [unknown kind "NotAKind"], got %v`, got)
	}
	if got, ok := failures[intOrStringPort]; ok {
		t.Errorf("intorstring-port.yaml: expected no failures (integer port is a tolerated IntOrString), got %v", got)
	}
	if got := failures[listRoot]; len(got) != 1 || got[0] != `unknown kind ""` {
		t.Errorf(`list-root.yaml: expected [unknown kind ""], got %v`, got)
	}
	if got := failures[empty]; len(got) != 1 || got[0] != `unknown kind ""` {
		t.Errorf(`empty.yaml: expected [unknown kind ""], got %v`, got)
	}
}
