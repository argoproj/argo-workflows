package artifacts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"

	argoerrors "github.com/argoproj/argo-workflows/v4/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"

	ktesting "k8s.io/client-go/testing"

	servertypes "github.com/argoproj/argo-workflows/v4/server/types"

	sqldbmocks "github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfv1 "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	authmocks "github.com/argoproj/argo-workflows/v4/server/auth/mocks"
	"github.com/argoproj/argo-workflows/v4/util/instanceid"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	armocks "github.com/argoproj/argo-workflows/v4/workflow/artifactrepositories/mocks"
	artifactscommon "github.com/argoproj/argo-workflows/v4/workflow/artifacts/common"
	"github.com/argoproj/argo-workflows/v4/workflow/artifacts/resource"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	hydratorfake "github.com/argoproj/argo-workflows/v4/workflow/hydrator/fake"
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
	data            []byte
	saveStreamError error
}

func (a *fakeArtifactDriver) Load(_ context.Context, _ *wfv1.Artifact, path string) error {
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

func (a *fakeArtifactDriver) OpenStream(_ context.Context, artifact *wfv1.Artifact) (io.ReadCloser, error) {
	// fmt.Printf("deletethis: artifact=%+v\n", artifact)

	key, err := artifact.GetKey()
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
		foundKey := slices.Contains(keysInBucket, key)
		if !foundKey {
			return nil, fmt.Errorf("artifact key '%s' not found in bucket '%s'", key, artifact.S3.Bucket)
		}
	}

	return io.NopCloser(bytes.NewReader(a.data)), nil
}

func (a *fakeArtifactDriver) Save(_ context.Context, _ string, _ *wfv1.Artifact) error {
	return fmt.Errorf("not implemented")
}

func (a *fakeArtifactDriver) SaveStream(_ context.Context, _ io.Reader, _ *wfv1.Artifact) error {
	return a.saveStreamError
}

func (a *fakeArtifactDriver) IsDirectory(_ context.Context, artifact *wfv1.Artifact) (bool, error) {
	key, err := artifact.GetKey()
	if err != nil {
		return false, err
	}

	if strings.HasSuffix(key, "my-gcs-artifact.tgz") {
		return false, argoerrors.New(argoerrors.CodeNotImplemented, "IsDirectory currently unimplemented for GCS")
	}

	return strings.HasSuffix(key, "my-s3-artifact-directory") || strings.HasSuffix(key, "my-s3-artifact-directory/"), nil
}

func (a *fakeArtifactDriver) ListObjects(_ context.Context, artifact *wfv1.Artifact) ([]string, error) {
	key, err := artifact.GetKey()
	if err != nil {
		return nil, err
	}
	if artifact.Name == "my-s3-artifact-directory" {
		prefix := "my-wf/my-node-1/my-s3-artifact-directory"
		subdir := []string{
			prefix + "/subdirectory/b.txt",
			prefix + "/subdirectory/c.txt",
		}
		// XSS test strings. Loosely adapted from https://cheatsheetseries.owasp.org/cheatsheets/XSS_Filter_Evasion_Cheat_Sheet.html#waf-bypass-strings-for-xss
		xss := []string{
			prefix + `/xss/xss\"><img src=x onerror="alert(document.domain)">.html`,
			prefix + `/xss/javascript:alert(document.domain)`,
			prefix + `/xss/javascript:\u0061lert(1)`,
			prefix + `/xss/<Input value = "XSS" type = text>`,
		}
		switch {
		case strings.HasSuffix(key, "subdirectory"):
			return subdir, nil
		case strings.HasSuffix(key, "xss"):
			return xss, nil
		default:
			return append(append([]string{
				prefix + "/a.txt",
				prefix + "/index.html",
			}, subdir...), xss...), nil
		}
	}
	return []string{}, nil
}

func newServer(t *testing.T) *ArtifactServer {
	t.Helper()
	gatekeeper := &authmocks.Gatekeeper{}
	kube := kubefake.NewClientset()
	instanceID := "my-instanceid"
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "my-wf", Labels: map[string]string{
			common.LabelKeyControllerInstanceID: instanceID,
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
							{
								Name: "my-s3-artifact-directory",
								ArtifactLocation: wfv1.ArtifactLocation{
									S3: &wfv1.S3Artifact{
										Key: "my-wf/my-node-1/my-s3-artifact-directory",
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
	argo := fakewfv1.NewClientset(wf, &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "your-wf"},
	})
	ctx := context.WithValue(context.WithValue(logging.TestContext(t.Context()), auth.KubeKey, kube), auth.WfKey, argo)
	gatekeeper.On("ContextWithRequest", mock.Anything, mock.Anything).Return(ctx, nil)
	a := &sqldbmocks.WorkflowArchive{}
	a.On("GetWorkflow", mock.Anything, "my-uuid", "", "").Return(wf, nil)

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

	return newArtifactServer(gatekeeper, hydratorfake.Noop, a, instanceid.NewService(instanceID), fakeArtifactDriverFactory, artifactRepositories, logging.RequireLoggerFromContext(ctx))
}

func TestArtifactServer_GetArtifactFile(t *testing.T) {
	s := newServer(t)

	tests := []struct {
		path string
		// expected results:
		statusCode int
		// redirect       bool
		location string
		// success        bool
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
				`<a href="..">..</a>`,
				`<a href="./b.txt">b.txt</a>`,
				`<a href="./c.txt">c.txt</a>`,
			},
		},
		{
			path:        "/artifact-files/my-ns/workflows/my-wf/my-node-1/outputs/my-s3-artifact-directory/xss/",
			statusCode:  200,
			isDirectory: true,
			directoryFiles: []string{
				`<a href="..">..</a>`,
				`<a href="./xss%5c%22%3e%3cimg%20src=x%20onerror=%22alert%28document.domain%29%22%3e.html">xss\&#34;&gt;&lt;img src=x onerror=&#34;alert(document.domain)&#34;&gt;.html</a>`,
				`<a href="./javascript:alert%28document.domain%29">javascript:alert(document.domain)</a></li>`,
				`<a href="./javascript:%5cu0061lert%281%29">javascript:\u0061lert(1)</a>`,
				`<a href="./%3cInput%20value%20=%20%22XSS%22%20type%20=%20text%3e">&lt;Input value = &#34;XSS&#34; type = text&gt;</a>`,
			},
		},
		{
			path:        "/artifact-files/my-ns/workflows/my-wf/my-node-1/inputs/my-s3-input-artifact",
			statusCode:  200,
			isDirectory: false,
		},
		{
			path:        "/artifact-files/my-ns/workflows/my-wf/my-node-1/inputs/my-s3-artifact-directory/subdirectory/",
			statusCode:  200,
			isDirectory: true,
			directoryFiles: []string{
				`<a href="..">..</a>`,
				`<a href="./b.txt">b.txt</a>`,
				`<a href="./c.txt">c.txt</a>`,
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
					assert.Contains(t, recorder.Header().Get("Content-Security-Policy"), "sandbox allow-same-origin")
					assert.Equal(t, "SAMEORIGIN", recorder.Header().Get("X-Frame-Options"))
					// verify that the files are contained in the listing we got back
					assert.Len(t, tt.directoryFiles, strings.Count(string(all), "<li>"))
					for _, file := range tt.directoryFiles {
						assert.Contains(t, string(all), file)
					}
				} else {
					assert.Equal(t, "my-data", string(all))
				}
			}
		})
	}
}

