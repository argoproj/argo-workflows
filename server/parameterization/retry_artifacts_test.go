package parameterization

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func artifactRetryFixture(url string) (*wfv1.Workflow, *wfv1.Workflow) {
	wf, _ := partialFixture()
	producer := &wf.Spec.Templates[2]
	producer.Metadata.Annotations[artifactChecksumContract] = `{"file":"checksum"}`
	producer.Outputs = wfv1.Outputs{Artifacts: []wfv1.Artifact{{Name: "file", Path: "/tmp/file", Archive: &wfv1.ArchiveStrategy{None: &wfv1.NoneStrategy{}}, ArtifactLocation: wfv1.ArtifactLocation{HTTP: &wfv1.HTTPArtifact{URL: url}}}}, Parameters: []wfv1.Parameter{{Name: "checksum", ValueFrom: &wfv1.ValueFrom{Path: "/tmp/checksum"}}}}
	wf.Spec.Templates[3].Inputs.Artifacts = []wfv1.Artifact{{Name: "file", Path: "/tmp/file"}}
	wf.Spec.Templates[0].DAG.Tasks[2].Arguments.Artifacts = []wfv1.Artifact{{Name: "file", From: "{{tasks.keep.outputs.artifacts.file}}"}}
	node := wf.Status.Nodes["k"]
	node.Outputs = producer.Outputs.DeepCopy()
	digest := wfv1.AnyString(fmt.Sprintf("%x", sha256.Sum256([]byte("original file"))))
	node.Outputs.Parameters[0].Value = &digest
	node.Outputs.Parameters[0].ValueFrom = nil
	wf.Status.Nodes["k"] = node
	next := wf.DeepCopy()
	value := wfv1.AnyString("new")
	next.Spec.Arguments.Parameters[0].Value = &value
	return wf, next
}

func TestPartialRetryVerifiesRealArtifactContentWithoutChangingProducer(t *testing.T) {
	body := "original file"
	status := 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status); _, _ = io.WriteString(w, body) }))
	defer server.Close()
	wf, next := artifactRetryFixture(server.URL)
	before := wf.DeepCopy()
	ctx := logging.TestContext(t.Context())
	plan, err := planPartialRetry(ctx, wf, next)
	require.NoError(t, err)
	require.Equal(t, []string{"change"}, plan.AffectedTasks)
	require.Contains(t, plan.RetainedNodes, "k")
	require.NoError(t, (Store{}).verifyRetainedArtifacts(ctx, wf, plan))
	body = "changed file"
	require.ErrorContains(t, (Store{}).verifyRetainedArtifacts(ctx, wf, plan), "content changed")
	status = 404
	require.ErrorContains(t, (Store{}).verifyRetainedArtifacts(ctx, wf, plan), "unavailable")
	require.Equal(t, before, wf)
}

func TestArtifactRetryRejectsUnprovenContracts(t *testing.T) {
	tests := map[string]func(*wfv1.Workflow){
		"missing checksum": func(w *wfv1.Workflow) { delete(w.Spec.Templates[2].Metadata.Annotations, artifactChecksumContract) },
		"unknown parameter": func(w *wfv1.Workflow) {
			w.Spec.Templates[2].Metadata.Annotations[artifactChecksumContract] = `{"file":"unknown"}`
		},
		"archive":            func(w *wfv1.Workflow) { w.Spec.Templates[2].Outputs.Artifacts[0].Archive = nil },
		"optional":           func(w *wfv1.Workflow) { w.Spec.Templates[2].Outputs.Artifacts[0].Optional = true },
		"unsupported store":  func(w *wfv1.Workflow) { w.Spec.Templates[2].Outputs.Artifacts[0].HTTP = nil },
		"missing dependency": func(w *wfv1.Workflow) { w.Spec.Templates[0].DAG.Tasks[2].Dependencies = nil },
		"dynamic artifact": func(w *wfv1.Workflow) {
			w.Spec.Templates[0].DAG.Tasks[2].Arguments.Artifacts[0].From = "{{=tasks.keep.outputs.artifacts.file}}"
		},
		"untracked source": func(w *wfv1.Workflow) {
			w.Spec.Templates[0].DAG.Tasks[2].Arguments.Artifacts[0] = wfv1.Artifact{Name: "file", ArtifactLocation: wfv1.ArtifactLocation{HTTP: &wfv1.HTTPArtifact{URL: "https://example.invalid/file"}}}
		},
		"input default": func(w *wfv1.Workflow) {
			w.Spec.Templates[3].Inputs.Artifacts[0].HTTP = &wfv1.HTTPArtifact{URL: "https://example.invalid/file"}
		},
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			wf, _ := artifactRetryFixture("https://example.invalid/file")
			change(wf)
			next := wf.DeepCopy()
			value := wfv1.AnyString("new")
			next.Spec.Arguments.Parameters[0].Value = &value
			_, err := planPartialRetry(logging.TestContext(t.Context()), wf, next)
			require.Error(t, err)
		})
	}
}

func TestArtifactRetryMissingEvidenceAndReadFailuresAreRedacted(t *testing.T) {
	for _, missing := range []string{"outputs", "value", "digest", "location", "read"} {
		t.Run(missing, func(t *testing.T) {
			wf, next := artifactRetryFixture("https://example.invalid/private")
			ctx := logging.TestContext(t.Context())
			plan, err := planPartialRetry(ctx, wf, next)
			require.NoError(t, err)
			node := wf.Status.Nodes["k"]
			switch missing {
			case "outputs":
				node.Outputs = nil
			case "value":
				node.Outputs.Parameters = nil
			case "digest":
				value := wfv1.AnyString("secret-password")
				node.Outputs.Parameters[0].Value = &value
			case "location":
				node.Outputs.Artifacts[0].HTTP = nil
			}
			wf.Status.Nodes["k"] = node
			err = verifyRetainedArtifactStreams(ctx, wf, plan, func(context.Context, *wfv1.Artifact) (io.ReadCloser, error) {
				return nil, fmt.Errorf("private-url secret-password")
			})
			require.Error(t, err)
			require.NotContains(t, err.Error(), "secret-password")
			require.NotContains(t, err.Error(), "private-url")
		})
	}
}

type repeatedBytes struct{}

func (repeatedBytes) Read(b []byte) (int, error) { clear(b); return len(b), nil }
func TestArtifactRetryBoundsAndCancellation(t *testing.T) {
	wf, next := artifactRetryFixture("https://example.invalid/file")
	ctx := logging.TestContext(t.Context())
	plan, err := planPartialRetry(ctx, wf, next)
	require.NoError(t, err)
	require.ErrorContains(t, verifyRetainedArtifactStreams(ctx, wf, plan, func(context.Context, *wfv1.Artifact) (io.ReadCloser, error) {
		return io.NopCloser(repeatedBytes{}), nil
	}), "64 MiB")
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorContains(t, verifyRetainedArtifactStreams(canceled, wf, plan, func(context.Context, *wfv1.Artifact) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("original file")), nil
	}), "verification failed")
}

func TestArtifactCredentialsRemainCallerScoped(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	client := dynamicfake.NewSimpleDynamicClient(scheme, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credential", Namespace: "other"}, Data: map[string][]byte{"password": []byte("private")}})
	resources := retryArtifactResources{Store{Namespace: "lab", Client: client}}
	_, err := resources.GetSecret(t.Context(), "credential", "password")
	require.Error(t, err)
	require.Equal(t, "lab", client.Actions()[0].GetNamespace())
	require.NotContains(t, err.Error(), "private")
}
