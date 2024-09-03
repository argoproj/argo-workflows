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
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-inputs-overridable-wf-
  label:
    workflows.argoproj.io/test: "true"
spec:
  serviceAccountName: argo
  entrypoint: whalesay
  arguments:
    parameters:
      - name: message
        value: arg-value
  templates:
    - name: whalesay
      container:
        image: argoproj/argosay:v2
        args:
        - echo
        - "{{inputs.parameters.message}}"
      inputs:
        parameters:
          - name: message
            valueFrom:
              configMapKeyRef:
                name: cmref-parameters
                key: cmref-key
`).
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
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-inputs-overridable-wf-
  label:
    workflows.argoproj.io/test: "true"
spec:
  serviceAccountName: argo
  entrypoint: whalesay
  arguments:
    parameters:
      - name: message
        valueFrom:
          configMapKeyRef:
            name: new-cmref-parameters
            key: cmref-key
  templates:
    - name: whalesay
      container:
        image: argoproj/argosay:v2
        args:
        - echo
        - "{{inputs.parameters.message}}"
      inputs:
        parameters:
          - name: message
            valueFrom:
              configMapKeyRef:
                name: cmref-parameters
                key: cmref-key
`).
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
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-wf-
  label:
    workflows.argoproj.io/test: "true"
spec:
  serviceAccountName: argo
  entrypoint: whalesay
  arguments:
    parameters:
      - name: message
        value: arg-value
  templates:
    - name: whalesay
      container:
        image: argoproj/argosay:v2
        args:
        - echo
        - "{{inputs.parameters.message}}"
      inputs:
        parameters:
          - name: message
            value: input-value
`).
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
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-inputs-overridable-wf-
  label:
    workflows.argoproj.io/test: "true"
spec:
  serviceAccountName: argo
  entrypoint: whalesay
  arguments:
    parameters:
      - name: message
        valueFrom:
          configMapKeyRef:
            name: cmref-parameters
            key: cmref-key
  templates:
    - name: whalesay
      container:
        image: argoproj/argosay:v2
        args:
        - echo
        - "{{inputs.parameters.message}}"
      inputs:
        parameters:
          - name: message
            value: input-value
`).
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
