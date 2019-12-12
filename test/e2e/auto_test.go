package e2e

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/e2e/fixtures"
)

type AutoSuite struct {
	fixtures.E2ESuite
}

func (s AutoSuite) TestFunctional() {
	s.runDir("functional", wfv1.NodeSucceeded)
}

func (s AutoSuite) TestExpectedFailures() {
	s.runDir("expectedfailures", wfv1.NodeFailed)
}

func (s AutoSuite) TestLintFail() {
	s.runDir("lintfail", wfv1.NodeError)
}

func (s AutoSuite) runDir(dir string, nodePhase wfv1.NodePhase) {
	err := filepath.Walk(dir, func(path string, file os.FileInfo, err error) error {
		if !file.IsDir() {
			s.runFile(path, nodePhase)
		}
		return err
	})
	assert.NoError(s.T(), err)
}

func (s AutoSuite) runFile(name string, nodePhase wfv1.NodePhase) {
	s.Run(name, func() {
		bytes, err := ioutil.ReadFile(name)
		if assert.NoError(s.T(), err) {
			obj := &unstructured.Unstructured{}
			err = yaml.Unmarshal(bytes, &obj.Object)
			if assert.NoError(s.T(), err) {
				skip, ok := obj.GetAnnotations()["argo-e2e/skip"]
				if !ok || skip == "true" {
					s.T().SkipNow()
				}
				s.Given().
					Workflow("@" + name).
					When().
					SubmitWorkflow().
					WaitForWorkflow().
					Then().
					Expect(func(t *testing.T, wf *wfv1.WorkflowStatus) {
						assert.Equal(t, nodePhase, wf.Phase)
					})
			}
		}
	})
}

func TestFunctionalSuite(t *testing.T) {
	suite.Run(t, new(AutoSuite))
}
