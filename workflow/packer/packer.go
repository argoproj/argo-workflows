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

func DecompressWorkflowTaskSetSpec(ctx context.Context, spec *wfv1.WorkflowTaskSetSpec) error {
	for name, task := range spec.Tasks {
		if task.CompressedTemplate != "" {
			content, err := file.DecodeDecompressString(ctx, task.CompressedTemplate)
			if err != nil {
				return fmt.Errorf("failed to decompress task %q: %w", name, err)
			}

			if err := json.Unmarshal([]byte(content), &task); err != nil {
				return fmt.Errorf("failed to unmarshal task %q: %w", name, err)
			}

			task.CompressedTemplate = ""
			spec.Tasks[name] = task
		}
	}
	return nil
}

func DecompressWorkflowTaskSetStatus(ctx context.Context, status *wfv1.WorkflowTaskSetStatus) error {
	for name, node := range status.Nodes {
		if node.CompressedNode != "" {
			content, err := file.DecodeDecompressString(ctx, node.CompressedNode)
			if err != nil {
				return fmt.Errorf("failed to decompress node %q: %w", name, err)
			}

			if err := json.Unmarshal([]byte(content), &node); err != nil {
				return fmt.Errorf("failed to unmarshal node %q: %w", name, err)
			}
			node.CompressedNode = ""
			status.Nodes[name] = node
		}
	}
	return nil
}

// getSize return the entire k8s resource json string size
func getSize(resource any) (int, error) {
	nodeContent, err := json.Marshal(resource)
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
const tooLargeTaskSetSpec = "workflowtasksets/spec is longer than maximum allowed size."

func IsTooLargeError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), tooLarge)
}

func IsTooLargeTaskSetSpecError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), tooLargeTaskSetSpec)
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

// CompressWorkflowTaskSetSpec compress tasks individually instead of serializing the whole map.
//
// Reason:
// WorkflowTaskSetSpec.Tasks is treated as a mergeable collection where finished tasks
// may be removed/updated via k8s merge semantics in other parts of the system.
//
// If we compressed the entire collection as a single blob:
// - we would lose per-task merge/delete semantics
// - finished task cleanup (merge-by-key behavior) would break
// - partial updates would become impossible without full decompression
//
// Therefore each task is compressed independently.
func CompressWorkflowTaskSetSpec(ctx context.Context, spec *wfv1.WorkflowTaskSetSpec) error {
	if spec == nil || len(spec.Tasks) == 0 {
		return nil
	}

	for name, task := range spec.Tasks {
		rawJSON, err := json.Marshal(task)
		if err != nil {
			return fmt.Errorf("marshal task %s: %w", name, err)
		}
		rawEncoded := string(rawJSON)
		compressed := file.CompressEncodeString(ctx, rawEncoded)

		if len(compressed) < len(rawEncoded) {
			spec.Tasks[name] = wfv1.Template{
				CompressedTemplate: compressed,
			}
		}
	}

	size, err := getSize(spec)
	if err != nil {
		return err
	}

	if size > getMaxWorkflowSize() {
		if err := DecompressWorkflowTaskSetSpec(ctx, spec); err != nil {
			return fmt.Errorf(
				"%s compressed size %d > maxSize %d; additionally failed to decompress workflow task set spec: %w",
				tooLargeTaskSetSpec,
				size,
				getMaxWorkflowSize(),
				err,
			)
		}

		return fmt.Errorf(
			"%s compressed size %d > maxSize %d",
			tooLargeTaskSetSpec,
			size,
			getMaxWorkflowSize(),
		)
	}

	return nil
}

// CompressWorkflowTaskSetStatus compress nodes individually instead of serializing the whole map.
//
// Reason:
// WorkflowTaskSetStatus.nodes is treated as a mergeable collection where finished tasks
// may be removed/updated via k8s merge semantics in other parts of the system.
//
// If we compressed the entire collection as a single blob:
// - we would lose per-task merge/delete semantics
// - finished task cleanup (merge-by-key behavior) would break
// - partial updates would become impossible without full decompression
//
// Therefore each task is compressed independently.
func CompressWorkflowTaskSetStatus(ctx context.Context, status *wfv1.WorkflowTaskSetStatus) error {
	if status == nil || len(status.Nodes) == 0 {
		return nil
	}

	for name, node := range status.Nodes {
		rawJSON, err := json.Marshal(node)
		if err != nil {
			return fmt.Errorf("marshal node %s: %w", name, err)
		}
		rawEncoded := string(rawJSON)
		compressed := file.CompressEncodeString(ctx, rawEncoded)

		if len(compressed) < len(rawEncoded) {
			status.Nodes[name] = wfv1.NodeResult{
				CompressedNode: compressed,
			}
		} else {
			node.CompressedNode = ""
			status.Nodes[name] = node
		}
	}

	size, err := getSize(status)
	if err != nil {
		return err
	}

	if size > getMaxWorkflowSize() {
		if err := DecompressWorkflowTaskSetStatus(ctx, status); err != nil {
			return fmt.Errorf(
				"%s compressed size %d > maxSize %d; additionally failed to decompress workflow task set status: %w",
				tooLargeTaskSetSpec,
				size,
				getMaxWorkflowSize(),
				err,
			)
		}
		return fmt.Errorf("%s compressed size %d > maxSize %d",
			tooLargeTaskSetSpec, size, getMaxWorkflowSize())
	}
	return nil
}
