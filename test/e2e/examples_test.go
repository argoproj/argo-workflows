// +build examples

package e2e

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/robertkrimen/otto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type ExamplesSuite struct {
	fixtures.E2ESuite
}

func jsonify(v interface{}) interface{} {
	data, _ := json.Marshal(v)
	x := make(map[string]interface{})
	_ = json.Unmarshal(data, &x)
	return x
}

func (s *ExamplesSuite) Test() {
	dir, err := ioutil.ReadDir("../../examples")
	s.Assert().NoError(err)
	for _, info := range dir {
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml") {
			continue
		}
		data, err := ioutil.ReadFile("../../examples/" + info.Name())
		s.Assert().NoError(err)
		un := &unstructured.Unstructured{}
		s.Assert().NoError(yaml.Unmarshal(data, un))
		if un.GetLabels()[fixtures.Label] == "" {
			continue
		}
		s.Run(info.Name(), func() {
			s.DeleteResources()
			s.Assert().NoError(err)
			s.Given().
				Workflow(string(data)).
				When().
				SubmitWorkflow().
				WaitForWorkflow().
				Then().
				ExpectWorkflow(func(t *testing.T, m *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
					verify, ok := m.GetAnnotations()[fixtures.Verify]
					nodes := wfv1.Nodes{}
					for _, n := range status.Nodes {
						nodes[n.DisplayName] = n
					}
					if ok {
						vm := otto.New()
						assert.NoError(t, vm.Set("metadata", jsonify(m)))
						assert.NoError(t, vm.Set("status", jsonify(status)))
						assert.NoError(t, vm.Set("nodes", jsonify(nodes)))
						_, err := vm.Run(verify)
						assert.NoError(t, err, verify)
					} else {
						assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
					}
				})
		})
	}
}

func TestExamplesSuite(t *testing.T) {
	suite.Run(t, new(ExamplesSuite))
}
