package artifacts

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	apierr "k8s.io/apimachinery/pkg/api/errors"

	argoerrors "github.com/argoproj/argo-workflows/v3/errors"

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
	return os.WriteFile(path, a.data, 0o600)
}

var bucketsOfKeys = map[string][]string{
	"my-bucket": {
		"my-wf/my-node-1/my-s3-input-artifact.tgz",
		"my-wf/my-node-1/my-s3-artifact-directory",
		"my-wf/my-node-1/my-s3-artifact-directory/a.txt",
		"my-wf/my-node-1/my-s3-artifact-directory/subdirectory/b.txt",
		"my-wf/my-node-1/my-gcs-artifact",
		"my-wf/my-node-1/my-gcs-artifact.tgz",
		"my-wf/my-node-1/my-oss-artifact.zip",
		"my-wf/my-node-1/my-s3-artifact.tgz",
		"my-wf/my-node-inline/main.log",
	},
	"my-bucket-2": {
		"my-wf/my-node-2/my-s3-artifact-bucket-2",
	},
	"my-bucket-3": {
		"my-wf/my-node-2/my-s3-artifact-bucket-3",
	},
	"my-bucket-4": {
		"my-wf/my-node-3/my-s3-artifact.tgz",
	},
}

func (a *fakeArtifactDriver) OpenStream(artifact *wfv1.Artifact) (io.ReadCloser, error) {
	//fmt.Printf("deletethis: artifact=%+v\n", artifact)

	key, err := artifact.ArtifactLocation.GetKey()
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(key, "deletedFile.txt") {
		return nil, argoerrors.New(argoerrors.CodeNotFound, "file deleted")
	} else if strings.HasSuffix(key, "somethingElseWentWrong.txt") {
		return nil, errors.New("whatever")
	}

	if artifact.S3 != nil {
		// make sure it's a recognizable bucket/key
		keysInBucket, found := bucketsOfKeys[artifact.S3.Bucket]
		if !found {
			return nil, fmt.Errorf("artifact bucket not found: %+v", artifact)
		}
		foundKey := false
		for _, recognizableKey := range keysInBucket {
			if key == recognizableKey {
				foundKey = true
				break
			}
		}
		if !foundKey {
			return nil, fmt.Errorf("artifact key '%s' not found in bucket '%s'", key, artifact.S3.Bucket)
		}
	}

	return io.NopCloser(bytes.NewReader(a.data)), nil
}

func (a *fakeArtifactDriver) Save(_ string, _ *wfv1.Artifact) error {
	return fmt.Errorf("not implemented")
}

func (a *fakeArtifactDriver) IsDirectory(artifact *wfv1.Artifact) (bool, error) {
	key, err := artifact.GetKey()
	if err != nil {
		return false, err
	}

	if strings.HasSuffix(key, "my-gcs-artifact.tgz") {
		return false, argoerrors.New(argoerrors.CodeNotImplemented, "IsDirectory currently unimplemented for GCS")
	}

	return strings.HasSuffix(key, "my-s3-artifact-directory") || strings.HasSuffix(key, "my-s3-artifact-directory/"), nil
}

func (a *fakeArtifactDriver) ListObjects(artifact *wfv1.Artifact) ([]string, error) {
	key, err := artifact.GetKey()
	if err != nil {
		return nil, err
	}
	if artifact.Name == "my-s3-artifact-directory" {
		if strings.HasSuffix(key, "subdirectory") {
			return []string{
				"my-wf/my-node-1/my-s3-artifact-directory/subdirectory/b.txt",
				"my-wf/my-node-1/my-s3-artifact-directory/subdirectory/c.txt",
			}, nil
		} else {
			return []string{
				"my-wf/my-node-1/my-s3-artifact-directory/a.txt",
				"my-wf/my-node-1/my-s3-artifact-directory/index.html",
				"my-wf/my-node-1/my-s3-artifact-directory/subdirectory/b.txt",
				"my-wf/my-node-1/my-s3-artifact-directory/subdirectory/c.txt",
			}, nil
		}
	}
	return []string{}, nil
}

