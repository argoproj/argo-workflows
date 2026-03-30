package apiclient

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
)

// init probes that protobuf can marshal API messages embedding Kubernetes
// types. Kubernetes v1.35 (k8s.io/* v0.35) generates its types without
// ProtoMessage() unless the kubernetes_protomessage_one_more_release build tag
// is set (v1.36 drops the method entirely), which makes
// google.golang.org/protobuf panic deep inside its legacy bridge on the first
// real API call. Builds in this repo avoid that via the build tag (exported by
// the Makefile) or the patched vendor tree (hack/vendor-patches.sh) — but
// module consumers of this package get neither by default. Probing here turns
// an obscure runtime panic into an immediate, actionable one at startup.
func init() {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf(
				"pkg/apiclient cannot marshal Kubernetes types (%v); "+
					"build with -tags=kubernetes_protomessage_one_more_release (Kubernetes v1.35 only) "+
					"or use the argo-workflows patched vendor tree (`make vendor`) — see docs/upgrading.md",
				r))
		}
	}()
	_, _ = proto.Marshal(&workflowpkg.WorkflowGetRequest{GetOptions: &metav1.GetOptions{}})
}
