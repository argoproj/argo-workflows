//go:build functional
// +build functional

package e2e

import (
	"testing"

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
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-configmapkeyselector-wf-
  label:
    workflows.argoproj.io/test: "true"
spec:
  entrypoint: whalesay
  arguments:
    parameters:
    - name: message
      value: msg
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
        valueFrom:
          configMapKeyRef:
            name: cmref-parameters
            key: '{{ workflow.parameters.message }}'
    container:
      image: argoproj/argosay:v2
      args:
        - echo
        - "{{inputs.parameters.message}}"
`).
		When().
		CreateConfigMap(
			"cmref-parameters",
			map[string]string{"msg": "hello world"},
			map[string]string{"workflows.argoproj.io/configmap-type": "Parameter"}).
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
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-configmapkeyselector-wf-
  label:
    workflows.argoproj.io/test: "true"
spec:
  entrypoint: whalesay
  arguments:
    parameters:
    - name: cm-name
      value: cmref-parameters
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
        valueFrom:
          configMapKeyRef:
            name: '{{ workflow.parameters.cm-name}}'
            key: msg
    container:
      image: argoproj/argosay:v2
      args:
        - echo
        - "{{inputs.parameters.message}}"
`).
		When().
		CreateConfigMap(
			"cmref-parameters",
			map[string]string{"msg": "hello world"},
			map[string]string{"workflows.argoproj.io/configmap-type": "Parameter"}).
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
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-configmapkeyselector-wf-
  label:
    workflows.argoproj.io/test: "true"
spec:
  entrypoint: whalesay
  arguments:
    parameters:
    - name: cm-name
      value: cmref-parameters
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
        valueFrom:
          configMapKeyRef:
            name: '{{ workflow.parameters.cm-name}'
            key: msg
    container:
      image: argoproj/argosay:v2
      args:
        - echo
        - "{{inputs.parameters.message}}"
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeErrored)
}

func TestConfigMapKeySelectorSubstitutionSuite(t *testing.T) {
	suite.Run(t, new(WorkflowConfigMapSelectorSubstitutionSuite))
}