func newServer() *ArtifactServer {
	gatekeeper := &authmocks.Gatekeeper{}
	kube := kubefake.NewSimpleClientset()
	instanceId := "my-instanceid"
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "my-wf", Labels: map[string]string{
			common.LabelKeyControllerInstanceID: instanceId,
		}},
		Spec: wfv1.WorkflowSpec{
			Templates: []wfv1.Template{
				{
					Name: "template-1",
				},
				{
					Name: "template-2",
					ArchiveLocation: &wfv1.ArtifactLocation{
						S3: &wfv1.S3Artifact{
							Key: "key-1",
							S3Bucket: wfv1.S3Bucket{
								Bucket:   "my-bucket-3",
								Endpoint: "minio:9000",
							},
						},
					},
				},
			},
		},
		Status: wfv1.WorkflowStatus{
			Nodes: wfv1.Nodes{
				"my-node-1": wfv1.NodeStatus{
					TemplateName: "template-1",
					Inputs: &wfv1.Inputs{
						Artifacts: wfv1.Artifacts{
							{
								Name: "my-s3-input-artifact",
								ArtifactLocation: wfv1.ArtifactLocation{
									S3: &wfv1.S3Artifact{
										Key: "my-wf/my-node-1/my-s3-input-artifact.tgz",
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
										Key: "my-wf/my-node-1/my-s3-artifact.tgz",
									},
								},
							},
							{
								Name: "my-s3-artifact-directory",
								ArtifactLocation: wfv1.ArtifactLocation{
									S3: &wfv1.S3Artifact{
										// S3 is a configured artifact repo, so does not need key
										Key: "my-wf/my-node-1/my-s3-artifact-directory",
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
										Key: "my-wf/my-node-1/my-gcs-artifact",
									},
								},
							},
							{
								Name: "my-gcs-artifact-file",
								ArtifactLocation: wfv1.ArtifactLocation{
									GCS: &wfv1.GCSArtifact{
										// GCS is not a configured artifact repo, so must have bucket
										GCSBucket: wfv1.GCSBucket{
											Bucket: "my-bucket",
										},
										Key: "my-wf/my-node-1/my-gcs-artifact.tgz",
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
										Key: "my-wf/my-node-1/my-oss-artifact.zip",
									},
								},
							},
						},
					},
				},

				"my-node-2": wfv1.NodeStatus{
					TemplateName: "template-2",
					Outputs: &wfv1.Outputs{
						Artifacts: wfv1.Artifacts{
							{
								Name: "my-s3-artifact-bucket-3",
								ArtifactLocation: wfv1.ArtifactLocation{
									S3: &wfv1.S3Artifact{
										// S3 is a configured artifact repo, so does not need key
										Key: "my-wf/my-node-2/my-s3-artifact-bucket-3",
									},
								},
							},
							{
								Name: "my-s3-artifact-bucket-2",
								ArtifactLocation: wfv1.ArtifactLocation{
									S3: &wfv1.S3Artifact{
										// S3 is a configured artifact repo, so does not need key
										Key: "my-wf/my-node-2/my-s3-artifact-bucket-2",
										S3Bucket: wfv1.S3Bucket{
											Bucket:   "my-bucket-2",
											Endpoint: "minio:9000",
										},
									},
								},
							},
						},
					},
				},

				"my-node-3": wfv1.NodeStatus{
					TemplateRef: &wfv1.TemplateRef{
						Name:     "my-template",
						Template: "template-3",
					},
					Outputs: &wfv1.Outputs{
						Artifacts: wfv1.Artifacts{
							{
								Name: "my-s3-artifact",
								ArtifactLocation: wfv1.ArtifactLocation{
									S3: &wfv1.S3Artifact{
										// S3 is a configured artifact repo, so does not need key
										Key: "my-wf/my-node-3/my-s3-artifact.tgz",
										S3Bucket: wfv1.S3Bucket{
											Bucket:   "my-bucket-4",
											Endpoint: "minio:9000",
										},
									},
								},
							},
						},
					},
				},
				"my-node-inline": wfv1.NodeStatus{
					TemplateName: "",
					Outputs: &wfv1.Outputs{
						Artifacts: wfv1.Artifacts{
							{
								Name: "my-s3-artifact-inline",
								ArtifactLocation: wfv1.ArtifactLocation{
									S3: &wfv1.S3Artifact{
										// S3 is a configured artifact repo, so does not need key
										Key: "my-wf/my-node-inline/main.log",
									},
								},
							},
						},
					},
				},
				// a node without input/output artifacts
				"my-node-no-artifacts": wfv1.NodeStatus{},
			},
			StoredTemplates: map[string]wfv1.Template{
				"namespaced/my-template/template-3": {
					Name: "template-3",
					Outputs: wfv1.Outputs{
						Artifacts: wfv1.Artifacts{
							{
								Name: "my-s3-artifact",
								Path: "my-s3-artifact.tgz",
							},
						},
					},
				},
			},
		},
	}
	argo := fakewfv1.NewSimpleClientset(wf, &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "your-wf"},
	})
	ctx := context.WithValue(context.WithValue(context.Background(), auth.KubeKey, kube), auth.WfKey, argo)
	gatekeeper.On("ContextWithRequest", mock.Anything, mock.Anything).Return(ctx, nil)
	a := &sqldbmocks.WorkflowArchive{}
	a.On("GetWorkflow", "my-uuid", "", "").Return(wf, nil)

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

