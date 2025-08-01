//go:build functional

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type WorkflowConfigMapSelectorSubstitutionSuite struct {
	fixtures.E2ESuite
}

func (s *WorkflowConfigMapSelectorSubstitutionSuite) TestKeySubstitution() {
	s.Given().
		Workflow("@functional/workflow-template-configmapkeyselector-wf.yaml").
		When().
		CreateConfigMap(
			"cmref-parameters",
			map[string]string{"msg": "hello world"},
			map[string]string{"workflows.argoproj.io/configmap-type": "Parameter"}).
		Wait(1 * time.Second).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		DeleteConfigMap("cmref-parameters").
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowConfigMapSelectorSubstitutionSuite) TestNameSubstitution() {
	s.Given().
		Workflow("@functional/workflow-template-configmapkeyselector-wf.yaml").
		When().
		CreateConfigMap(
			"cmref-parameters",
			map[string]string{"msg": "hello world"},
			map[string]string{"workflows.argoproj.io/configmap-type": "Parameter"}).
		Wait(1 * time.Second).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		DeleteConfigMap("cmref-parameters").
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowConfigMapSelectorSubstitutionSuite) TestInvalidNameParameterSubstitution() {
	s.Given().
		Workflow("@functional/workflow-template-configmapkeyselector-wf.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeErrored)
}

func (s *WorkflowConfigMapSelectorSubstitutionSuite) TestDefaultParamValueWhenNotFound() {
	s.Given().
		Workflow("@functional/workflow-template-configmapkeyselector-wf-default-param.yaml").
		When().
		CreateConfigMap(
			"cmref-parameters",
			map[string]string{"msg": "hello world"},
			map[string]string{"workflows.argoproj.io/configmap-type": "Parameter"}).
		Wait(1 * time.Second).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		DeleteConfigMap("cmref-parameters").
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowConfigMapSelectorSubstitutionSuite) TestGlobalArgDefaultCMParamValueWhenNotFound() {
	s.Given().
		Workflow("@functional/workflow-template-cmkeyselector-wf-global-arg-default-param.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "default value", status.Nodes[metadata.Name].Outputs.Parameters[0].Value.String())
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func TestConfigMapKeySelectorSubstitutionSuite(t *testing.T) {
	suite.Run(t, new(WorkflowConfigMapSelectorSubstitutionSuite))
}
