//go:build functional
// +build functional

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
            name: '{{ workflow.parameters.cm-name }}'
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

func (s *WorkflowConfigMapSelectorSubstitutionSuite) TestDefaultParamValueWhenNotFound() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-configmapkeyselector-wf-default-param-
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
          default: "default-val"
          configMapKeyRef:
            name: cmref-parameters
            key: not-existing-key
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
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-cmkeyselector-wf-global-arg-default-param-
  label:
    workflows.argoproj.io/test: "true"
spec:
  serviceAccountName: argo
  entrypoint: whalesay
  arguments:
    parameters:
      - name: simple-global-param
        valueFrom:
          default: "default value"
          configMapKeyRef:
            name: not-existing-cm
            key: not-existing-key
  templates:
    - name: whalesay
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args: ["sleep 1; echo -n {{workflow.parameters.simple-global-param}} > /tmp/message.txt"]
      outputs:
        parameters:
         - name: message
           valueFrom:
             path: /tmp/message.txt
`).
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
