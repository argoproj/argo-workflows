package parameterization

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// NativeRetryProof is set only by Argo Server's typed retry endpoint. The native
// Workflow service checks the precondition and records the receipt atomically
// with its ordinary retry update. It does not introduce another scheduler.
type NativeRetryProof struct {
	UID             string
	ResourceVersion string
	Receipt         string
	Revision        string
	RetainedNodes   map[string]string
	// NextSpec is resolved and checked by the parameter retry planner. The
	// native service applies it only after preserving the selected-node proof.
	NextSpec *wfv1.WorkflowSpec
}
type retryProofKey struct{}

func WithRetryProof(ctx context.Context, proof NativeRetryProof) context.Context {
	return context.WithValue(ctx, retryProofKey{}, proof)
}
func RetryProof(ctx context.Context) (NativeRetryProof, bool) {
	p, ok := ctx.Value(retryProofKey{}).(NativeRetryProof)
	return p, ok
}

type NativeInputProof struct {
	UID, ResourceVersion, NodeID, Receipt string
}
type inputProofKey struct{}

func WithInputProof(ctx context.Context, proof NativeInputProof) context.Context {
	return context.WithValue(ctx, inputProofKey{}, proof)
}
func InputProof(ctx context.Context) (NativeInputProof, bool) {
	p, ok := ctx.Value(inputProofKey{}).(NativeInputProof)
	return p, ok
}

// ResultFingerprint excludes graph adjacency changed by native retry. Execution
// inputs, outputs, phases and completion timestamps must remain identical.
func ResultFingerprint(node wfv1.NodeStatus) string {
	node.Children = nil
	node.OutboundNodes = nil
	return Digest(node)
}
