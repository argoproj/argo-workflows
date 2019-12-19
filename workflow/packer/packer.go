package packer

import (
	"encoding/json"
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util/file"
)

//maxWorkflowSize is the maximum  size for workflow.yaml
const defaultMaxWorkflowSize = 1024 * 1024

var maxWorkflowSize = defaultMaxWorkflowSize

func DecompressWorkflow(wf *wfv1.Workflow) (*wfv1.Workflow, error) {
	if len(wf.Status.Nodes) == 0 && wf.Status.CompressedNodes != "" {
		nodeContent, err := file.DecodeDecompressString(wf.Status.CompressedNodes)
		if err != nil {
			return nil, err
		}
		decompressedWf := wf.DeepCopy()
		err = json.Unmarshal([]byte(nodeContent), &decompressedWf.Status.Nodes)
		decompressedWf.Status.CompressedNodes = ""
		return decompressedWf, err
	}
	return wf, nil
}

// getSize return the entire workflow json string size
func getSize(wf *wfv1.Workflow) (int, error) {
	nodeContent, err := json.Marshal(wf)
	if err != nil {
		return 0, err
	}
	return len(nodeContent), nil
}

func IsLargeWorkflow(wf *wfv1.Workflow) (bool, error) {
	size, err := getSize(wf)
	return size > maxWorkflowSize, err
}

const tooLarge = "workflow is longer than maximum allowed size."

func IsTooLargeError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), tooLarge)
}

func CompressWorkflow(wf *wfv1.Workflow) (*wfv1.Workflow, error) {
	large, err := IsLargeWorkflow(wf)
	if err != nil {
		return nil, err
	}
	if !large {
		return wf, nil
	}
	nodeContent, err := json.Marshal(wf.Status.Nodes)
	if err != nil {
		return nil, err
	}
	compressedWf := wf.DeepCopy()
	compressedWf.Status.CompressedNodes = file.CompressEncodeString(string(nodeContent))
	compressedWf.Status.Nodes = nil

	// still too large?
	large, err = IsLargeWorkflow(compressedWf)
	if err != nil {
		return nil, err
	}
	if large {
		compressedSize, _ := getSize(compressedWf)
		return nil, fmt.Errorf("%s compressed size %d > maxSize %d", tooLarge, compressedSize, maxWorkflowSize)
	}
	return compressedWf, nil
}
