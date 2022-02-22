package artifacts

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	testhttp "github.com/stretchr/testify/http"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	sqldbmocks "github.com/argoproj/argo-workflows/v3/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fakewfv1 "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	authmocks "github.com/argoproj/argo-workflows/v3/server/auth/mocks"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	armocks "github.com/argoproj/argo-workflows/v3/workflow/artifactrepositories/mocks"
	artifactscommon "github.com/argoproj/argo-workflows/v3/workflow/artifacts/common"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/resource"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	hydratorfake "github.com/argoproj/argo-workflows/v3/workflow/hydrator/fake"
)

func mustParse(text string) *url.URL {
	u, err := url.Parse(text)
	if err != nil {
		panic(err)
	}
	return u
}

type fakeArtifactDriver struct {
	artifactscommon.ArtifactDriver
	data []byte
}

func (a *fakeArtifactDriver) Load(_ *wfv1.Artifact, path string) error {
	return ioutil.WriteFile(path, a.data, 0o600)
}

func (a *fakeArtifactDriver) Save(_ string, _ *wfv1.Artifact) error {
	return fmt.Errorf("not implemented")
}

func newServer() *ArtifactServer {
	gatekeeper := &authmocks.Gatekeeper{}
	kube := kubefake.NewSimpleClientset()
	instanceId := "my-instanceid"
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "my-wf", Labels: map[string]string{
			common.LabelKeyControllerInstanceID: instanceId,
		}},
		Status: wfv1.WorkflowStatus{
			Nodes: wfv1.Nodes{
				"my-node": wfv1.NodeStatus{
					Inputs: &wfv1.Inputs{
						Artifacts: wfv1.Artifacts{
							{
								Name: "my-s3-input-artifact",
								ArtifactLocation: wfv1.ArtifactLocation{
									S3: &wfv1.S3Artifact{
										Key: "my-wf/my-node/my-s3-input-artifact.tgz",
									},
								},
							},
						},
					},
					Outputs: &wfv1.Outputs{
						Artifacts: wfv1.Artifacts{
							{
								Name: "my-s3-artifact",
								ArtifactLocation: wfv1.ArtifactLocation{
									S3: &wfv1.S3Artifact{
										// S3 is a configured artifact repo, so does not need key
										Key: "my-wf/my-node/my-s3-artifact.tgz",
									},
								},
							},
							{
								Name: "my-gcs-artifact",
								ArtifactLocation: wfv1.ArtifactLocation{
									GCS: &wfv1.GCSArtifact{
										// GCS is not a configured artifact repo, so must have bucket
										GCSBucket: wfv1.GCSBucket{
											Bucket: "my-bucket",
										},
										Key: "my-wf/my-node/my-gcs-artifact",
									},
								},
							},
							{
								Name: "my-oss-artifact",
								ArtifactLocation: wfv1.ArtifactLocation{
									OSS: &wfv1.OSSArtifact{
										// OSS is not a configured artifact repo, so must have bucket
										OSSBucket: wfv1.OSSBucket{
											Bucket: "my-bucket",
										},
										Key: "my-wf/my-node/my-oss-artifact.zip",
									},
								},
							},
						},
					},
				},
				// a node without input/output artifacts
				"my-node-no-artifacts": wfv1.NodeStatus{},
			},
		},
	}
	argo := fakewfv1.NewSimpleClientset(wf, &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "your-wf"},
	})
	ctx := context.WithValue(context.WithValue(context.Background(), auth.KubeKey, kube), auth.WfKey, argo)
	gatekeeper.On("ContextWithRequest", mock.Anything, mock.Anything).Return(ctx, nil)
	a := &sqldbmocks.WorkflowArchive{}
	a.On("GetWorkflow", "my-uuid").Return(wf, nil)

	fakeArtifactDriverFactory := func(_ context.Context, _ *wfv1.Artifact, _ resource.Interface) (artifactscommon.ArtifactDriver, error) {
		return &fakeArtifactDriver{data: []byte("my-data")}, nil
	}

	artifactRepositories := armocks.DummyArtifactRepositories(&wfv1.ArtifactRepository{
		S3: &wfv1.S3ArtifactRepository{
			S3Bucket: wfv1.S3Bucket{
				Endpoint: "my-endpoint",
				Bucket:   "my-bucket",
			},
		},
	})

	return newArtifactServer(gatekeeper, hydratorfake.Noop, a, instanceid.NewService(instanceId), fakeArtifactDriverFactory, artifactRepositories)
}

