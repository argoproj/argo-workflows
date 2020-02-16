package commands

import (
	"bytes"
	"fmt"
	"testing"
	"text/tabwriter"
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestPrintNode
func TestPrintNode(t *testing.T) {
	var result bytes.Buffer
	w := tabwriter.NewWriter(&result, 0, 8, 1, '\t', 0)
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

	printNode(w, node, nodePrefix, getArgs)
	w.Flush()
	assert.Equal(t, fmt.Sprintf("%s %s\t%s\t%s\t%s\n", jobStatusIconMap[wfv1.NodeRunning], nodeName, nodeID, "0s", nodeMessage), result.String())
	result.Reset()

	node.TemplateName = nodeTemplateName
	printNode(w, node, nodePrefix, getArgs)
	w.Flush()
	assert.Equal(t, fmt.Sprintf("%s %s (%s)\t%s\t%s\t%s\n", jobStatusIconMap[wfv1.NodeRunning], nodeName, nodeTemplateName, nodeID, "0s", nodeMessage), result.String())
	result.Reset()

	node.TemplateName = nodeTemplateName
	printNode(w, node, nodePrefix, getArgs)
	w.Flush()
	assert.Equal(t, fmt.Sprintf("%s %s (%s)\t%s\t%s\t%s\n", jobStatusIconMap[wfv1.NodeRunning], nodeName, nodeTemplateName, nodeID, "0s", nodeMessage), result.String())
	result.Reset()

	node.Type = wfv1.NodeTypeSuspend
	printNode(w, node, nodePrefix, getArgs)
	w.Flush()
	assert.Equal(t, fmt.Sprintf("%s %s (%s)\t%s\t%s\t%s\n", nodeTypeIconMap[wfv1.NodeTypeSuspend], nodeName, nodeTemplateName, "", "", nodeMessage), result.String())
	result.Reset()

	node.TemplateRef = &wfv1.TemplateRef{
		Name:     nodeTemplateRefName,
		Template: nodeTemplateRefName,
	}
	printNode(w, node, nodePrefix, getArgs)
	w.Flush()
	assert.Equal(t, fmt.Sprintf("%s %s (%s/%s)\t%s\t%s\t%s\n", nodeTypeIconMap[wfv1.NodeTypeSuspend], nodeName, nodeTemplateRefName, nodeTemplateRefName, "", "", nodeMessage), result.String())
	result.Reset()

	getArgs.output = "wide"
	printNode(w, node, nodePrefix, getArgs)
	w.Flush()
	assert.Equal(t, fmt.Sprintf("%s %s (%s/%s)\t%s\t%s\t%s\t%s\n", nodeTypeIconMap[wfv1.NodeTypeSuspend], nodeName, nodeTemplateRefName, nodeTemplateRefName, "", "", getArtifactsString(node), nodeMessage), result.String())
	result.Reset()

	getArgs.status = "foobar"
	printNode(w, node, nodePrefix, getArgs)
	w.Flush()
	assert.Equal(t, "", result.String())
	result.Reset()
}
