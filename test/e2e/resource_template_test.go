//go:build executor

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/test/e2e/fixtures"
)

type ResourceTemplateSuite struct {
	fixtures.E2ESuite
}

func (s *ResourceTemplateSuite) TestResourceTemplateWithWorkflow() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-resource-tmpl-with-wf-
spec:
  entrypoint: main
  templates:
    - name: main
      resource:
        action: create
        setOwnerReference: true
        successCondition: status.phase == Succeeded
        failureCondition: status.phase == Failed
        manifest: |
          apiVersion: argoproj.io/v1alpha1
          kind: Workflow
          metadata:
            generateName: k8s-wf-resource-
          spec:
            entrypoint: main
            templates:
              - name: main
                container:
                  image: argoproj/argosay:v2
                  command: ["/argosay"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ResourceTemplateSuite) TestResourceTemplateWithPod() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-resource-tmpl-with-pod-
spec:
  entrypoint: main
  templates:
    - name: main
      resource:
        action: create
        setOwnerReference: true
        successCondition: status.phase == Succeeded
        failureCondition: status.phase == Failed
        manifest: |
          apiVersion: v1
          kind: Pod
          metadata:
            generateName: k8s-pod-resource-
          spec:
            containers:
            - name: argosay-container
              image: argoproj/argosay:v2
              command: ["/argosay"]
            restartPolicy: Never
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ResourceTemplateSuite) TestResourceTemplateWithArtifact() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-resource-tmpl-with-artifact-
spec:
  entrypoint: main
  templates:
    - name: main
      inputs:
        artifacts:
        - name: manifest
          path: /tmp/manifestfrom-path.yaml
          raw:
            data: |
              apiVersion: v1
              kind: Pod
              metadata:
                generateName: k8s-pod-resource-
              spec:
                containers:
                - name: argosay-container
                  image: argoproj/argosay:v2
                  command: ["/argosay"]
                restartPolicy: Never
      resource:
        action: create
        successCondition: status.phase == Succeeded
        failureCondition: status.phase == Failed
        manifestFrom:
          artifact:
            name: manifest
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ResourceTemplateSuite) TestResourceTemplateWithOutputs() {
	s.Given().
		Workflow("@testdata/resource-templates/outputs.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			outputs := status.Nodes[md.Name].Outputs
			require.NotNil(t, outputs)
			parameters := outputs.Parameters
			require.Len(t, parameters, 2)
			assert.Equal(t, "my-pod", parameters[0].Value.String(), "metadata.name is capture for json")
			assert.Equal(t, "my-pod", parameters[1].Value.String(), "metadata.name is capture for jq")
			for _, value := range status.TaskResultsCompletionStatus {
				assert.True(t, value)
			}
		})
}

func (s *ResourceTemplateSuite) TestResourceTemplateAutomountServiceAccountTokenDisabled() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: k8s-resource-tmpl-with-automountservicetoken-disabled-
spec:
  serviceAccountName: argo
  automountServiceAccountToken: false
  executor:
    serviceAccountName: default
  entrypoint: main
  templates:
    - name: main
      resource:
        action: create
        setOwnerReference: true
        successCondition: status.phase == Succeeded
        failureCondition: status.phase == Failed
        manifest: |
          apiVersion: argoproj.io/v1alpha1
          kind: Workflow
          metadata:
            generateName: k8s-wf-resource-
          spec:
            entrypoint: main
            templates:
              - name: main
                container:
                  image: argoproj/argosay:v2
                  command: ["/argosay"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *ResourceTemplateSuite) TestResourceTemplateFailed() {
	s.Given().
		Workflow("@testdata/resource-templates/failed.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow().
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
		})
}

// TestAgentResourceTemplateWithOutputs is the agent-path analogue of
// TestResourceTemplateWithOutputs: the resource is created and watched by the shared
// resource-agent pod instead of a per-node pod, and the output parameters are extracted
// by the agent from the watched object.
func (s *ResourceTemplateSuite) TestAgentResourceTemplateWithOutputs() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: agent-resource-tmpl-outputs-
spec:
  entrypoint: main
  templates:
    - name: main
      resource:
        mode: agent
        action: create
        setOwnerReference: true
        successCondition: status.phase == Succeeded
        failureCondition: status.phase == Failed
        manifest: |
          apiVersion: v1
          kind: Pod
          metadata:
            name: {{workflow.name}}-pod
          spec:
            containers:
            - name: main
              image: argoproj/argosay:v2
              command: ["/argosay"]
            restartPolicy: Never
      outputs:
        parameters:
          - name: json
            valueFrom:
              jsonPath: '{.metadata.name}'
          - name: jq
            valueFrom:
              jqFilter: '.metadata.name'
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, time.Minute*2).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			node := status.Nodes[md.Name]
			assert.Equal(t, wfv1.NodeTypeResourceAgent, node.Type)
			outputs := node.Outputs
			require.NotNil(t, outputs)
			parameters := outputs.Parameters
			require.Len(t, parameters, 2)
			assert.Equal(t, md.Name+"-pod", parameters[0].Value.String(), "metadata.name is captured for json")
			assert.Equal(t, md.Name+"-pod", parameters[1].Value.String(), "metadata.name is captured for jq")
		})
}