func TestArtifactServer_RenderDirectoryListings(t *testing.T) {
	s := newServer(t)

	t.Run("Empty Directory", func(t *testing.T) {
		expected := `<html><body><ul>
<li><a href="..">..</a></li>
</ul></body></html>`
		actual, err := s.renderDirectoryListing([]string{}, "")
		require.NoError(t, err)
		assert.Equal(t, expected, string(actual))
	})

	t.Run("Single File", func(t *testing.T) {
		expected := `<html><body><ul>
<li><a href="..">..</a></li>
<li><a href="./foo.html">foo.html</a></li>
</ul></body></html>`
		actual, err := s.renderDirectoryListing([]string{"foo.html"}, "")
		require.NoError(t, err)
		assert.Equal(t, expected, string(actual))
	})

	t.Run("Nested Files", func(t *testing.T) {
		expected := `<html><body><ul>
<li><a href="..">..</a></li>
<li><a href="./foo.html">foo.html</a></li>
<li><a href="./dir/">dir/</a></li>
<li><a href="./dir2/">dir2/</a></li>
</ul></body></html>`
		actual, err := s.renderDirectoryListing([]string{
			"dir/foo.html",
			"dir/dir/bar.html",
			"dir/dir2/baz.html",
			"dir/dir/bar2.html",
		}, "dir")
		require.NoError(t, err)
		assert.Equal(t, expected, string(actual))
	})

	t.Run("XSS Filtering", func(t *testing.T) {
		expected := `<html><body><ul>
<li><a href="..">..</a></li>
<li><a href="./xss%5c%22%3e%3cimg%20src=x%20onerror=%22alert%28document.domain%29%22%3e.html">xss\&#34;&gt;&lt;img src=x onerror=&#34;alert(document.domain)&#34;&gt;.html</a></li>
<li><a href="./javascript:alert%28document.domain%29">javascript:alert(document.domain)</a></li>
<li><a href="./javascript:%5cu0061lert%281%29">javascript:\u0061lert(1)</a></li>
<li><a href="./%3cInput%20value%20=%20%22XSS%22%20type%20=%20text%3e">&lt;Input value = &#34;XSS&#34; type = text&gt;</a></li>
</ul></body></html>`
		actual, err := s.renderDirectoryListing([]string{
			`xss\"><img src=x onerror="alert(document.domain)">.html`,
			`javascript:alert(document.domain)`,
			`javascript:\u0061lert(1)`,
			`<Input value = "XSS" type = text>`,
		}, "")
		require.NoError(t, err)
		assert.Equal(t, expected, string(actual))
	})
}

func TestArtifactServer_GetOutputArtifact(t *testing.T) {
	s := newServer(t)

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
			require.Equal(t, 200, recorder.Result().StatusCode)
			assert.Equal(t, fmt.Sprintf(`filename="%s"`, tt.fileName), recorder.Header().Get("Content-Disposition"))
			all, err := io.ReadAll(recorder.Result().Body)
			if err != nil {
				panic(fmt.Sprintf("failed to read http body: %v", err))
			}
			assert.Equal(t, "my-data", string(all))
		})
	}
}

func TestArtifactServer_GetOutputArtifactWithTemplate(t *testing.T) {
	s := newServer(t)

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
			require.Equal(t, 200, recorder.Result().StatusCode)
			assert.Equal(t, fmt.Sprintf(`filename="%s"`, tt.fileName), recorder.Header().Get("Content-Disposition"))
			all, err := io.ReadAll(recorder.Result().Body)
			if err != nil {
				panic(fmt.Sprintf("failed to read http body: %v", err))
			}
			assert.Equal(t, "my-data", string(all))
		})
	}
}

func TestArtifactServer_GetOutputArtifactWithInlineTemplate(t *testing.T) {
	s := newServer(t)

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
			require.Equal(t, 200, recorder.Result().StatusCode)
			assert.Equal(t, fmt.Sprintf(`filename="%s"`, tt.fileName), recorder.Header().Get("Content-Disposition"))
			all, err := io.ReadAll(recorder.Result().Body)
			if err != nil {
				panic(fmt.Sprintf("failed to read http body: %v", err))
			}
			assert.Equal(t, "my-data", string(all))
		})
	}
}

func TestArtifactServer_GetInputArtifact(t *testing.T) {
	s := newServer(t)

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
			require.Equal(t, 200, recorder.Result().StatusCode)
			assert.Equal(t, fmt.Sprintf(`filename="%s"`, tt.fileName), recorder.Result().Header.Get("Content-Disposition"))
			all, err := io.ReadAll(recorder.Result().Body)
			if err != nil {
				panic(fmt.Sprintf("failed to read http body: %v", err))
			}
			assert.Equal(t, "my-data", string(all))
		})
	}
}

// TestArtifactServer_NodeWithoutArtifact makes sure that the server doesn't panic due to a nil-pointer error
// when trying to get an artifact from a node result without any artifacts
func TestArtifactServer_NodeWithoutArtifact(t *testing.T) {
	s := newServer(t)
	r := &http.Request{}
	r.URL = mustParse(fmt.Sprintf("/input-artifacts/my-ns/my-wf/my-node-no-artifacts/%s", "my-artifact"))
	recorder := httptest.NewRecorder()
	s.GetInputArtifact(recorder, r)
	// make sure there is no nil pointer panic
	assert.Equal(t, 500, recorder.Result().StatusCode)
	s.GetOutputArtifact(recorder, r)
	assert.Equal(t, 500, recorder.Result().StatusCode)
}

