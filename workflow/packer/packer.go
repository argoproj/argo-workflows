package packer

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/file"
)

const envVarMaxWorkflowSize = "MAX_WORKFLOW_SIZE"
const envVarMaxNodeStatusSize = "MAX_NODE_STATUS_SIZE"

func getMaxWorkflowSize() int {
	s, _ := strconv.Atoi(os.Getenv(envVarMaxWorkflowSize))
	if s == 0 {
		s = 1024 * 1024
	}
	return s
}

func getMaxNodeStatusSize() int {
	s, _ := strconv.Atoi(os.Getenv(envVarMaxNodeStatusSize))
	if s == 0 {
		s = 1024 * 1024
	}
	return s
}

func SetMaxWorkflowSize(s int) func() {
	_ = os.Setenv(envVarMaxWorkflowSize, strconv.Itoa(s))
	return func() { _ = os.Unsetenv(envVarMaxWorkflowSize) }
}

func DecompressWorkflow(wf *wfv1.Workflow) error {
	if len(wf.Status.Nodes) == 0 && wf.Status.CompressedNodes != "" {
		nodeContent, err := file.DecodeDecompressString(wf.Status.CompressedNodes)
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(nodeContent), &wf.Status.Nodes)
		wf.Status.CompressedNodes = ""
		return err
	}
	return nil
}

// getSize return the entire workflow json string size
func getSize(wf *wfv1.Workflow) (int, error) {
	nodeContent, err := json.Marshal(wf)
	if err != nil {
		return 0, err
	}
	return len(nodeContent), nil
}

// getNodeStatusSize return the workflow node status json string size
func getNodeStatusSize(wf *wfv1.Workflow) (int, error) {
	nodeContent, err := json.Marshal(wf.Status.Nodes)
	if err != nil {
		return 0, err
	}
	return len(nodeContent) + len(wf.Status.CompressedNodes), nil
}

func IsLargeWorkflow(wf *wfv1.Workflow) (bool, error) {
	nodesSize, err := getNodeStatusSize(wf)
	if nodesSize > getMaxNodeStatusSize() {
		return true, err
	}
	size, err := getSize(wf)
	return size > getMaxWorkflowSize(), err
}

const tooLarge = "workflow is longer than maximum allowed size."

func IsTooLargeError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), tooLarge)
}

func CompressWorkflowIfNeeded(wf *wfv1.Workflow) error {
	large, err := IsLargeWorkflow(wf)
	if err != nil {
		return err
	}
	if !large {
		return nil
	}
	return compressWorkflow(wf)
}

func compressWorkflow(wf *wfv1.Workflow) error {
	nodes := wf.Status.Nodes
	nodeContent, err := json.Marshal(nodes)
	if err != nil {
		return err
	}
	wf.Status.CompressedNodes = file.CompressEncodeString(string(nodeContent))
	wf.Status.Nodes = nil
	// still too large?
	large, err := IsLargeWorkflow(wf)
	if err != nil {
		wf.Status.CompressedNodes = ""
		wf.Status.Nodes = nodes
		return err
	}
	if large {
		compressedSize, err1 := getSize(wf)
		nodesSize, err2 := getNodeStatusSize(wf)
		wf.Status.CompressedNodes = ""
		wf.Status.Nodes = nodes
		if err1 != nil {
			return err
		}
		if err2 != nil {
			return err2
		}
		return fmt.Errorf("%s compressed size %d > maxSize %d or node status size %d > maxNodeStatusSize %d",
			tooLarge, compressedSize, getMaxWorkflowSize(), nodesSize, getMaxNodeStatusSize())
	}
	return nil
}
