package packer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/file"
)

const envVarName = "MAX_WORKFLOW_SIZE"

func getMaxWorkflowSize() int {
	s, _ := strconv.Atoi(os.Getenv(envVarName))
	if s == 0 {
		s = 1024 * 1024
	}
	return s
}

func SetMaxWorkflowSize(s int) func() {
	_ = os.Setenv(envVarName, strconv.Itoa(s))
	return func() { _ = os.Unsetenv(envVarName) }
}

func DecompressWorkflow(ctx context.Context, wf *wfv1.Workflow) error {
	if len(wf.Status.Nodes) == 0 && wf.Status.CompressedNodes != "" {
		nodeContent, err := file.DecodeDecompressString(ctx, wf.Status.CompressedNodes)
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

func IsLargeWorkflow(wf *wfv1.Workflow) (bool, error) {
	size, err := getSize(wf)
	return size > getMaxWorkflowSize(), err
}

const tooLarge = "workflow is longer than maximum allowed size."

func IsTooLargeError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), tooLarge)
}

func CompressWorkflowIfNeeded(ctx context.Context, wf *wfv1.Workflow) error {
	large, err := IsLargeWorkflow(wf)
	if err != nil {
		return err
	}
	if !large {
		return nil
	}
	return compressWorkflow(ctx, wf)
}

func compressWorkflow(ctx context.Context, wf *wfv1.Workflow) error {
	nodes := wf.Status.Nodes
	nodeContent, err := json.Marshal(nodes)
	if err != nil {
		return err
	}
	wf.Status.CompressedNodes = file.CompressEncodeString(ctx, string(nodeContent))
	wf.Status.Nodes = nil
	// still too large?
	large, err := IsLargeWorkflow(wf)
	if err != nil {
		wf.Status.CompressedNodes = ""
		wf.Status.Nodes = nodes
		return err
	}
	if large {
		compressedSize, err := getSize(wf)
		wf.Status.CompressedNodes = ""
		wf.Status.Nodes = nodes
		if err != nil {
			return err
		}
		return fmt.Errorf("%s compressed size %d > maxSize %d", tooLarge, compressedSize, getMaxWorkflowSize())
	}
	return nil
}
