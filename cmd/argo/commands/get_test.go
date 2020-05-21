package commands

import (
	"bytes"
	"fmt"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

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
	kubernetesNodeName := "testKnodeName"
	nodePrefix := ""
	nodeTemplateName := "testTemplate"
	nodeTemplateRefName := "testTemplateRef"
	nodeID := "testID"
	nodeMessage := "test"
	getArgs := getFlags{
		output: "",
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
	node.HostNodeName = kubernetesNodeName
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t%s\t%s\t%s\t%s\t%s\n", jobStatusIconMap[wfv1.NodeRunning], nodeName, "", nodeID, "0s", nodeMessage, ""), node, nodePrefix, getArgs)

	// Compatibility test
	getArgs.status = "Running"
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t\t%s\t%s\t%s\t\n", jobStatusIconMap[wfv1.NodeRunning], nodeName, nodeID, "0s", nodeMessage), node, nodePrefix, getArgs)

	getArgs.status = ""
	getArgs.nodeFieldSelectorString = "phase=Running"
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t\t%s\t%s\t%s\t\n", jobStatusIconMap[wfv1.NodeRunning], nodeName, nodeID, "0s", nodeMessage), node, nodePrefix, getArgs)

	getArgs.nodeFieldSelectorString = "phase!=foobar"
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t\t%s\t%s\t%s\t\n", jobStatusIconMap[wfv1.NodeRunning], nodeName, nodeID, "0s", nodeMessage), node, nodePrefix, getArgs)

	getArgs.nodeFieldSelectorString = "phase!=Running"
	testPrintNodeImpl(t, "", node, nodePrefix, getArgs)

	// Compatibility test
	getArgs.nodeFieldSelectorString = ""
	getArgs.status = "foobar"
	testPrintNodeImpl(t, "", node, nodePrefix, getArgs)

	getArgs.status = ""
	getArgs.nodeFieldSelectorString = "phase=foobar"
	testPrintNodeImpl(t, "", node, nodePrefix, getArgs)

	getArgs = getFlags{
		output: "",
	}

	node.TemplateName = nodeTemplateName
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t%s\t%s\t%s\t%s\t%s\n", jobStatusIconMap[wfv1.NodeRunning], nodeName, nodeTemplateName, nodeID, "0s", nodeMessage, ""), node, nodePrefix, getArgs)

	node.Type = wfv1.NodeTypeSuspend
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t%s\t%s\t%s\t%s\t%s\n", nodeTypeIconMap[wfv1.NodeTypeSuspend], nodeName, nodeTemplateName, "", "", nodeMessage, ""), node, nodePrefix, getArgs)

	node.TemplateRef = &wfv1.TemplateRef{
		Name:     nodeTemplateRefName,
		Template: nodeTemplateRefName,
	}
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t%s/%s\t%s\t%s\t%s\t%s\n", nodeTypeIconMap[wfv1.NodeTypeSuspend], nodeName, nodeTemplateRefName, nodeTemplateRefName, "", "", nodeMessage, ""), node, nodePrefix, getArgs)

	getArgs.output = "wide"
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t%s/%s\t%s\t%s\t%s\t%s\t%s\t\n", nodeTypeIconMap[wfv1.NodeTypeSuspend], nodeName, nodeTemplateRefName, nodeTemplateRefName, "", "", getArtifactsString(node), nodeMessage, ""), node, nodePrefix, getArgs)

	node.Type = wfv1.NodeTypePod
	testPrintNodeImpl(t, fmt.Sprintf("%s %s\t%s/%s\t%s\t%s\t%s\t%s\t%s\t%s\n", jobStatusIconMap[wfv1.NodeRunning], nodeName, nodeTemplateRefName, nodeTemplateRefName, nodeID, "0s", getArtifactsString(node), nodeMessage, "", kubernetesNodeName), node, nodePrefix, getArgs)

	getArgs.status = "foobar"
	testPrintNodeImpl(t, "", node, nodePrefix, getArgs)
}

func TestStatusToNodeFieldSelector(t *testing.T) {
	one := statusToNodeFieldSelector("Running")
	assert.Equal(t, "phase=Running", one)
}
