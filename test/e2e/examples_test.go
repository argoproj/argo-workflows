//go:build examples

package e2e

import (
	"strconv"
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
			isTestBroken := false
			isEvironmentNotReady := false
			isTestTooLong := false
			isTestExpectedToFail := false
			isTestBrokenRaw, noTestBrokenLabelExists := wf.GetLabels()["workflows.argoproj.io/no-test-broken"]
			if noTestBrokenLabelExists {
				isTestBroken, err = strconv.ParseBool(isTestBrokenRaw)
				if err != nil {
					s.T().Fatalf("Error parsing annotation \"workflows.argoproj.io/no-test-broken\": %v", err)
				}
			}
			isEvironmentNotReadyRaw, noTestBrokenEnvironmentLabelExists := wf.GetLabels()["workflows.argoproj.io/no-test-environment"]
			if noTestBrokenEnvironmentLabelExists {
				isEvironmentNotReady, err = strconv.ParseBool(isEvironmentNotReadyRaw)
				if err != nil {
					s.T().Fatalf("Error parsing annotation \"workflows.argoproj.io/no-test-environment\": %v", err)
				}
			}
			isTestTooLongRaw, noTestTooLongLabelExists := wf.GetLabels()["workflows.argoproj.io/no-test-duration"]
			if noTestTooLongLabelExists {
				isTestTooLong, err = strconv.ParseBool(isTestTooLongRaw)
				if err != nil {
					s.T().Fatalf("Error parsing annotation \"workflows.argoproj.io/no-test-duration\": %v", err)
				}
			}
			isTestExpectedToFailRaw, noTestTooLongLabelExists := wf.GetLabels()["workflows.argoproj.io/no-test-expected-failure"]
			if noTestTooLongLabelExists {
				isTestExpectedToFail, err = strconv.ParseBool(isTestExpectedToFailRaw)
				if err != nil {
					s.T().Fatalf("Error parsing annotation \"workflows.argoproj.io/no-test-expected-failure\": %v", err)
				}
			}
			if isTestBroken || isEvironmentNotReady || isTestTooLong || isTestExpectedToFail {
				continue
			}
			s.T().Run(path, func(t *testing.T) {
				s.T().Logf("Found example workflow at %s\n", path)
				s.Given().
					ExampleWorkflow(&wf).
					When().
					SubmitWorkflow().
					WaitForWorkflow(fixtures.ToBeSucceeded)
			})
		}
		return nil
	})
	s.CheckError(err)
}

func TestExamplesSuite(t *testing.T) {
	suite.Run(t, new(ExamplesSuite))
}
