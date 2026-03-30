package apiclient

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
)

// The package-level init probe already ran by the time this test executes; this
// pins that marshalling messages carrying Kubernetes types works (via the build
// tag or the patched vendor tree — see protocompat.go).
func TestProtoCompat_MarshalKubernetesTypes(t *testing.T) {
	data, err := proto.Marshal(&workflowpkg.WorkflowGetRequest{
		Name:       "wf",
		Namespace:  "ns",
		GetOptions: &metav1.GetOptions{},
	})
	require.NoError(t, err)
	require.NotEmpty(t, data)
}