func TestArtifactServer_GetArtifactFile(t *testing.T) {
	s := newServer()

	tests := []struct {
		path string
		// expected results:
		statusCode int
		//redirect       bool
		location string
		//success        bool
		isDirectory    bool
		directoryFiles []string // verify these files are in there, if this is a directory
	}{
		{
			path:       "/artifact-files/my-ns/workflows/my-wf/my-node-1/outputs/my-s3-artifact-directory",
			statusCode: 307, // redirect
			location:   "/artifact-files/my-ns/workflows/my-wf/my-node-1/outputs/my-s3-artifact-directory/",
		},
		{
			path:       "/artifact-files/my-ns/workflows/my-wf/my-node-1/outputs/my-s3-artifact-directory/",
			statusCode: 307, // redirect
			location:   "/artifact-files/my-ns/workflows/my-wf/my-node-1/outputs/my-s3-artifact-directory/index.html",
		},
		{
			path:        "/artifact-files/my-ns/workflows/my-wf/my-node-1/outputs/my-s3-artifact-directory/subdirectory/",
			statusCode:  200,
			isDirectory: true,
			directoryFiles: []string{
				"..",
				"b.txt",
				"c.txt",
			},
		},
		{
			path:        "/artifact-files/my-ns/workflows/my-wf/my-node-1/outputs/my-s3-artifact-directory/a.txt",
			statusCode:  200,
			isDirectory: false,
		},
		{
			path:        "/artifact-files/my-ns/workflows/my-wf/my-node-1/outputs/my-s3-artifact-directory/subdirectory/b.txt",
			statusCode:  200,
			isDirectory: false,
		},
		{
			path:        "/artifact-files/my-ns/workflows/my-wf/my-node-1/outputs/my-s3-artifact-directory/deletedFile.txt",
			statusCode:  404,
			isDirectory: false,
		},
		{
			path:        "/artifact-files/my-ns/workflows/my-wf/my-node-1/outputs/my-s3-artifact-directory/somethingElseWentWrong.txt",
			statusCode:  500,
			isDirectory: false,
		},
		{
			path:        "/artifact-files/my-ns/workflows/my-wf/my-node-1/outputs/my-gcs-artifact-file/my-gcs-artifact.tgz",
			statusCode:  200,
			isDirectory: false,
		},
		{
			path:        "/artifact-files/my-ns/workflows/my-wf/my-node-2/outputs/my-s3-artifact-bucket-3",
			statusCode:  200,
			isDirectory: false,
		},
		{
			path:        "/artifact-files/my-ns/workflows/my-wf/my-node-2/outputs/my-s3-artifact-bucket-2",
			statusCode:  200,
			isDirectory: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			r := &http.Request{}
			r.URL = mustParse(tt.path)
			recorder := httptest.NewRecorder()

			s.GetArtifactFile(recorder, r)
			assert.Equal(t, tt.statusCode, recorder.Result().StatusCode)
			if tt.statusCode >= 300 && tt.statusCode <= 399 { // redirect
				assert.Equal(t, tt.location, recorder.Header().Get("Location"))
			} else if tt.statusCode >= 200 && tt.statusCode <= 299 { // success
				all, err := io.ReadAll(recorder.Result().Body)
				if err != nil {
					panic(fmt.Sprintf("failed to read http body: %v", err))
				}
				if tt.isDirectory {
					fmt.Printf("got directory listing:\n%s\n", all)
					// verify that the files are contained in the listing we got back
					assert.Equal(t, len(tt.directoryFiles), strings.Count(string(all), "<li>"))
					for _, file := range tt.directoryFiles {
						assert.True(t, strings.Contains(string(all), file))
					}
				} else {
					assert.Equal(t, "my-data", string(all))
				}

			}
		})
	}
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
			r.URL = mustParse(fmt.Sprintf("/artifacts/my-ns/my-wf/my-node-1/%s", tt.artifactName))
			recorder := httptest.NewRecorder()

			s.GetOutputArtifact(recorder, r)
			if assert.Equal(t, 200, recorder.Result().StatusCode) {
				assert.Equal(t, fmt.Sprintf(`filename="%s"`, tt.fileName), recorder.Header().Get("Content-Disposition"))
				all, err := io.ReadAll(recorder.Result().Body)
				if err != nil {
					panic(fmt.Sprintf("failed to read http body: %v", err))
				}
				assert.Equal(t, "my-data", string(all))
			}
		})
	}
}

