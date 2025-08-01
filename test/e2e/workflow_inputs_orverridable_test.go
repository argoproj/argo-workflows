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

type WorkflowInputsOverridableSuite struct {
	fixtures.E2ESuite
}

func (s *WorkflowInputsOverridableSuite) TestArgsValueParamsOverrideInputParamsValueFrom() {
	s.Given().
		Workflow("@functional/workflow-inputs-overridable-wf.yaml").
		When().
		CreateConfigMap(
			"cmref-parameters",
			map[string]string{"cmref-key": "input-value"},
			map[string]string{"workflows.argoproj.io/configmap-type": "Parameter"}).
		Wait(1 * time.Second).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		DeleteConfigMap("cmref-parameters").
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "arg-value", status.Nodes[metadata.Name].Inputs.Parameters[0].Value.String())
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowInputsOverridableSuite) TestArgsValueFromParamsOverrideInputParamsValueFrom() {
	s.Given().
		Workflow("@functional/workflow-inputs-overridable-wf.yaml").
		When().
		CreateConfigMap(
			"cmref-parameters",
			map[string]string{"cmref-key": "input-value"},
			map[string]string{"workflows.argoproj.io/configmap-type": "Parameter"}).
		CreateConfigMap(
			"new-cmref-parameters",
			map[string]string{"cmref-key": "arg-value"},
			map[string]string{"workflows.argoproj.io/configmap-type": "Parameter"}).
		Wait(1 * time.Second).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		DeleteConfigMap("cmref-parameters").
		DeleteConfigMap("new-cmref-parameters").
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "arg-value", status.Nodes[metadata.Name].Inputs.Parameters[0].Value.String())
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowInputsOverridableSuite) TestArgsValueParamsOverrideInputParamsValue() {
	s.Given().
		Workflow("@functional/workflow-template-wf.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "arg-value", status.Nodes[metadata.Name].Inputs.Parameters[0].Value.String())
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowInputsOverridableSuite) TestArgsValueFromParamsOverrideInputParamsValue() {
	s.Given().
		Workflow("@functional/workflow-inputs-overridable-wf.yaml").
		When().
		CreateConfigMap(
			"cmref-parameters",
			map[string]string{"cmref-key": "arg-value"},
			map[string]string{"workflows.argoproj.io/configmap-type": "Parameter"}).
		Wait(1 * time.Second).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		DeleteConfigMap("cmref-parameters").
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, "arg-value", status.Nodes[metadata.Name].Inputs.Parameters[0].Value.String())
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func TestWorkflowInputsOverridableSuiteSuite(t *testing.T) {
	suite.Run(t, new(WorkflowInputsOverridableSuite))
}
