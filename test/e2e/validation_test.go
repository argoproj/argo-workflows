// +build e2e

package e2e

import (
	"fmt"
	"strings"
	"testing"

	"github.com/argoproj/argo/test/util"
)

func TestValidateExamples(t *testing.T) {
	failures, err := util.ValidateArgoYamlRecursively("../../examples", []string{"testvolume.yaml"})
	if err != nil {
		t.Errorf("There was an error: %s", err)
	}
	if len(failures) > 0 {
		var fails = []string{}
		for path, fail := range failures {
			fails = append(fails, fmt.Sprintf("Validation failed - %s: %s", path, strings.Join(fail, "\n")))
		}
		t.Errorf("There were validation failures:\n%s", strings.Join(fails, "\n"))
	}
}

func TestValidateE2E(t *testing.T) {
	failures, err := util.ValidateArgoYamlRecursively(".", []string{
		"argo-server-deployment.yaml",
		"kustomization.yaml",
		"lintfail",
		"malformed",
		"manifests",
		"argo-server-test-role.yaml",
		"testvolume.yaml",
	})
	if err != nil {
		t.Errorf("There was an error: %s", err)
	}
	if len(failures) > 0 {
		var fails = []string{}
		for path, fail := range failures {
			fails = append(fails, fmt.Sprintf("Validation failed - %s: %s", path, strings.Join(fail, "\n")))
		}
		t.Errorf("There were validation failures:\n%s", strings.Join(fails, "\n"))
	}
}
