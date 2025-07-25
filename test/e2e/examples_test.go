//go:build examples

package e2e

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	fileutil "github.com/argoproj/argo-workflows/v3/util/file"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type ExamplesSuite struct {
	fixtures.E2ESuite
}

func (s *ExamplesSuite) BeforeTest(suiteName, testName string) {
	s.E2ESuite.BeforeTest(suiteName, testName)
	s.Given().KubectlApply("../../examples/configmaps/simple-parameters-configmap.yaml", fixtures.NoError)
}

func (s *ExamplesSuite) TestExampleWorkflows() {
	ctx := logging.TestContext(s.T().Context())
	err := fileutil.WalkManifests(ctx, "../../examples", func(path string, data []byte) error {
		wfs, err := common.SplitWorkflowYAMLFile(ctx, data, true)
		if err != nil {
			s.T().Fatalf("Error parsing %s: %v", path, err)
		}
		for _, wf := range wfs {
			if _, ok := wf.GetLabels()["workflows.argoproj.io/test"]; ok {
				s.T().Logf("Found example workflow at %s with test label\n", path)
				s.Given().
					ExampleWorkflow(&wf).
					When().
					SubmitWorkflow().
					WaitForWorkflow(fixtures.ToBeSucceeded)
			}
		}
		return nil
	})
	s.CheckError(err)
}

func TestExamplesSuite(t *testing.T) {
	suite.Run(t, new(ExamplesSuite))
}