func TestArtifactServer_GetOutputArtifactWithoutInstanceID(t *testing.T) {
	s := newServer(t)
	r := &http.Request{}
	r.URL = mustParse("/artifacts/my-ns/your-wf/my-node-1/my-artifact")
	recorder := httptest.NewRecorder()
	s.GetOutputArtifact(recorder, r)
	assert.NotEqual(t, 200, recorder.Result().StatusCode)
}

func TestArtifactServer_GetOutputArtifactByUID(t *testing.T) {
	s := newServer(t)
	r := &http.Request{}
	r.URL = mustParse("/artifacts/my-uuid/my-node-1/my-artifact")
	recorder := httptest.NewRecorder()
	s.GetOutputArtifactByUID(recorder, r)
	assert.Equal(t, 401, recorder.Result().StatusCode)
}

func TestArtifactServer_GetArtifactByUIDInvalidRequestPath(t *testing.T) {
	s := newServer(t)
	r := &http.Request{}
	// missing my-artifact part to have a valid URL
	r.URL = mustParse("/input-artifacts/my-uuid/my-node-1")
	recorder := httptest.NewRecorder()
	s.GetInputArtifactByUID(recorder, r)
	// make sure there is no index out of bounds error
	assert.Equal(t, 400, recorder.Result().StatusCode)
	output, err := io.ReadAll(recorder.Result().Body)
	require.NoError(t, err)
	assert.Contains(t, string(output), "Bad Request")

	recorder = httptest.NewRecorder()
	s.GetOutputArtifactByUID(recorder, r)
	assert.Equal(t, 400, recorder.Result().StatusCode)
	output, err = io.ReadAll(recorder.Result().Body)
	require.NoError(t, err)
	assert.Contains(t, string(output), "Bad Request")
}

func TestArtifactServer_httpBadRequestError(t *testing.T) {
	s := newServer(t)
	recorder := httptest.NewRecorder()
	s.httpBadRequestError(recorder)

	assert.Equal(t, http.StatusBadRequest, recorder.Result().StatusCode)
	output, err := io.ReadAll(recorder.Result().Body)
	require.NoError(t, err)
	assert.Contains(t, string(output), "Bad Request")
}

func TestArtifactServer_httpFromError(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	s := newServer(t)
	recorder := httptest.NewRecorder()
	err := errors.New("math: square root of negative number")

	s.httpFromError(ctx, err, recorder)

	assert.Equal(t, http.StatusInternalServerError, recorder.Result().StatusCode)
	output, err := io.ReadAll(recorder.Result().Body)
	require.NoError(t, err)
	assert.Equal(t, "Internal Server Error\n", string(output))

	recorder = httptest.NewRecorder()
	err = apierr.NewUnauthorized("")

	s.httpFromError(ctx, err, recorder)

	assert.Equal(t, http.StatusUnauthorized, recorder.Result().StatusCode)
	output, err = io.ReadAll(recorder.Result().Body)
	require.NoError(t, err)
	assert.Contains(t, string(output), "Unauthorized")

	recorder = httptest.NewRecorder()
	err = argoerrors.New(argoerrors.CodeNotFound, "not found")

	s.httpFromError(ctx, err, recorder)
	assert.Equal(t, http.StatusNotFound, recorder.Result().StatusCode)
}

// TestArtifactServer_GetArtifactByUID_WorkflowNotInArchive verifies that querying an artifact
// by UID for a workflow that is not yet in the archive returns 404 instead of panicking
// with a nil pointer dereference. This covers the race window between workflow completion
// and the archiver persisting the record.
func TestArtifactServer_GetArtifactByUID_WorkflowNotInArchive(t *testing.T) {
	gatekeeper := &authmocks.Gatekeeper{}
	kube := kubefake.NewClientset()
	instanceID := "my-instanceid"
	argo := fakewfv1.NewClientset()
	ctx := context.WithValue(context.WithValue(logging.TestContext(t.Context()), auth.KubeKey, kube), auth.WfKey, argo)
	gatekeeper.On("ContextWithRequest", mock.Anything, mock.Anything).Return(ctx, nil)

	a := &sqldbmocks.WorkflowArchive{}
	// Simulate the race: workflow exists but is not yet archived — GetWorkflow returns (nil, nil).
	a.On("GetWorkflow", mock.Anything, "not-yet-archived-uid", "", "").Return((*wfv1.Workflow)(nil), nil)

	fakeArtifactDriverFactory := func(_ context.Context, _ *wfv1.Artifact, _ resource.Interface) (artifactscommon.ArtifactDriver, error) {
		return &fakeArtifactDriver{data: []byte("my-data")}, nil
	}
	artifactRepositories := armocks.DummyArtifactRepositories(&wfv1.ArtifactRepository{
		S3: &wfv1.S3ArtifactRepository{
			S3Bucket: wfv1.S3Bucket{Endpoint: "my-endpoint", Bucket: "my-bucket"},
		},
	})
	s := newArtifactServer(gatekeeper, hydratorfake.Noop, a, instanceid.NewService(instanceID), fakeArtifactDriverFactory, artifactRepositories, logging.RequireLoggerFromContext(ctx))

	t.Run("GetOutputArtifactByUID", func(t *testing.T) {
		r := &http.Request{}
		r.URL = mustParse("/artifacts/not-yet-archived-uid/my-node-1/my-artifact")
		recorder := httptest.NewRecorder()
		s.GetOutputArtifactByUID(recorder, r)
		assert.Equal(t, http.StatusNotFound, recorder.Result().StatusCode)
	})

	t.Run("GetInputArtifactByUID", func(t *testing.T) {
		r := &http.Request{}
		r.URL = mustParse("/input-artifacts/not-yet-archived-uid/my-node-1/my-artifact")
		recorder := httptest.NewRecorder()
		s.GetInputArtifactByUID(recorder, r)
		assert.Equal(t, http.StatusNotFound, recorder.Result().StatusCode)
	})
}

