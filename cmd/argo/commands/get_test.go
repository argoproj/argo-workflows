package commands

import (
	"bytes"
	"context"
	"fmt"

	"sigs.k8s.io/yaml"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"text/tabwriter"

	"github.com/argoproj/argo/pkg/apiclient/mocks"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/mock"
)

var wfWithStatus = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-06-24T22:53:35Z"
  generation: 6
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
  name: hello-world
  namespace: default
  resourceVersion: "1110858"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/default/workflows/hello-world-2xg9p
  uid: 8c25e2e7-6b35-4a49-a667-87b4cd1afa3c
spec:
  arguments: {}
  entrypoint: whalesay
  templates:
  - arguments: {}
    container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
status:
  conditions:
  - status: "True"
    type: Completed
  finishedAt: "2020-06-24T22:53:41Z"
  nodes:
    hello-world-2xg9p:
      displayName: hello-world-2xg9p
      finishedAt: "2020-06-24T22:53:39Z"
      id: hello-world-2xg9p
      name: hello-world-2xg9p
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2020-06-24T22:53:35Z"
      templateName: whalesay
      templateScope: local/hello-world-2xg9p
      type: Pod
  phase: Succeeded
  resourcesDuration:
    cpu: 3
    memory: 0
  startedAt: "2020-06-24T22:53:35Z"
`

func testPrintNodeImpl(t *testing.T, expected string, node wfv1.NodeStatus, nodePrefix string, getArgs getFlags) {
	var result bytes.Buffer
	w := tabwriter.NewWriter(&result, 0, 8, 1, '\t', 0)
	printNode(w, node, nodePrefix, getArgs)
	err := w.Flush()
	assert.NoError(t, err)
	assert.Equal(t, expected, result.String())
}

// TestPrintNode
func TestPrintNode(t *testing.T) {
	nodeName := "testNode"
	nodePrefix := ""
	nodeTemplateName := "testTemplate"
	nodeTemplateRefName := "testTemplateRef"
	nodeID := "testID"
	nodeMessage := "test"
	getArgs := getFlags{
		output: "",
		status: "",
	}
	timestamp := metav1.Time{
		Time: time.Now(),
	}
	node := wfv1.NodeStatus{
		Name:        nodeName,
		Phase:       wfv1.NodeRunning,
		DisplayName: nodeName,
		Type:        wfv1.NodeTypePod,
		ID:          nodeID,
		StartedAt:   timestamp,
		FinishedAt:  timestamp,
		Message:     nodeMessage,
	}
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t%s\t%s\t%s\t%s\n", jobStatusIconMap[wfv1.NodeRunning], nodeName, "", nodeID, "0s", nodeMessage), node, nodePrefix, getArgs)

	node.TemplateName = nodeTemplateName
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t%s\t%s\t%s\t%s\n", jobStatusIconMap[wfv1.NodeRunning], nodeName, nodeTemplateName, nodeID, "0s", nodeMessage), node, nodePrefix, getArgs)

	node.Type = wfv1.NodeTypeSuspend
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t%s\t%s\t%s\t%s\n", nodeTypeIconMap[wfv1.NodeTypeSuspend], nodeName, nodeTemplateName, "", "", nodeMessage), node, nodePrefix, getArgs)

	node.TemplateRef = &wfv1.TemplateRef{
		Name:     nodeTemplateRefName,
		Template: nodeTemplateRefName,
	}
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t%s/%s\t%s\t%s\t%s\n", nodeTypeIconMap[wfv1.NodeTypeSuspend], nodeName, nodeTemplateRefName, nodeTemplateRefName, "", "", nodeMessage), node, nodePrefix, getArgs)

	getArgs.output = "wide"
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t%s/%s\t%s\t%s\t%s\t%s\t\n", nodeTypeIconMap[wfv1.NodeTypeSuspend], nodeName, nodeTemplateRefName, nodeTemplateRefName, "", "", getArtifactsString(node), nodeMessage), node, nodePrefix, getArgs)

	getArgs.status = "foobar"
	testPrintNodeImpl(t, "", node, nodePrefix, getArgs)
}

func TestGetCommand(t *testing.T) {
	client := mocks.Client{}
	wfClient := mocks.WorkflowServiceClient{}
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(wfWithStatus), &wf)
	assert.NoError(t, err)
	wfClient.On("GetWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&wf, nil)
	client.On("NewWorkflowServiceClient").Return(&wfClient)
	CLIOpt.client = &client
	CLIOpt.ctx = context.TODO()
	getCommand := NewGetCommand()
	getCommand.SetArgs([]string{"hello-world"})
	execFunc := func() {
		err := getCommand.Execute()
		assert.NoError(t, err)
	}
	output := CaptureOutput(execFunc)
	assert.Contains(t, output, "Name:")
	assert.Contains(t, output, "Namespace:")
	assert.Contains(t, output, "ServiceAccount:")
	assert.Contains(t, output, "Status:")
}
