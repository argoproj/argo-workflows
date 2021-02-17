// +build examples

package e2e

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type ExamplesSuite struct {
	fixtures.E2ESuite
}

func (s *ExamplesSuite) Test() {
	dir, err := ioutil.ReadDir("../../examples")
	s.Assert().NoError(err)
	for _, info := range dir {
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml") {
			continue
		}
		s.Run(info.Name(), func() {
			s.DeleteResources()
			un := &unstructured.Unstructured{}
			data, err := ioutil.ReadFile("../../examples/" + info.Name())
			s.Assert().NoError(err)
			err = yaml.Unmarshal(data, un)
			s.Assert().NoError(err)
			if un.GetKind() != "Workflow" {
				s.T().SkipNow()
			}
			if un.GetLabels()[fixtures.Label] == "" {
				s.T().SkipNow()
			}
			data, err = yaml.Marshal(un)
			s.Assert().NoError(err)
			s.Given().
				Workflow(string(data)).
				When().
				SubmitWorkflow().
				WaitForWorkflow(fixtures.ToBeSucceeded, "to be succeeded")
		})
	}
}

func TestExamplesSuite(t *testing.T) {
	suite.Run(t, new(ExamplesSuite))
}