func TestArtifactServer_GetOutputArtifact(t *testing.T) {
	s := newServer()

	tests := []struct {
		fileName     string
		artifactName string
	}{
		{
			fileName:     "my-s3-artifact.tgz",
			artifactName: "my-s3-artifact",
		},
		{
			fileName:     "my-gcs-artifact",
			artifactName: "my-gcs-artifact",
		},
		{
			fileName:     "my-oss-artifact.zip",
			artifactName: "my-oss-artifact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.artifactName, func(t *testing.T) {
			r := &http.Request{}
			r.URL = mustParse(fmt.Sprintf("/artifacts/my-ns/my-wf/my-node/%s", tt.artifactName))
			w := &testhttp.TestResponseWriter{}
			s.GetOutputArtifact(w, r)
			if assert.Equal(t, 200, w.StatusCode) {
				assert.Equal(t, fmt.Sprintf(`filename="%s"`, tt.fileName), w.Header().Get("Content-Disposition"))
				assert.Equal(t, "my-data", w.Output)
			}
		})
	}
}

func TestArtifactServer_GetInputArtifact(t *testing.T) {
	s := newServer()

	tests := []struct {
		fileName     string
		artifactName string
	}{
		{
			fileName:     "my-s3-input-artifact.tgz",
			artifactName: "my-s3-input-artifact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.artifactName, func(t *testing.T) {
			r := &http.Request{}
			r.URL = mustParse(fmt.Sprintf("/input-artifacts/my-ns/my-wf/my-node/%s", tt.artifactName))
			w := &testhttp.TestResponseWriter{}
			s.GetInputArtifact(w, r)
			if assert.Equal(t, 200, w.StatusCode) {
				assert.Equal(t, fmt.Sprintf(`filename="%s"`, tt.fileName), w.Header().Get("Content-Disposition"))
				assert.Equal(t, "my-data", w.Output)
			}
		})
	}
}

// TestArtifactServer_NodeWithoutArtifact makes sure that the server doesn't panic due to a nil-pointer error
// when trying to get an artifact from a node result without any artifacts
func TestArtifactServer_NodeWithoutArtifact(t *testing.T) {
	s := newServer()
	r := &http.Request{}
	r.URL = mustParse(fmt.Sprintf("/input-artifacts/my-ns/my-wf/my-node-no-artifacts/%s", "my-artifact"))
	w := &testhttp.TestResponseWriter{}
	s.GetInputArtifact(w, r)
	// make sure there is no nil pointer panic
	assert.Equal(t, 500, w.StatusCode)
	s.GetOutputArtifact(w, r)
	assert.Equal(t, 500, w.StatusCode)
}

func TestArtifactServer_GetOutputArtifactWithoutInstanceID(t *testing.T) {
	s := newServer()
	r := &http.Request{}
	r.URL = mustParse("/artifacts/my-ns/your-wf/my-node/my-artifact")
	w := &testhttp.TestResponseWriter{}
	s.GetOutputArtifact(w, r)
	assert.NotEqual(t, 200, w.StatusCode)
}

func TestArtifactServer_GetOutputArtifactByUID(t *testing.T) {
	s := newServer()
	r := &http.Request{}
	r.URL = mustParse("/artifacts/my-uuid/my-node/my-artifact")
	w := &testhttp.TestResponseWriter{}
	s.GetOutputArtifactByUID(w, r)
	assert.Equal(t, 401, w.StatusCode)
}

func TestArtifactServer_GetArtifactByUIDInvalidRequestPath(t *testing.T) {
	s := newServer()
	r := &http.Request{}
	// missing my-artifact part to have a valid URL
	r.URL = mustParse("/input-artifacts/my-uuid/my-node")
	w := &testhttp.TestResponseWriter{}
	s.GetInputArtifactByUID(w, r)
	// make sure there is no index out of bounds error
	assert.Equal(t, 500, w.StatusCode)
	assert.Equal(t, "request path is not valid", w.Output)

	w = &testhttp.TestResponseWriter{}
	s.GetOutputArtifactByUID(w, r)
	assert.Equal(t, 500, w.StatusCode)
	assert.Equal(t, "request path is not valid", w.Output)
}