// TestAgentResourceTemplateFailureCondition asserts the agent's watch path reports a
// node Failed when the watched resource meets the failure condition.
func (s *ResourceTemplateSuite) TestAgentResourceTemplateFailureCondition() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: agent-resource-tmpl-failure-
spec:
  entrypoint: main
  templates:
    - name: main
      resource:
        mode: agent
        action: create
        setOwnerReference: true
        successCondition: status.phase == Succeeded
        failureCondition: status.phase == Failed
        manifest: |
          apiVersion: v1
          kind: Pod
          metadata:
            name: {{workflow.name}}-pod
          spec:
            containers:
            - name: main
              image: argoproj/argosay:v2
              command: ["/argosay"]
              args: ["exit", "1"]
            restartPolicy: Never
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed, time.Minute*2).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			node := status.Nodes[md.Name]
			assert.Equal(t, wfv1.NodeTypeResourceAgent, node.Type)
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
		})
}

// TestAgentResourceTemplatePluginManifestFrom exercises a manifestFrom artifact served
// by the artifact-driver plugin registered in the e2e controller config ("test"): the
// first step stores the manifest through the plugin, and the resource agent loads it
// back through the plugin sidecar installed in the agent pod.
func (s *ResourceTemplateSuite) TestAgentResourceTemplatePluginManifestFrom() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: agent-resource-tmpl-plugin-
spec:
  entrypoint: main
  artifactRepositoryRef:
    key: plugin-v1
  templates:
    - name: main
      steps:
        - - name: produce
            template: produce
        - - name: apply-manifest
            template: apply-manifest
            arguments:
              artifacts:
                - name: manifest
                  from: "{{steps.produce.outputs.artifacts.manifest}}"

    - name: produce
      container:
        image: argoproj/argosay:v2
        command: [sh, -c]
        args:
          - |
            printf '%s\n' \
              'apiVersion: v1' \
              'kind: ConfigMap' \
              'metadata:' \
              '  name: {{workflow.name}}-from-plugin' \
              'data:' \
              '  state: ready' \
              > /tmp/manifest.yaml
      outputs:
        artifacts:
          - name: manifest
            path: /tmp/manifest.yaml
            archive:
              none: {}

    - name: apply-manifest
      inputs:
        artifacts:
          - name: manifest
            path: /tmp/manifest.yaml
      resource:
        mode: agent
        action: create
        setOwnerReference: true
        successCondition: data.state == ready
        manifestFrom:
          artifact:
            name: manifest
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, time.Minute*2).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

// TestAgentResourceTemplateWorkflowOfWorkflows fans out several child Workflows through
// one shared resource-agent pod — the scenario the agent was built for — and requires
// every child to be watched to completion.
func (s *ResourceTemplateSuite) TestAgentResourceTemplateWorkflowOfWorkflows() {
	s.Given().
		Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: agent-resource-tmpl-wofw-
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: child
            template: create-child
            arguments:
              parameters:
                - name: name
                  value: "{{item}}"
            withItems: [one, two, three]

    - name: create-child
      inputs:
        parameters:
          - name: name
      resource:
        mode: agent
        action: create
        setOwnerReference: true
        successCondition: status.phase == Succeeded
        failureCondition: status.phase == Failed
        manifest: |
          apiVersion: argoproj.io/v1alpha1
          kind: Workflow
          metadata:
            name: {{workflow.name}}-{{inputs.parameters.name}}
          spec:
            entrypoint: main
            templates:
              - name: main
                container:
                  image: argoproj/argosay:v2
                  command: ["/argosay"]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, time.Minute*2).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
			agentNodes := 0
			for _, node := range status.Nodes {
				if node.Type == wfv1.NodeTypeResourceAgent {
					agentNodes++
					assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
				}
			}
			assert.Equal(t, 3, agentNodes, "expected one ResourceAgent node per child workflow")
		})
}

func TestResourceTemplateSuite(t *testing.T) {
	suite.Run(t, new(ResourceTemplateSuite))
}