// TestArtifactServer_GetArtifactFile_ArchivedWorkflowNotFound verifies that the
// archived-workflows branch of GetArtifactFile returns 404 when GetWorkflow returns
// (nil, nil) instead of panicking with a nil pointer dereference.
func TestArtifactServer_GetArtifactFile_ArchivedWorkflowNotFound(t *testing.T) {
	gatekeeper := &authmocks.Gatekeeper{}
	kube := kubefake.NewClientset()
	instanceID := "my-instanceid"
	argo := fakewfv1.NewClientset()
	ctx := context.WithValue(context.WithValue(logging.TestContext(t.Context()), auth.KubeKey, kube), auth.WfKey, argo)
	gatekeeper.On("ContextWithRequest", mock.Anything, mock.Anything).Return(ctx, nil)

	a := &sqldbmocks.WorkflowArchive{}
	a.On("GetWorkflow", mock.Anything, "not-yet-archived-uid", "", "").Return((*wfv1.Workflow)(nil), nil)

	fakeArtifactDriverFactory := func(_ context.Context, _ *wfv1.Artifact, _ resource.Interface) (artifactscommon.ArtifactDriver, error) {
		return &fakeArtifactDriver{data: []byte("my-data")}, nil
	}
	artifactRepositories := armocks.DummyArtifactRepositories(&wfv1.ArtifactRepository{
		S3: &wfv1.S3ArtifactRepository{
			S3Bucket: wfv1.S3Bucket{Endpoint: "my-endpoint", Bucket: "my-bucket"},
		},
	})
	s := newArtifactServer(gatekeeper, hydratorfake.Noop, a, instanceid.NewService(instanceID), fakeArtifactDriverFactory, artifactRepositories, logging.RequireLoggerFromContext(ctx))

	r := &http.Request{}
	r.URL = mustParse("/artifact-files/my-ns/archived-workflows/not-yet-archived-uid/my-node-1/outputs/my-artifact")
	recorder := httptest.NewRecorder()
	s.GetArtifactFile(recorder, r)
	assert.Equal(t, http.StatusNotFound, recorder.Result().StatusCode)
}

// newServerForUpload creates a test server with WorkflowTemplate for upload tests
func newServerForUpload(t *testing.T, saveStreamError error) *ArtifactServer {
	t.Helper()
	return newServerForUploadWithAuth(t, saveStreamError, true)
}

// newServerForUploadWithAuth is like newServerForUpload but lets the caller control
// whether SelfSubjectAccessReview checks (auth.CanI) are allowed or denied.
func newServerForUploadWithAuth(t *testing.T, saveStreamError error, authorized bool) *ArtifactServer {
	t.Helper()
	gatekeeper := &authmocks.Gatekeeper{}
	kube := kubefake.NewClientset()
	kube.PrependReactor("create", "selfsubjectaccessreviews", func(_ ktesting.Action) (bool, runtime.Object, error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: authorized},
		}, nil
	})
	instanceID := "my-instanceid"

	// WorkflowTemplate with artifact in arguments
	wft := &wfv1.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-wft",
			Labels:    map[string]string{common.LabelKeyControllerInstanceID: instanceID},
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "main",
			Arguments: wfv1.Arguments{
				Artifacts: []wfv1.Artifact{
					{
						Name: "input-artifact",
						ArtifactLocation: wfv1.ArtifactLocation{
							S3: &wfv1.S3Artifact{
								S3Bucket: wfv1.S3Bucket{
									Endpoint: "s3.amazonaws.com",
									Bucket:   "my-bucket",
								},
								Key: "original/key.zip",
							},
						},
					},
				},
			},
			Templates: []wfv1.Template{
				{
					Name: "main",
					Container: &corev1.Container{
						Image:   "alpine",
						Command: []string{"echo"},
						Args:    []string{"hello"},
					},
				},
			},
		},
	}

	// WorkflowTemplate without location (relies on default repository)
	wftNoLocation := &wfv1.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-wft-no-location",
			Labels:    map[string]string{common.LabelKeyControllerInstanceID: instanceID},
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "main",
			Arguments: wfv1.Arguments{
				Artifacts: []wfv1.Artifact{
					{
						Name: "input-artifact",
						// No ArtifactLocation - relies on default repository
					},
				},
			},
			Templates: []wfv1.Template{
				{
					Name: "main",
					Container: &corev1.Container{
						Image:   "alpine",
						Command: []string{"echo"},
						Args:    []string{"hello"},
					},
				},
			},
		},
	}

	argo := fakewfv1.NewClientset(wft, wftNoLocation)
	ctx := context.WithValue(context.WithValue(logging.TestContext(t.Context()), auth.KubeKey, kube), auth.WfKey, argo)
	gatekeeper.On("ContextWithRequest", mock.Anything, mock.Anything).Return(ctx, nil)
	a := &sqldbmocks.WorkflowArchive{}

	fakeArtifactDriverFactory := func(_ context.Context, _ *wfv1.Artifact, _ resource.Interface) (artifactscommon.ArtifactDriver, error) {
		return &fakeArtifactDriver{data: []byte("my-data"), saveStreamError: saveStreamError}, nil
	}

	artifactRepositories := armocks.DummyArtifactRepositories(&wfv1.ArtifactRepository{
		S3: &wfv1.S3ArtifactRepository{
			S3Bucket: wfv1.S3Bucket{
				Endpoint: "my-endpoint",
				Bucket:   "my-bucket",
			},
		},
	})

	return newArtifactServer(gatekeeper, hydratorfake.Noop, a, instanceid.NewService(instanceID), fakeArtifactDriverFactory, artifactRepositories, logging.RequireLoggerFromContext(ctx))
}