func TestArtifactServer_GetOutputArtifactWithTemplate(t *testing.T) {
	s := newServer()

	tests := []struct {
		fileName     string
		artifactName string
	}{
		{
			fileName:     "my-s3-artifact.tgz",
			artifactName: "my-s3-artifact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.artifactName, func(t *testing.T) {
			r := &http.Request{}
			r.URL = mustParse(fmt.Sprintf("/artifacts/my-ns/my-wf/my-node-3/%s", tt.artifactName))
			recorder := httptest.NewRecorder()

			s.GetOutputArtifact(recorder, r)
			if assert.Equal(t, 200, recorder.Result().StatusCode) {
				assert.Equal(t, fmt.Sprintf(`filename="%s"`, tt.fileName), recorder.Header().Get("Content-Disposition"))
				all, err := io.ReadAll(recorder.Result().Body)
				if err != nil {
					panic(fmt.Sprintf("failed to read http body: %v", err))
				}
				assert.Equal(t, "my-data", string(all))
			}
		})
	}
}

func TestArtifactServer_GetOutputArtifactWithInlineTemplate(t *testing.T) {
	s := newServer()

	tests := []struct {
		fileName     string
		artifactName string
	}{
		{
			fileName:     "main.log",
			artifactName: "my-s3-artifact-inline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.artifactName, func(t *testing.T) {
			r := &http.Request{}
			r.URL = mustParse(fmt.Sprintf("/artifacts/my-ns/my-wf/my-node-inline/%s", tt.artifactName))
			recorder := httptest.NewRecorder()

			s.GetOutputArtifact(recorder, r)
			if assert.Equal(t, 200, recorder.Result().StatusCode) {
				assert.Equal(t, fmt.Sprintf(`filename="%s"`, tt.fileName), recorder.Header().Get("Content-Disposition"))
				all, err := io.ReadAll(recorder.Result().Body)
				if err != nil {
					panic(fmt.Sprintf("failed to read http body: %v", err))
				}
				assert.Equal(t, "my-data", string(all))
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
			r.URL = mustParse(fmt.Sprintf("/input-artifacts/my-ns/my-wf/my-node-1/%s", tt.artifactName))
			recorder := httptest.NewRecorder()
			s.GetInputArtifact(recorder, r)
			if assert.Equal(t, 200, recorder.Result().StatusCode) {
				assert.Equal(t, fmt.Sprintf(`filename="%s"`, tt.fileName), recorder.Result().Header.Get("Content-Disposition"))
				all, err := io.ReadAll(recorder.Result().Body)
				if err != nil {
					panic(fmt.Sprintf("failed to read http body: %v", err))
				}
				assert.Equal(t, "my-data", string(all))
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
	r.URL = mustParse("/artifacts/my-ns/your-wf/my-node-1/my-artifact")
	w := &testhttp.TestResponseWriter{}
	s.GetOutputArtifact(w, r)
	assert.NotEqual(t, 200, w.StatusCode)
}

func TestArtifactServer_GetOutputArtifactByUID(t *testing.T) {
	s := newServer()
	r := &http.Request{}
	r.URL = mustParse("/artifacts/my-uuid/my-node-1/my-artifact")
	w := &testhttp.TestResponseWriter{}
	s.GetOutputArtifactByUID(w, r)
	assert.Equal(t, 401, w.StatusCode)
}

func TestArtifactServer_GetArtifactByUIDInvalidRequestPath(t *testing.T) {
	s := newServer()
	r := &http.Request{}
	// missing my-artifact part to have a valid URL
	r.URL = mustParse("/input-artifacts/my-uuid/my-node-1")
	w := &testhttp.TestResponseWriter{}
	s.GetInputArtifactByUID(w, r)
	// make sure there is no index out of bounds error
	assert.Equal(t, 400, w.StatusCode)
	assert.Contains(t, w.Output, "Bad Request")

	w = &testhttp.TestResponseWriter{}
	s.GetOutputArtifactByUID(w, r)
	assert.Equal(t, 400, w.StatusCode)
	assert.Contains(t, w.Output, "Bad Request")
}

func TestArtifactServer_httpBadRequestError(t *testing.T) {
	s := newServer()
	w := &testhttp.TestResponseWriter{}
	s.httpBadRequestError(w)

	assert.Equal(t, http.StatusBadRequest, w.StatusCode)
	assert.Contains(t, w.Output, "Bad Request")
}

func TestArtifactServer_httpFromError(t *testing.T) {
	s := newServer()
	w := &testhttp.TestResponseWriter{}
	err := errors.New("math: square root of negative number")

	s.httpFromError(err, w)

	assert.Equal(t, http.StatusInternalServerError, w.StatusCode)
	assert.Equal(t, "Internal Server Error\n", w.Output)

	w = &testhttp.TestResponseWriter{}
	err = apierr.NewUnauthorized("")

	s.httpFromError(err, w)

	assert.Equal(t, http.StatusUnauthorized, w.StatusCode)
	assert.Contains(t, w.Output, "Unauthorized")

	w = &testhttp.TestResponseWriter{}
	err = argoerrors.New(argoerrors.CodeNotFound, "not found")

	s.httpFromError(err, w)
	assert.Equal(t, http.StatusNotFound, w.StatusCode)
}