// createMultipartRequest creates a multipart form request with a file
func createMultipartRequest(t *testing.T, path string, fileContent []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test-file.zip")
	require.NoError(t, err)
	_, err = part.Write(fileContent)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestArtifactServer_UploadInputArtifact(t *testing.T) {
	t.Run("Success - upload artifact with location", func(t *testing.T) {
		s := newServerForUpload(t, nil)
		recorder := httptest.NewRecorder()
		req := createMultipartRequest(t, "/upload-artifacts/my-ns/my-wft/input-artifact", []byte("test content"))

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)

		var response map[string]any
		err := json.NewDecoder(recorder.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "input-artifact", response["name"])
		assert.Contains(t, response["key"], "uploads/my-ns/")
		assert.Contains(t, response["key"], "test-file.zip")

		// key must be uploads/{namespace}/{uuid}/{filename}, with the uuid segment
		// a proper RFC-4122 (version 4, variant 10) UUID rather than an arbitrary
		// hex string that happens to parse.
		key, ok := response["key"].(string)
		require.True(t, ok)
		parts := strings.Split(key, "/")
		require.Len(t, parts, 4)
		parsed, err := uuid.Parse(parts[2])
		require.NoError(t, err)
		assert.Equal(t, uuid.Version(4), parsed.Version())
		assert.Equal(t, uuid.RFC4122, parsed.Variant())
	})

	t.Run("Success - upload artifact without location (uses default repository)", func(t *testing.T) {
		s := newServerForUpload(t, nil)
		recorder := httptest.NewRecorder()
		req := createMultipartRequest(t, "/upload-artifacts/my-ns/my-wft-no-location/input-artifact", []byte("test content"))

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)

		var response map[string]any
		err := json.NewDecoder(recorder.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "input-artifact", response["name"])
		assert.Contains(t, response["key"], "uploads/my-ns/")
	})

	t.Run("Error - method not allowed", func(t *testing.T) {
		s := newServerForUpload(t, nil)
		recorder := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/upload-artifacts/my-ns/my-wft/input-artifact", nil)

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusMethodNotAllowed, recorder.Result().StatusCode)
	})

	t.Run("Error - invalid path", func(t *testing.T) {
		s := newServerForUpload(t, nil)
		recorder := httptest.NewRecorder()
		req := createMultipartRequest(t, "/upload-artifacts/my-ns", []byte("test content"))

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Result().StatusCode)
	})

	t.Run("Error - WorkflowTemplate not found", func(t *testing.T) {
		s := newServerForUpload(t, nil)
		recorder := httptest.NewRecorder()
		req := createMultipartRequest(t, "/upload-artifacts/my-ns/non-existent-wft/input-artifact", []byte("test content"))

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusNotFound, recorder.Result().StatusCode)
	})

	t.Run("Error - artifact not found in template", func(t *testing.T) {
		s := newServerForUpload(t, nil)
		recorder := httptest.NewRecorder()
		req := createMultipartRequest(t, "/upload-artifacts/my-ns/my-wft/non-existent-artifact", []byte("test content"))

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusNotFound, recorder.Result().StatusCode)
	})

	t.Run("Error - SaveStream fails", func(t *testing.T) {
		s := newServerForUpload(t, errors.New("storage error"))
		recorder := httptest.NewRecorder()
		req := createMultipartRequest(t, "/upload-artifacts/my-ns/my-wft/input-artifact", []byte("test content"))

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusInternalServerError, recorder.Result().StatusCode)
	})

	t.Run("Error - path traversal in filename", func(t *testing.T) {
		s := newServerForUpload(t, nil)
		recorder := httptest.NewRecorder()

		// Create multipart request with malicious filename
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "../../../etc/passwd")
		require.NoError(t, err)
		_, err = part.Write([]byte("malicious content"))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload-artifacts/my-ns/my-wft/input-artifact", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)
		var response map[string]any
		err = json.NewDecoder(recorder.Body).Decode(&response)
		require.NoError(t, err)
		// path.Base strips traversal — key should contain only "passwd"
		assert.Contains(t, response["key"], "/passwd")
		assert.NotContains(t, response["key"], "..")
	})

	t.Run("Success - Windows-style backslash traversal in filename is sanitized", func(t *testing.T) {
		s := newServerForUpload(t, nil)
		recorder := httptest.NewRecorder()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", `..\..\etc\passwd`)
		require.NoError(t, err)
		_, err = part.Write([]byte("malicious content"))
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload-artifacts/my-ns/my-wft/input-artifact", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)
		var response map[string]any
		err = json.NewDecoder(recorder.Body).Decode(&response)
		require.NoError(t, err)
		key, ok := response["key"].(string)
		require.True(t, ok)
		assert.True(t, strings.HasSuffix(key, "/passwd"), "expected key to end with /passwd, got %q", key)
		assert.NotContains(t, key, `\`)
		assert.NotContains(t, key, "..")
	})

	t.Run("Error - missing file in form", func(t *testing.T) {
		s := newServerForUpload(t, nil)
		recorder := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload-artifacts/my-ns/my-wft/input-artifact", strings.NewReader(""))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Result().StatusCode)
	})

	t.Run("Error - upload without workflows.create returns 403 before body read", func(t *testing.T) {
		s := newServerForUploadWithAuth(t, nil, false)
		recorder := httptest.NewRecorder()
		req := createMultipartRequest(t, "/upload-artifacts/my-ns/my-wft/input-artifact", []byte("test content"))
		req.Body = io.NopCloser(&failOnReadReader{t: t})

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusForbidden, recorder.Result().StatusCode)
	})

	t.Run("Error - upload over ARGO_SERVER_MAX_ARTIFACT_UPLOAD_BYTES returns 413", func(t *testing.T) {
		t.Setenv("ARGO_SERVER_MAX_ARTIFACT_UPLOAD_BYTES", "10")
		s := newServerForUpload(t, nil)
		recorder := httptest.NewRecorder()
		req := createMultipartRequest(t, "/upload-artifacts/my-ns/my-wft/input-artifact", []byte("this content is longer than 10 bytes"))

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Result().StatusCode)
	})

	t.Run("Success - multipart temp files removed after successful upload", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMPDIR", tmpDir)
		s := newServerForUpload(t, nil)
		recorder := httptest.NewRecorder()
		// Exceeds the handler's 32MiB in-memory threshold, forcing multipart to spill to disk.
		largeContent := bytes.Repeat([]byte("a"), 33<<20)
		req := createMultipartRequest(t, "/upload-artifacts/my-ns/my-wft/input-artifact", largeContent)

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)
		leaked, err := filepath.Glob(filepath.Join(tmpDir, "multipart-*"))
		require.NoError(t, err)
		assert.Empty(t, leaked)
	})

	t.Run("Error - multipart temp files removed after SaveStream error", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMPDIR", tmpDir)
		s := newServerForUpload(t, errors.New("storage error"))
		recorder := httptest.NewRecorder()
		largeContent := bytes.Repeat([]byte("a"), 33<<20)
		req := createMultipartRequest(t, "/upload-artifacts/my-ns/my-wft/input-artifact", largeContent)

		s.UploadInputArtifact(recorder, req)

		assert.Equal(t, http.StatusInternalServerError, recorder.Result().StatusCode)
		leaked, err := filepath.Glob(filepath.Join(tmpDir, "multipart-*"))
		require.NoError(t, err)
		assert.Empty(t, leaked)
	})
}

// failOnReadReader fails the test if its Read method is invoked. It is used to prove
// that the request body is never consumed when authorization is denied.
type failOnReadReader struct {
	t *testing.T
}

func (r *failOnReadReader) Read(_ []byte) (int, error) {
	r.t.Helper()
	r.t.Fatal("request body must not be read before authorization checks pass")
	return 0, io.EOF
}

func TestArtifactServer_GateKeepingForwardsHeaders(t *testing.T) {
	gatekeeper := &authmocks.Gatekeeper{}
	ctx := logging.TestContext(t.Context())

	var gatekeeperCtx context.Context
	gatekeeper.On("ContextWithRequest", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		gatekeeperCtx = args.Get(0).(context.Context)
	}).Return(ctx, nil)

	s := &ArtifactServer{
		gatekeeper: gatekeeper,
	}

	r := httptest.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"/artifacts/my-ns/my-wf/my-node/my-artifact",
		nil,
	)
	r.Header.Set("X-Forwarded-User", "pradeep")
	r.Header.Set("X-Proxy-Auth", "secret")
	r.Header.Set("X-Forwarded-Groups", "admin")

	_, err := s.gateKeeping(r, servertypes.NamespaceHolder("my-ns"))

	require.NoError(t, err)
	gatekeeper.AssertExpectations(t)

	md, ok := metadata.FromIncomingContext(gatekeeperCtx)
	require.True(t, ok)
	assert.Equal(t, []string{"pradeep"}, md.Get("x-forwarded-user"))
	assert.Equal(t, []string{"secret"}, md.Get("x-proxy-auth"))
	assert.Equal(t, []string{"admin"}, md.Get("x-forwarded-groups"))
}

// templateIdentityLocation is an S3 location whose bucket, endpoint and credential
// selectors all name id, so an assertion shows which template it was taken from.
func templateIdentityLocation(id string) *wfv1.ArtifactLocation {
	return &wfv1.ArtifactLocation{
		S3: &wfv1.S3Artifact{
			Key: "archive-key-" + id,
			S3Bucket: wfv1.S3Bucket{
				Bucket:   "bucket-" + id,
				Endpoint: id + ".example.com:9000",
				AccessKeySecret: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "secret-" + id},
					Key:                  "accessKey",
				},
				SecretKeySecret: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "secret-" + id},
					Key:                  "secretKey",
				},
			},
		},
	}
}

// templateIdentityNode is a node with one input and one output artifact that only
// carry an object key, so their location must come from the node's template.
func templateIdentityNode(id string) wfv1.NodeStatus {
	return wfv1.NodeStatus{
		ID: id,
		Inputs: &wfv1.Inputs{Artifacts: wfv1.Artifacts{{
			Name:             "in",
			ArtifactLocation: wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{Key: "my-wf/" + id + "/in.tgz"}},
		}}},
		Outputs: &wfv1.Outputs{Artifacts: wfv1.Artifacts{{
			Name:             "out",
			ArtifactLocation: wfv1.ArtifactLocation{S3: &wfv1.S3Artifact{Key: "my-wf/" + id + "/out.tgz"}},
		}}},
	}
}

// templateIdentityS3 is the S3 location expected for an artifact with the given
// object key that is served from templateIdentityLocation(id).
func templateIdentityS3(id, key string) *wfv1.S3Artifact {
	s3 := templateIdentityLocation(id).S3
	s3.Key = key
	return s3
}

func templateIdentityMainTemplate(archiveLocation *wfv1.ArtifactLocation) wfv1.Template {
	return wfv1.Template{Name: "main", ArchiveLocation: archiveLocation}
}

func newTemplateIdentityServer(t *testing.T) (context.Context, *ArtifactServer, *int) {
	t.Helper()
	repository := &wfv1.ArtifactRepository{S3: &wfv1.S3ArtifactRepository{S3Bucket: templateIdentityLocation("repository").S3.S3Bucket}}
	return newTemplateIdentityServerWithRepositories(t, armocks.DummyArtifactRepositories(repository))
}

// unavailableArtifactRepositories fails every lookup, for artifacts that must not
// need the artifact repository.
func unavailableArtifactRepositories() *armocks.Interface {
	repositories := &armocks.Interface{}
	repositories.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("artifact repository unavailable"))
	return repositories
}

func newTemplateIdentityServerWithRepositories(t *testing.T, repositories *armocks.Interface) (context.Context, *ArtifactServer, *int) {
	t.Helper()
	ctx := context.WithValue(logging.TestContext(t.Context()), auth.KubeKey, kubefake.NewClientset())
	driversCreated := 0
	driverFactory := func(_ context.Context, _ *wfv1.Artifact, _ resource.Interface) (artifactscommon.ArtifactDriver, error) {
		driversCreated++
		return &fakeArtifactDriver{data: []byte("my-data")}, nil
	}
	s := newArtifactServer(&authmocks.Gatekeeper{}, hydratorfake.Noop, &sqldbmocks.WorkflowArchive{}, instanceid.NewService(""),
		driverFactory, repositories, logging.RequireLoggerFromContext(ctx))
	return ctx, s, &driversCreated
}

// TestArtifactServer_GetArtifactAndDriverTemplateIdentity checks that an artifact
// without its own location is served from the archive location of the template
// that actually ran the node, identified by scope, resource and template name,
// and never from another template that merely shares its name.
//
// Cases marked mapOrder differ only in which entry of Status.StoredTemplates
// matches. A lookup by bare template name would pick one of them in map
// iteration order, which Go leaves unspecified, so those cases are repeated to
// make it very unlikely that such a lookup passes by chance.
func TestArtifactServer_GetArtifactAndDriverTemplateIdentity(t *testing.T) {
	const mapOrderAttempts = 256

	referencedNode := func(id string, ref *wfv1.TemplateRef) wfv1.NodeStatus {
		n := templateIdentityNode(id)
		n.TemplateRef = ref
		return n
	}
	localNode := func(id, templateName string) wfv1.NodeStatus {
		n := templateIdentityNode(id)
		n.TemplateName = templateName
		return n
	}
	scopedNode := func(id, scope, templateName string) wfv1.NodeStatus {
		n := localNode(id, templateName)
		n.TemplateScope = scope
		return n
	}

	tests := []struct {
		name            string
		localTemplates  []wfv1.Template
		storedTemplates map[string]wfv1.Template
		node            wfv1.NodeStatus
		isInput         bool
		// wantLocation is the id passed to templateIdentityLocation for the expected location.
		wantLocation string
		// mapOrder marks cases whose same-named candidates are all Status.StoredTemplates entries.
		mapOrder bool
	}{
		{
			name: "first of two WorkflowTemplates with the same template name",
			storedTemplates: map[string]wfv1.Template{
				"namespaced/wft-a/main": templateIdentityMainTemplate(templateIdentityLocation("wft-a")),
				"namespaced/wft-b/main": templateIdentityMainTemplate(templateIdentityLocation("wft-b")),
			},
			node:         referencedNode("node-a", &wfv1.TemplateRef{Name: "wft-a", Template: "main"}),
			wantLocation: "wft-a",
			mapOrder:     true,
		},
		{
			name: "second of two WorkflowTemplates with the same template name",
			storedTemplates: map[string]wfv1.Template{
				"namespaced/wft-a/main": templateIdentityMainTemplate(templateIdentityLocation("wft-a")),
				"namespaced/wft-b/main": templateIdentityMainTemplate(templateIdentityLocation("wft-b")),
			},
			node:         referencedNode("node-b", &wfv1.TemplateRef{Name: "wft-b", Template: "main"}),
			wantLocation: "wft-b",
			mapOrder:     true,
		},
		{
			name:           "referenced main is not shadowed by local main",
			localTemplates: []wfv1.Template{templateIdentityMainTemplate(templateIdentityLocation("local"))},
			storedTemplates: map[string]wfv1.Template{
				"namespaced/wft-a/main": templateIdentityMainTemplate(templateIdentityLocation("wft-a")),
			},
			node:         referencedNode("node-ref", &wfv1.TemplateRef{Name: "wft-a", Template: "main"}),
			wantLocation: "wft-a",
		},
		{
			name:           "input artifact of referenced main is not shadowed by local main",
			localTemplates: []wfv1.Template{templateIdentityMainTemplate(templateIdentityLocation("local"))},
			storedTemplates: map[string]wfv1.Template{
				"namespaced/wft-a/main": templateIdentityMainTemplate(templateIdentityLocation("wft-a")),
			},
			node:         referencedNode("node-ref", &wfv1.TemplateRef{Name: "wft-a", Template: "main"}),
			isInput:      true,
			wantLocation: "wft-a",
		},
		{
			name:           "local main keeps its own location next to a referenced main",
			localTemplates: []wfv1.Template{templateIdentityMainTemplate(templateIdentityLocation("local"))},
			storedTemplates: map[string]wfv1.Template{
				"namespaced/wft-a/main": templateIdentityMainTemplate(templateIdentityLocation("wft-a")),
			},
			node:         localNode("node-local", "main"),
			wantLocation: "local",
		},
		{
			name:           "referenced main without archive location uses the repository, not local main",
			localTemplates: []wfv1.Template{templateIdentityMainTemplate(templateIdentityLocation("local"))},
			storedTemplates: map[string]wfv1.Template{
				"namespaced/wft-a/main": templateIdentityMainTemplate(nil),
			},
			node:         referencedNode("node-ref", &wfv1.TemplateRef{Name: "wft-a", Template: "main"}),
			wantLocation: "repository",
		},
		{
			name: "namespaced template with the same name as a cluster template",
			storedTemplates: map[string]wfv1.Template{
				"namespaced/shared/main": templateIdentityMainTemplate(templateIdentityLocation("namespaced")),
				"cluster/shared/main":    templateIdentityMainTemplate(templateIdentityLocation("cluster")),
			},
			node:         referencedNode("node-namespaced", &wfv1.TemplateRef{Name: "shared", Template: "main"}),
			wantLocation: "namespaced",
			mapOrder:     true,
		},
		{
			name: "cluster template with the same name as a namespaced template",
			storedTemplates: map[string]wfv1.Template{
				"namespaced/shared/main": templateIdentityMainTemplate(templateIdentityLocation("namespaced")),
				"cluster/shared/main":    templateIdentityMainTemplate(templateIdentityLocation("cluster")),
			},
			node:         referencedNode("node-cluster", &wfv1.TemplateRef{Name: "shared", Template: "main", ClusterScope: true}),
			wantLocation: "cluster",
			mapOrder:     true,
		},
		{
			name:           "namespaced template scope without template ref",
			localTemplates: []wfv1.Template{templateIdentityMainTemplate(templateIdentityLocation("local"))},
			storedTemplates: map[string]wfv1.Template{
				"namespaced/wft-a/main": templateIdentityMainTemplate(templateIdentityLocation("wft-a")),
			},
			node:         scopedNode("node-scoped", "namespaced/wft-a", "main"),
			wantLocation: "wft-a",
		},
		{
			name:           "cluster template scope without template ref",
			localTemplates: []wfv1.Template{templateIdentityMainTemplate(templateIdentityLocation("local"))},
			storedTemplates: map[string]wfv1.Template{
				"cluster/cwft-a/main": templateIdentityMainTemplate(templateIdentityLocation("cwft-a")),
			},
			node:         scopedNode("node-scoped", "cluster/cwft-a", "main"),
			wantLocation: "cwft-a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, s, _ := newTemplateIdentityServer(t)
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "my-wf"},
				Spec:       wfv1.WorkflowSpec{Templates: tt.localTemplates},
				Status: wfv1.WorkflowStatus{
					Nodes:           wfv1.Nodes{tt.node.ID: tt.node},
					StoredTemplates: tt.storedTemplates,
				},
			}
			artifactName := "out"
			if tt.isInput {
				artifactName = "in"
			}
			want := templateIdentityS3(tt.wantLocation, "my-wf/"+tt.node.ID+"/"+artifactName+".tgz")

			attempts := 1
			if tt.mapOrder {
				attempts = mapOrderAttempts
			}
			for attempt := range attempts {
				art, driver, err := s.getArtifactAndDriver(ctx, tt.node.ID, artifactName, tt.isInput, wf, nil)
				require.NoError(t, err, "attempt %d", attempt)
				require.NotNil(t, driver, "attempt %d", attempt)
				require.Equal(t, want, art.S3, "attempt %d", attempt)
			}
		})
	}
}

// An artifact that records its own complete location is served from it, without
// consulting its template or the artifact repository, so it stays downloadable
// when its template's stored snapshot is missing.
func TestArtifactServer_GetArtifactAndDriverExplicitLocation(t *testing.T) {
	tests := []struct {
		name            string
		localTemplates  []wfv1.Template
		storedTemplates map[string]wfv1.Template
	}{
		{
			name:           "stored template with its own archive location",
			localTemplates: []wfv1.Template{templateIdentityMainTemplate(templateIdentityLocation("local"))},
			storedTemplates: map[string]wfv1.Template{
				"namespaced/wft-a/main": templateIdentityMainTemplate(templateIdentityLocation("wft-a")),
			},
		},
		{name: "stored template missing"},
		{
			name:           "stored template missing, local template with the same name",
			localTemplates: []wfv1.Template{templateIdentityMainTemplate(templateIdentityLocation("local"))},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, s, _ := newTemplateIdentityServerWithRepositories(t, unavailableArtifactRepositories())
			explicit := templateIdentityS3("explicit", "my-wf/node-ref/explicit.tgz")
			node := wfv1.NodeStatus{
				ID:          "node-ref",
				TemplateRef: &wfv1.TemplateRef{Name: "wft-a", Template: "main"},
				Outputs:     &wfv1.Outputs{Artifacts: wfv1.Artifacts{{Name: "out", ArtifactLocation: wfv1.ArtifactLocation{S3: explicit.DeepCopy()}}}},
			}
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "my-wf"},
				Spec:       wfv1.WorkflowSpec{Templates: tt.localTemplates},
				Status: wfv1.WorkflowStatus{
					Nodes:           wfv1.Nodes{node.ID: node},
					StoredTemplates: tt.storedTemplates,
				},
			}

			art, driver, err := s.getArtifactAndDriver(ctx, node.ID, "out", false, wf, nil)

			require.NoError(t, err)
			require.NotNil(t, driver)
			assert.Equal(t, explicit, art.S3)
		})
	}
}

// A node whose non-local template is missing from the stored snapshot is an
// error when its artifacts need the template's location, rather than being
// served from a same-named local template's location.
func TestArtifactServer_GetArtifactAndDriverMissingStoredTemplate(t *testing.T) {
	referencedNode := templateIdentityNode("node-missing")
	referencedNode.TemplateRef = &wfv1.TemplateRef{Name: "wft-missing", Template: "main"}
	namespacedScopeNode := templateIdentityNode("node-missing")
	namespacedScopeNode.TemplateScope, namespacedScopeNode.TemplateName = "namespaced/wft-missing", "main"
	clusterScopeNode := templateIdentityNode("node-missing")
	clusterScopeNode.TemplateScope, clusterScopeNode.TemplateName = "cluster/cwft-missing", "main"
	localMain := []wfv1.Template{templateIdentityMainTemplate(templateIdentityLocation("local"))}

	tests := []struct {
		name           string
		node           wfv1.NodeStatus
		localTemplates []wfv1.Template
		isInput        bool
		// wantTemplateKey is the scoped identity of the missing template, named in the error.
		wantTemplateKey string
	}{
		{name: "template ref, no template with that name", node: referencedNode, wantTemplateKey: "namespaced/wft-missing/main"},
		{name: "template ref, local template with the same name", node: referencedNode, localTemplates: localMain, wantTemplateKey: "namespaced/wft-missing/main"},
		{name: "input artifact, template ref, local template with the same name", node: referencedNode, localTemplates: localMain, isInput: true, wantTemplateKey: "namespaced/wft-missing/main"},
		{name: "namespaced template scope, local template with the same name", node: namespacedScopeNode, localTemplates: localMain, wantTemplateKey: "namespaced/wft-missing/main"},
		{name: "cluster template scope, local template with the same name", node: clusterScopeNode, localTemplates: localMain, wantTemplateKey: "cluster/cwft-missing/main"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, s, driversCreated := newTemplateIdentityServer(t)
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "my-wf"},
				Spec:       wfv1.WorkflowSpec{Templates: tt.localTemplates},
				Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{tt.node.ID: tt.node}},
			}
			artifactName := "out"
			if tt.isInput {
				artifactName = "in"
			}

			_, driver, err := s.getArtifactAndDriver(ctx, tt.node.ID, artifactName, tt.isInput, wf, nil)

			require.ErrorContains(t, err, fmt.Sprintf("no template found for %q", tt.wantTemplateKey))
			assert.Nil(t, driver)
			assert.Zero(t, *driversCreated, "no artifact driver may be created for an unresolved template")
		})
	}
}

// A local template is resolved from the spec the controller executed, which is
// Status.StoredWorkflowSpec for a workflowTemplateRef workflow, not from the
// submitted Spec.
func TestArtifactServer_GetArtifactAndDriverStoredWorkflowSpec(t *testing.T) {
	tests := []struct {
		name      string
		specTmpls []wfv1.Template
	}{
		{name: "template only in the stored workflow spec"},
		{
			name:      "submitted spec has a same-named template",
			specTmpls: []wfv1.Template{templateIdentityMainTemplate(templateIdentityLocation("submitted"))},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, s, _ := newTemplateIdentityServer(t)
			node := templateIdentityNode("node-local")
			node.TemplateName = "main"
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "my-wf"},
				Spec: wfv1.WorkflowSpec{
					WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "wft-a"},
					Templates:           tt.specTmpls,
				},
				Status: wfv1.WorkflowStatus{
					Nodes:              wfv1.Nodes{node.ID: node},
					StoredWorkflowSpec: &wfv1.WorkflowSpec{Templates: []wfv1.Template{templateIdentityMainTemplate(templateIdentityLocation("stored"))}},
				},
			}

			art, driver, err := s.getArtifactAndDriver(ctx, node.ID, "out", false, wf, nil)

			require.NoError(t, err)
			require.NotNil(t, driver)
			assert.Equal(t, templateIdentityS3("stored", "my-wf/node-local/out.tgz"), art.S3)
		})
	}
}

// A node without a template name, such as one running an inline template, has
// no template identity, so its artifacts use the artifact repository.
func TestArtifactServer_GetArtifactAndDriverUnnamedTemplate(t *testing.T) {
	tests := []struct {
		name          string
		templateScope string
	}{
		{name: "no template scope"},
		{name: "local template scope", templateScope: "local/my-wf"},
		{name: "namespaced template scope", templateScope: "namespaced/wft-a"},
		{name: "cluster template scope", templateScope: "cluster/cwft-a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, s, _ := newTemplateIdentityServer(t)
			node := templateIdentityNode("node-inline")
			node.TemplateScope = tt.templateScope
			wf := &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns", Name: "my-wf"},
				Spec:       wfv1.WorkflowSpec{Templates: []wfv1.Template{templateIdentityMainTemplate(templateIdentityLocation("local"))}},
				Status: wfv1.WorkflowStatus{
					Nodes: wfv1.Nodes{node.ID: node},
					StoredTemplates: map[string]wfv1.Template{
						"namespaced/wft-a/main": templateIdentityMainTemplate(templateIdentityLocation("wft-a")),
						"cluster/cwft-a/main":   templateIdentityMainTemplate(templateIdentityLocation("cwft-a")),
					},
				},
			}

			art, driver, err := s.getArtifactAndDriver(ctx, node.ID, "out", false, wf, nil)

			require.NoError(t, err)
			require.NotNil(t, driver)
			assert.Equal(t, templateIdentityS3("repository", "my-wf/node-inline/out.tgz"), art.S3)
		})
	}
}
