package v1alpha1

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
)

func TestWorkflows(t *testing.T) {
	wfs := Workflows{
		{ObjectMeta: metav1.ObjectMeta{Name: "3"}, Status: WorkflowStatus{FinishedAt: metav1.NewTime(time.Time{}.Add(1))}},
		{ObjectMeta: metav1.ObjectMeta{Name: "2"}, Status: WorkflowStatus{FinishedAt: metav1.NewTime(time.Time{}.Add(0))}},
		{ObjectMeta: metav1.ObjectMeta{Name: "1"}, Status: WorkflowStatus{StartedAt: metav1.NewTime(time.Time{}.Add(0))}},
		{ObjectMeta: metav1.ObjectMeta{Name: "0"}, Status: WorkflowStatus{StartedAt: metav1.NewTime(time.Time{}.Add(1))}},
	}
	t.Run("Sort", func(t *testing.T) {
		sort.Sort(wfs)
		assert.Equal(t, "0", wfs[0].Name)
		assert.Equal(t, "1", wfs[1].Name)
		assert.Equal(t, "2", wfs[2].Name)
		assert.Equal(t, "3", wfs[3].Name)
	})
	t.Run("Filter", func(t *testing.T) {
		assert.Len(t, wfs.Filter(func(wf Workflow) bool { return true }), 4)
		assert.Len(t, wfs.Filter(func(wf Workflow) bool { return false }), 0)
	})
}

func TestGetTemplateByName(t *testing.T) {
	t.Run("Spec", func(t *testing.T) {
		wf := &Workflow{
			Spec: WorkflowSpec{
				Templates: []Template{
					{Name: "my-tmpl"},
				},
			},
		}
		assert.NotNil(t, wf.GetTemplateByName("my-tmpl"))
	})
	t.Run("StoredWorkflowSpec", func(t *testing.T) {
		wf := &Workflow{
			Status: WorkflowStatus{
				StoredWorkflowSpec: &WorkflowSpec{
					Templates: []Template{
						{Name: "my-tmpl"},
					},
				},
			},
		}
		assert.NotNil(t, wf.GetTemplateByName("my-tmpl"))
	})
	t.Run("StoredTemplates", func(t *testing.T) {
		wf := &Workflow{
			Status: WorkflowStatus{
				StoredTemplates: map[string]Template{
					"": {Name: "my-tmpl"},
				},
			},
		}
		assert.NotNil(t, wf.GetTemplateByName("my-tmpl"))
	})
}

func TestWorkflowCreatedAfter(t *testing.T) {
	t0 := time.Time{}
	t1 := t0.Add(time.Second)
	assert.False(t, WorkflowCreatedAfter(t1)(Workflow{ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t0}}}))
	assert.True(t, WorkflowCreatedAfter(t0)(Workflow{ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t1}}}))
}

func TestWorkflowFinishedBefore(t *testing.T) {
	t0 := time.Time{}.Add(time.Second)
	t1 := t0.Add(time.Second)
	assert.False(t, WorkflowFinishedBefore(t0)(Workflow{}))
	assert.False(t, WorkflowFinishedBefore(t1)(Workflow{}))
	assert.False(t, WorkflowFinishedBefore(t0)(Workflow{Status: WorkflowStatus{FinishedAt: metav1.Time{Time: t1}}}))
	assert.True(t, WorkflowFinishedBefore(t1)(Workflow{Status: WorkflowStatus{FinishedAt: metav1.Time{Time: t0}}}))
}

func TestWorkflowHappenedBetween(t *testing.T) {
	t0 := time.Time{}
	t1 := t0.Add(time.Second)
	t2 := t1.Add(time.Second)
	t3 := t2.Add(time.Second)
	assert.False(t, WorkflowRanBetween(t0, t3)(Workflow{}))
	assert.False(t, WorkflowRanBetween(t0, t1)(Workflow{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t0}},
		Status:     WorkflowStatus{FinishedAt: metav1.Time{Time: t1}},
	}))
	assert.False(t, WorkflowRanBetween(t1, t2)(Workflow{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t0}},
		Status:     WorkflowStatus{FinishedAt: metav1.Time{Time: t1}},
	}))
	assert.False(t, WorkflowRanBetween(t2, t3)(Workflow{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t0}},
		Status:     WorkflowStatus{FinishedAt: metav1.Time{Time: t1}},
	}))
	assert.False(t, WorkflowRanBetween(t0, t1)(Workflow{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t1}},
		Status:     WorkflowStatus{FinishedAt: metav1.Time{Time: t2}},
	}))
	assert.False(t, WorkflowRanBetween(t2, t3)(Workflow{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t1}},
		Status:     WorkflowStatus{FinishedAt: metav1.Time{Time: t2}},
	}))
	assert.True(t, WorkflowRanBetween(t0, t3)(Workflow{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t1}},
		Status:     WorkflowStatus{FinishedAt: metav1.Time{Time: t2}},
	}))
}

func TestWorkflowGetArtifactGCStrategy(t *testing.T) {
	tests := []struct {
		name                      string
		workflowArtGCStrategySpec string
		artifactGCStrategySpec    string
		expectedStrategy          ArtifactGCStrategy
	}{
		{
			name: "WorkflowLevel",
			workflowArtGCStrategySpec: `
              artifactGC:
                strategy: OnWorkflowCompletion`,
			artifactGCStrategySpec: "",
			expectedStrategy:       ArtifactGCOnWorkflowCompletion,
		},
		{
			name: "ArtifactOverride",
			workflowArtGCStrategySpec: `
              artifactGC:
                strategy: OnWorkflowCompletion`,
			artifactGCStrategySpec: `
                      artifactGC:
                        strategy: Never`,
			expectedStrategy: ArtifactGCNever,
		},
		{
			name: "NotDefined",
			workflowArtGCStrategySpec: `
              artifactGC:`,
			artifactGCStrategySpec: `
                      artifactGC:`,
			expectedStrategy: ArtifactGCNever,
		},
		{
			name: "NotDefined2",
			workflowArtGCStrategySpec: `
              artifactGC:
                strategy: ""`,
			artifactGCStrategySpec: `
                      artifactGC:
                        strategy: ""`,
			expectedStrategy: ArtifactGCNever,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			workflowSpec := fmt.Sprintf(`
            apiVersion: argoproj.io/v1alpha1
            kind: Workflow
            metadata:
              generateName: artifact-passing-
            spec:
              entrypoint: whalesay
              %s
              templates:
              - name: whalesay
                container:
                  image: docker/whalesay:latest
                  command: [sh, -c]
                  args: ["sleep 1; cowsay hello world | tee /tmp/hello_world.txt"]
                outputs:
                  artifacts:
                    - name: out
                      path: /out
                      s3:
                        key: out
                        %s`, tt.workflowArtGCStrategySpec, tt.artifactGCStrategySpec)

			wf := MustUnmarshalWorkflow(workflowSpec)
			a := wf.Spec.Templates[0].Outputs.Artifacts[0]
			gcStrategy := wf.GetArtifactGCStrategy(&a)
			assert.Equal(t, tt.expectedStrategy, gcStrategy)
		})
	}

}

func TestArtifact_ValidatePath(t *testing.T) {
	t.Run("empty path fails", func(t *testing.T) {
		a1 := Artifact{Name: "a1", Path: ""}
		err := a1.CleanPath()
		assert.EqualError(t, err, "Artifact 'a1' did not specify a path")
		assert.Equal(t, "", a1.Path)
	})

	t.Run("directory traversal above safe base dir fails", func(t *testing.T) {
		var assertPathError = func(err error) {
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "Directory traversal is not permitted")
			}
		}

		a1 := Artifact{Name: "a1", Path: "/tmp/.."}
		assertPathError(a1.CleanPath())
		assert.Equal(t, "/tmp/..", a1.Path)

		a2 := Artifact{Name: "a2", Path: "/tmp/../"}
		assertPathError(a2.CleanPath())
		assert.Equal(t, "/tmp/../", a2.Path)

		a3 := Artifact{Name: "a3", Path: "/tmp/../../etc/passwd"}
		assertPathError(a3.CleanPath())
		assert.Equal(t, "/tmp/../../etc/passwd", a3.Path)

		a4 := Artifact{Name: "a4", Path: "/tmp/../tmp"}
		assertPathError(a4.CleanPath())
		assert.Equal(t, "/tmp/../tmp", a4.Path)

		a5 := Artifact{Name: "a5", Path: "/tmp/../tmp/"}
		assertPathError(a5.CleanPath())
		assert.Equal(t, "/tmp/../tmp/", a5.Path)

		a6 := Artifact{Name: "a6", Path: "/tmp/subdir/../../tmp/subdir/"}
		assertPathError(a6.CleanPath())
		assert.Equal(t, "/tmp/subdir/../../tmp/subdir/", a6.Path)

		a7 := Artifact{Name: "a7", Path: "/tmp/../tmp-imposter"}
		assertPathError(a7.CleanPath())
		assert.Equal(t, "/tmp/../tmp-imposter", a7.Path)
	})

	t.Run("directory traversal with no safe base dir succeeds", func(t *testing.T) {
		a1 := Artifact{Name: "a1", Path: ".."}
		err := a1.CleanPath()
		assert.NoError(t, err)
		assert.Equal(t, "..", a1.Path)

		a2 := Artifact{Name: "a2", Path: "../"}
		err = a2.CleanPath()
		assert.NoError(t, err)
		assert.Equal(t, "..", a2.Path)

		a3 := Artifact{Name: "a3", Path: "../.."}
		err = a3.CleanPath()
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join("..", ".."), a3.Path)

		a4 := Artifact{Name: "a4", Path: "../etc/passwd"}
		err = a4.CleanPath()
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join("..", "etc", "passwd"), a4.Path)
	})

	t.Run("directory traversal ending within safe base dir succeeds", func(t *testing.T) {
		a1 := Artifact{Name: "a1", Path: "/tmp/../tmp/abcd"}
		err := a1.CleanPath()
		assert.NoError(t, err)
		assert.Equal(t, ensureRootPathSeparator(filepath.Join("tmp", "abcd")), a1.Path)

		a2 := Artifact{Name: "a2", Path: "/tmp/subdir/../../tmp/subdir/abcd"}
		err = a2.CleanPath()
		assert.NoError(t, err)
		assert.Equal(t, ensureRootPathSeparator(filepath.Join("tmp", "subdir", "abcd")), a2.Path)
	})

	t.Run("artifact path filenames are allowed to contain double dots", func(t *testing.T) {
		a1 := Artifact{Name: "a1", Path: "/tmp/..artifact.txt"}
		err := a1.CleanPath()
		assert.NoError(t, err)
		assert.Equal(t, ensureRootPathSeparator(filepath.Join("tmp", "..artifact.txt")), a1.Path)

		a2 := Artifact{Name: "a2", Path: "/tmp/artif..t.txt"}
		err = a2.CleanPath()
		assert.NoError(t, err)
		assert.Equal(t, ensureRootPathSeparator(filepath.Join("tmp", "artif..t.txt")), a2.Path)
	})

	t.Run("normal artifact path succeeds", func(t *testing.T) {
		a1 := Artifact{Name: "a1", Path: "/tmp"}
		err := a1.CleanPath()
		assert.NoError(t, err)
		assert.Equal(t, ensureRootPathSeparator("tmp"), a1.Path)

		a2 := Artifact{Name: "a2", Path: "/tmp/"}
		err = a2.CleanPath()
		assert.NoError(t, err)
		assert.Equal(t, ensureRootPathSeparator("tmp"), a2.Path)

		a3 := Artifact{Name: "a3", Path: "/tmp/abcd/some-artifact.txt"}
		err = a3.CleanPath()
		assert.NoError(t, err)
		assert.Equal(t, ensureRootPathSeparator(filepath.Join("tmp", "abcd", "some-artifact.txt")), a3.Path)
	})
}

func ensureRootPathSeparator(p string) string {
	if p[0] == os.PathSeparator {
		return p
	}
	return string(os.PathSeparator) + p
}

func TestArtifactLocation_IsArchiveLogs(t *testing.T) {
	var l *ArtifactLocation
	assert.False(t, l.IsArchiveLogs())
	assert.False(t, (&ArtifactLocation{}).IsArchiveLogs())
	assert.False(t, (&ArtifactLocation{ArchiveLogs: pointer.BoolPtr(false)}).IsArchiveLogs())
	assert.True(t, (&ArtifactLocation{ArchiveLogs: pointer.BoolPtr(true)}).IsArchiveLogs())
}

func TestArtifactLocation_HasLocation(t *testing.T) {
	var l *ArtifactLocation
	assert.False(t, l.HasLocation(), "Nil")
}

func TestArtifactoryArtifact(t *testing.T) {
	a := &ArtifactoryArtifact{URL: "http://my-host"}
	assert.False(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "http://my-host/my-key", a.URL)
	assert.Equal(t, "/my-key", key, "has leading slash")
}

func TestAzureArtifact(t *testing.T) {
	a := &AzureArtifact{Blob: "my-blob", AzureBlobContainer: AzureBlobContainer{Endpoint: "my-endpoint", Container: "my-container"}}
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-blob"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "my-blob", key)
}

func TestGitArtifact(t *testing.T) {
	a := &GitArtifact{Repo: "my-repo"}
	assert.True(t, a.HasLocation())
	assert.Error(t, a.SetKey("my-key"))
	_, err := a.GetKey()
	assert.Error(t, err)
}

func TestGCSArtifact(t *testing.T) {
	a := &GCSArtifact{Key: "my-key", GCSBucket: GCSBucket{Bucket: "my-bucket"}}
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "my-key", key)
}

func TestHDFSArtifact(t *testing.T) {
	a := &HDFSArtifact{HDFSConfig: HDFSConfig{Addresses: []string{"my-address"}}}
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "my-key", a.Path)
	assert.Equal(t, "my-key", key)
}

func TestHTTPArtifact(t *testing.T) {
	a := &HTTPArtifact{URL: "http://my-host"}
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "http://my-host/my-key", a.URL)
	assert.Equal(t, "/my-key", key, "has leading slack")
}

func TestOSSArtifact(t *testing.T) {
	a := &OSSArtifact{Key: "my-key", OSSBucket: OSSBucket{Endpoint: "my-endpoint", Bucket: "my-bucket"}}
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "my-key", key)
}

func TestRawArtifact(t *testing.T) {
	a := &RawArtifact{Data: "my-data"}
	assert.True(t, a.HasLocation())
	assert.Error(t, a.SetKey("my-key"))
	_, err := a.GetKey()
	assert.Error(t, err)
}

func TestS3Artifact(t *testing.T) {
	a := &S3Artifact{Key: "my-key", S3Bucket: S3Bucket{Endpoint: "my-endpoint", Bucket: "my-bucket"}}
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "my-key", key)
}

func TestArtifactLocation_Relocate(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		var l *ArtifactLocation
		assert.EqualError(t, l.Relocate(nil), "template artifact location not set")
		assert.Error(t, l.Relocate(&ArtifactLocation{}))
		assert.Error(t, (&ArtifactLocation{}).Relocate(nil))
		assert.Error(t, (&ArtifactLocation{}).Relocate(&ArtifactLocation{}))
		assert.Error(t, (&ArtifactLocation{}).Relocate(&ArtifactLocation{S3: &S3Artifact{}}))
		assert.Error(t, (&ArtifactLocation{S3: &S3Artifact{}}).Relocate(&ArtifactLocation{}))
	})
	t.Run("HasLocation", func(t *testing.T) {
		l := &ArtifactLocation{S3: &S3Artifact{S3Bucket: S3Bucket{Bucket: "my-bucket", Endpoint: "my-endpoint"}, Key: "my-key"}}
		assert.NoError(t, l.Relocate(&ArtifactLocation{S3: &S3Artifact{S3Bucket: S3Bucket{Bucket: "other-bucket"}}}))
		assert.Equal(t, "my-endpoint", l.S3.Endpoint, "endpoint is unchanged")
		assert.Equal(t, "my-bucket", l.S3.Bucket, "bucket is unchanged")
		assert.Equal(t, "my-key", l.S3.Key, "key is unchanged")
	})
	t.Run("NotHasLocation", func(t *testing.T) {
		l := &ArtifactLocation{S3: &S3Artifact{Key: "my-key"}}
		assert.NoError(t, l.Relocate(&ArtifactLocation{S3: &S3Artifact{S3Bucket: S3Bucket{Bucket: "my-bucket"}, Key: "other-key"}}))
		assert.Equal(t, "my-bucket", l.S3.Bucket, "bucket copied from argument")
		assert.Equal(t, "my-key", l.S3.Key, "key is unchanged")
	})
}

func TestArtifactLocation_Get(t *testing.T) {
	var l *ArtifactLocation

	v, err := l.Get()
	assert.Nil(t, v)
	assert.EqualError(t, err, "key unsupported: cannot get key for artifact location, because it is invalid")

	v, err = (&ArtifactLocation{}).Get()
	assert.Nil(t, v)
	assert.EqualError(t, err, "You need to configure artifact storage. More information on how to do this can be found in the docs: https://argo-workflows.readthedocs.io/en/release-3.5/configure-artifact-repository/")

	v, _ = (&ArtifactLocation{Azure: &AzureArtifact{}}).Get()
	assert.IsType(t, &AzureArtifact{}, v)

	v, _ = (&ArtifactLocation{Git: &GitArtifact{}}).Get()
	assert.IsType(t, &GitArtifact{}, v)

	v, _ = (&ArtifactLocation{GCS: &GCSArtifact{}}).Get()
	assert.IsType(t, &GCSArtifact{}, v)

	v, _ = (&ArtifactLocation{HDFS: &HDFSArtifact{}}).Get()
	assert.IsType(t, &HDFSArtifact{}, v)

	v, _ = (&ArtifactLocation{HTTP: &HTTPArtifact{}}).Get()
	assert.IsType(t, &HTTPArtifact{}, v)

	v, _ = (&ArtifactLocation{OSS: &OSSArtifact{}}).Get()
	assert.IsType(t, &OSSArtifact{}, v)

	v, _ = (&ArtifactLocation{Raw: &RawArtifact{}}).Get()
	assert.IsType(t, &RawArtifact{}, v)

	v, _ = (&ArtifactLocation{S3: &S3Artifact{}}).Get()
	assert.IsType(t, &S3Artifact{}, v)
}

func TestArtifactLocation_SetType(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.Error(t, l.SetType(nil))
	})
	t.Run("Artifactory", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&ArtifactoryArtifact{}))
		assert.NotNil(t, l.Artifactory)
	})
	t.Run("Azure", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&AzureArtifact{}))
		assert.NotNil(t, l.Azure)
	})
	t.Run("GCS", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&GCSArtifact{}))
		assert.NotNil(t, l.GCS)
	})
	t.Run("HDFS", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&HDFSArtifact{}))
		assert.NotNil(t, l.HDFS)
	})
	t.Run("HTTP", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&HTTPArtifact{}))
		assert.NotNil(t, l.HTTP)
	})
	t.Run("OSS", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&OSSArtifact{}))
		assert.NotNil(t, l.OSS)
	})
	t.Run("Raw", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&RawArtifact{}))
		assert.NotNil(t, l.Raw)
	})
	t.Run("S3", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&S3Artifact{}))
		assert.NotNil(t, l.S3)
	})
	t.Run("Azure", func(t *testing.T) {
		l := &ArtifactLocation{}
		assert.NoError(t, l.SetType(&AzureArtifact{}))
		assert.NotNil(t, l.Azure)
	})
}

func TestArtifactLocation_Key(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		var l *ArtifactLocation
		assert.False(t, l.HasKey())
		_, err := l.GetKey()
		assert.Error(t, err, "cannot get nil")
		err = l.SetKey("my-file")
		assert.Error(t, err, "cannot set nil")
	})
	t.Run("Empty", func(t *testing.T) {
		// unlike nil, empty is actually invalid
		l := &ArtifactLocation{}
		assert.False(t, l.HasKey())
		_, err := l.GetKey()
		assert.Error(t, err, "cannot get empty")
		err = l.SetKey("my-file")
		assert.Error(t, err, "cannot set empty")
	})
	t.Run("Artifactory", func(t *testing.T) {
		l := &ArtifactLocation{Artifactory: &ArtifactoryArtifact{URL: "http://my-host/my-dir?a=1"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "http://my-host/my-dir/my-file?a=1", l.Artifactory.URL, "appends to Artifactory path")
	})
	t.Run("Azure", func(t *testing.T) {
		l := &ArtifactLocation{Azure: &AzureArtifact{Blob: "my-dir"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "my-dir/my-file", l.Azure.Blob, "appends to Azure Blob name")
	})
	t.Run("Git", func(t *testing.T) {
		l := &ArtifactLocation{Git: &GitArtifact{}}
		assert.False(t, l.HasKey())
		_, err := l.GetKey()
		assert.Error(t, err)
		err = l.SetKey("my-file")
		assert.Error(t, err, "cannot set Git key")
	})
	t.Run("GCS", func(t *testing.T) {
		l := &ArtifactLocation{GCS: &GCSArtifact{Key: "my-dir"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "my-dir/my-file", l.GCS.Key, "appends to GCS key")
	})
	t.Run("HDFS", func(t *testing.T) {
		l := &ArtifactLocation{HDFS: &HDFSArtifact{Path: "my-path"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "my-path/my-file", l.HDFS.Path, "appends to HDFS path")
	})
	t.Run("HTTP", func(t *testing.T) {
		l := &ArtifactLocation{HTTP: &HTTPArtifact{URL: "http://my-host/my-dir?a=1"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "http://my-host/my-dir/my-file?a=1", l.HTTP.URL, "appends to HTTP URL path")
	})
	t.Run("OSS", func(t *testing.T) {
		l := &ArtifactLocation{OSS: &OSSArtifact{Key: "my-dir"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "my-dir/my-file", l.OSS.Key, "appends to OSS key")
	})
	t.Run("Raw", func(t *testing.T) {
		l := &ArtifactLocation{Raw: &RawArtifact{}}
		assert.False(t, l.HasKey())
		_, err := l.GetKey()
		assert.Error(t, err, "cannot get raw key")
		err = l.SetKey("my-file")
		assert.Error(t, err, "cannot set raw key")
	})
	t.Run("S3", func(t *testing.T) {
		l := &ArtifactLocation{S3: &S3Artifact{Key: "my-dir"}}
		err := l.AppendToKey("my-file")
		assert.NoError(t, err)
		assert.Equal(t, "my-dir/my-file", l.S3.Key, "appends to S3 key")
	})
}

func TestArtifactRepositoryRef_GetConfigMapOr(t *testing.T) {
	var r *ArtifactRepositoryRef
	assert.Equal(t, "my-cm", r.GetConfigMapOr("my-cm"))
	assert.Equal(t, "my-cm", (&ArtifactRepositoryRef{}).GetConfigMapOr("my-cm"))
	assert.Equal(t, "my-cm", (&ArtifactRepositoryRef{ConfigMap: "my-cm"}).GetConfigMapOr(""))
}

func TestArtifactRepositoryRef_GetKeyOr(t *testing.T) {
	var r *ArtifactRepositoryRef
	assert.Equal(t, "", r.GetKeyOr(""))
	assert.Equal(t, "my-key", (&ArtifactRepositoryRef{}).GetKeyOr("my-key"))
	assert.Equal(t, "my-key", (&ArtifactRepositoryRef{Key: "my-key"}).GetKeyOr(""))
}

func TestArtifactRepositoryRef_String(t *testing.T) {
	var l *ArtifactRepositoryRef
	assert.Equal(t, "nil", l.String())
	assert.Equal(t, "#", (&ArtifactRepositoryRef{}).String())
	assert.Equal(t, "my-cm#my-key", (&ArtifactRepositoryRef{ConfigMap: "my-cm", Key: "my-key"}).String())
}

func TestArtifactRepositoryRefStatus_String(t *testing.T) {
	var l *ArtifactRepositoryRefStatus
	assert.Equal(t, "nil", l.String())
	assert.Equal(t, "/#", (&ArtifactRepositoryRefStatus{}).String())
	assert.Equal(t, "default-artifact-repository", (&ArtifactRepositoryRefStatus{Default: true}).String())
	assert.Equal(t, "my-ns/my-cm#my-key", (&ArtifactRepositoryRefStatus{Namespace: "my-ns", ArtifactRepositoryRef: ArtifactRepositoryRef{ConfigMap: "my-cm", Key: "my-key"}}).String())
}

func TestArtifact_GetArchive(t *testing.T) {
	assert.NotNil(t, (&Artifact{}).GetArchive())
	assert.Equal(t, &ArchiveStrategy{None: &NoneStrategy{}}, (&Artifact{Archive: &ArchiveStrategy{None: &NoneStrategy{}}}).GetArchive())
}

func TestArtifactGC_GetStrategy(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		var artifactGC *ArtifactGC
		assert.Equal(t, ArtifactGCStrategyUndefined, artifactGC.GetStrategy())
	})
	t.Run("Unspecified", func(t *testing.T) {
		var artifactGC = &ArtifactGC{}
		assert.Equal(t, ArtifactGCStrategyUndefined, artifactGC.GetStrategy())
	})
	t.Run("Specified", func(t *testing.T) {
		var artifactGC = &ArtifactGC{Strategy: ArtifactGCOnWorkflowCompletion}
		assert.Equal(t, ArtifactGCOnWorkflowCompletion, artifactGC.GetStrategy())
	})
}

func TestPodGCStrategy_IsValid(t *testing.T) {
	for _, s := range []PodGCStrategy{
		PodGCOnPodNone,
		PodGCOnPodCompletion,
		PodGCOnPodSuccess,
		PodGCOnWorkflowCompletion,
		PodGCOnWorkflowSuccess,
	} {
		t.Run(string(s), func(t *testing.T) {
			assert.True(t, s.IsValid())
		})
	}
	t.Run("Invalid", func(t *testing.T) {
		assert.False(t, PodGCStrategy("Foo").IsValid())
	})
}

func TestPodGC_GetStrategy(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		var podGC *PodGC
		assert.Equal(t, PodGCOnPodNone, podGC.GetStrategy())
	})
	t.Run("Unspecified", func(t *testing.T) {
		var podGC = &PodGC{}
		assert.Equal(t, PodGCOnPodNone, podGC.GetStrategy())
	})
	t.Run("Specified", func(t *testing.T) {
		var podGC = &PodGC{Strategy: PodGCOnWorkflowSuccess}
		assert.Equal(t, PodGCOnWorkflowSuccess, podGC.GetStrategy())
	})
}

func TestPodGC_GetLabelSelector(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		var podGC *PodGC
		selector, err := podGC.GetLabelSelector()
		assert.NoError(t, err)
		assert.Equal(t, labels.Nothing(), selector)
	})
	t.Run("Unspecified", func(t *testing.T) {
		var podGC = &PodGC{}
		selector, err := podGC.GetLabelSelector()
		assert.NoError(t, err)
		assert.Equal(t, labels.Everything(), selector)
	})
	t.Run("Specified", func(t *testing.T) {
		labelSelector := &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"foo": "bar",
			},
		}
		var podGC = &PodGC{LabelSelector: labelSelector}
		selector, err := podGC.GetLabelSelector()
		assert.NoError(t, err)
		assert.Equal(t, "foo=bar", selector.String())
	})
}

func TestNodes_FindByDisplayName(t *testing.T) {
	assert.Nil(t, Nodes{}.FindByDisplayName(""))
	assert.NotNil(t, Nodes{"": NodeStatus{DisplayName: "foo"}}.FindByDisplayName("foo"))
}

func TestNodes_Any(t *testing.T) {
	assert.False(t, Nodes{"": NodeStatus{Name: "foo"}}.Any(func(node NodeStatus) bool { return node.Name == "bar" }))
	assert.True(t, Nodes{"": NodeStatus{Name: "foo"}}.Any(func(node NodeStatus) bool { return node.Name == "foo" }))
}

func TestNodes_Children(t *testing.T) {
	nodes := Nodes{
		"node_0": NodeStatus{Name: "node_0", Phase: NodeFailed, Children: []string{"node_1", "node_2"}},
		"node_1": NodeStatus{Name: "node_1", Phase: NodeFailed, Children: []string{}},
		"node_2": NodeStatus{Name: "node_2", Phase: NodeRunning, Children: []string{}},
	}
	t.Run("Found", func(t *testing.T) {
		ret := nodes.Children("node_0")
		assert.Equal(t, len(ret), 2)
		assert.Equal(t, ret["node_1"].Name, "node_1")
		assert.Equal(t, ret["node_2"].Name, "node_2")
	})
	t.Run("NotFound", func(t *testing.T) {
		assert.Empty(t, nodes.Children("node_1"))
	})
	t.Run("Empty", func(t *testing.T) {
		assert.Empty(t, Nodes{}.Children("node_1"))
	})
}

func TestNestedChildren(t *testing.T) {
	nodes := Nodes{
		"node_0": NodeStatus{Name: "node_0", Phase: NodeFailed, Children: []string{"node_1", "node_2"}},
		"node_1": NodeStatus{Name: "node_1", Phase: NodeFailed, Children: []string{"node_3"}},
		"node_2": NodeStatus{Name: "node_2", Phase: NodeRunning, Children: []string{}},
		"node_3": NodeStatus{Name: "node_3", Phase: NodeRunning, Children: []string{"node_4"}},
		"node_4": NodeStatus{Name: "node_4", Phase: NodeRunning, Children: []string{}},
	}
	t.Run("Get children", func(t *testing.T) {
		statuses, err := nodes.NestedChildrenStatus("node_0")
		assert.NoError(t, err)
		found := make(map[string]bool)
		// parent is already assumed to be found
		found["node_0"] = true
		for _, child := range statuses {
			_, ok := found[child.Name]
			assert.False(t, ok, "got %s", child.Name)
			found[child.Name] = true
		}
		assert.Equal(t, len(nodes), len(found))
	})
}

func TestNodes_Filter(t *testing.T) {
	nodes := Nodes{
		"node_1": NodeStatus{ID: "node_1", Phase: NodeFailed},
		"node_2": NodeStatus{ID: "node_2", Phase: NodeRunning},
		"node_3": NodeStatus{ID: "node_3", Phase: NodeFailed},
	}
	t.Run("Empty", func(t *testing.T) {
		assert.Empty(t, Nodes{}.Filter(func(x NodeStatus) bool { return x.Phase == NodeError }))
	})
	t.Run("NotFound", func(t *testing.T) {
		assert.Empty(t, nodes.Filter(func(x NodeStatus) bool { return x.Phase == NodeError }))
	})
	t.Run("Found", func(t *testing.T) {
		n := nodes.Filter(func(x NodeStatus) bool { return x.Phase == NodeFailed })
		assert.Equal(t, len(n), 2)
		assert.Equal(t, n["node_1"].ID, "node_1")
		assert.Equal(t, n["node_3"].ID, "node_3")
	})
}

// Map(f func(x NodeStatus) interface{}) map[string]interface{} {
func TestNodes_Map(t *testing.T) {
	nodes := Nodes{
		"node_1": NodeStatus{ID: "node_1", HostNodeName: "host_1"},
		"node_2": NodeStatus{ID: "node_2", HostNodeName: "host_2"},
	}
	t.Run("Empty", func(t *testing.T) {
		assert.Empty(t, Nodes{}.Map(func(x NodeStatus) interface{} { return x.HostNodeName }))
	})
	t.Run("Exist", func(t *testing.T) {
		n := nodes.Map(func(x NodeStatus) interface{} { return x.HostNodeName })
		assert.Equal(t, n["node_1"], "host_1")
		assert.Equal(t, n["node_2"], "host_2")
	})
}

// TestInputs_NoArtifacts makes sure that the code doesn't panic when trying to get artifacts from a node status
// without any artifacts
func TestInputs_NoArtifacts(t *testing.T) {
	s := NodeStatus{ID: "node_1", Inputs: nil, Outputs: nil}
	inArt := s.Inputs.GetArtifactByName("test-artifact")
	assert.Nil(t, inArt)
	outArt := s.Outputs.GetArtifactByName("test-artifact")
	assert.Nil(t, outArt)
}

func TestResourcesDuration_String(t *testing.T) {
	assert.Empty(t, ResourcesDuration{}.String(), "empty")
	assert.Equal(t, "1s*(100Mi memory)", ResourcesDuration{corev1.ResourceMemory: NewResourceDuration(1 * time.Second)}.String(), "memory")
}

func TestResourcesDuration_Add(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		assert.Empty(t, ResourcesDuration{}.Add(ResourcesDuration{}))
	})
	t.Run("X+Empty", func(t *testing.T) {
		s := ResourcesDuration{"x": NewResourceDuration(time.Second)}.
			Add(nil)
		assert.Equal(t, ResourceDuration(1), s["x"])
	})
	t.Run("Empty+X", func(t *testing.T) {
		s := ResourcesDuration{}.
			Add(ResourcesDuration{"x": NewResourceDuration(time.Second)})
		assert.Equal(t, ResourceDuration(1), s["x"])
	})
	t.Run("X+2X", func(t *testing.T) {
		s := ResourcesDuration{"x": NewResourceDuration(1 * time.Second)}.
			Add(ResourcesDuration{"x": NewResourceDuration(2 * time.Second)})
		assert.Equal(t, ResourceDuration(3), s["x"])
	})
	t.Run("X+Y", func(t *testing.T) {
		s := ResourcesDuration{"x": NewResourceDuration(1 * time.Second)}.
			Add(ResourcesDuration{"y": NewResourceDuration(2 * time.Second)})
		assert.Equal(t, ResourceDuration(1), s["x"])
		assert.Equal(t, ResourceDuration(2), s["y"])
	})
}

func TestResourceDuration(t *testing.T) {
	assert.Equal(t, ResourceDuration(1), NewResourceDuration(1*time.Second))
	assert.Equal(t, "1s", NewResourceDuration(1*time.Second).String())
}

func TestWorkflowConditions_UpsertConditionMessage(t *testing.T) {
	wfCond := Conditions{Condition{Type: ConditionTypeCompleted, Message: "Hello"}}
	wfCond.UpsertConditionMessage(Condition{Type: ConditionTypeCompleted, Message: "world!"})
	assert.Equal(t, "Hello, world!", wfCond[0].Message)
}

func TestShutdownStrategy_ShouldExecute(t *testing.T) {
	assert.False(t, ShutdownStrategyTerminate.ShouldExecute(true))
	assert.False(t, ShutdownStrategyTerminate.ShouldExecute(false))
	assert.False(t, ShutdownStrategyStop.ShouldExecute(false))
	assert.True(t, ShutdownStrategyStop.ShouldExecute(true))
}

func TestCronWorkflowConditions(t *testing.T) {
	cwfCond := Conditions{}
	cond := Condition{
		Type:    ConditionTypeSubmissionError,
		Message: "Failed to submit Workflow",
		Status:  metav1.ConditionTrue,
	}

	assert.Len(t, cwfCond, 0)
	cwfCond.UpsertCondition(cond)
	assert.Len(t, cwfCond, 1)
	cwfCond.RemoveCondition(ConditionTypeSubmissionError)
	assert.Len(t, cwfCond, 0)
}

func TestDisplayConditions(t *testing.T) {
	const fmtStr = "%-20s %v\n"
	cwfCond := Conditions{}

	assert.Equal(t, "Conditions:          None\n", cwfCond.DisplayString(fmtStr, nil))

	cond := Condition{
		Type:    ConditionTypeSubmissionError,
		Message: "Failed to submit Workflow",
		Status:  metav1.ConditionTrue,
	}
	cwfCond.UpsertCondition(cond)

	expected := `Conditions:          
✖ SubmissionError    Failed to submit Workflow
`
	assert.Equal(t, expected, cwfCond.DisplayString(fmtStr, map[ConditionType]string{ConditionTypeSubmissionError: "✖"}))
}

func TestPrometheus_GetDescIsStable(t *testing.T) {
	metric := &Prometheus{
		Name: "test-metric",
		Labels: []*MetricLabel{
			{Key: "foo", Value: "bar"},
			{Key: "hello", Value: "World"},
		},
		Histogram: &Histogram{
			Buckets: []Amount{{"10"}, {"20"}, {"30"}},
		},
	}
	stableDesc := metric.GetDesc()
	for i := 0; i < 10; i++ {
		if !assert.Equal(t, stableDesc, metric.GetDesc()) {
			break
		}
	}
}

func TestWorkflow_SearchArtifacts(t *testing.T) {
	wf := Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: WorkflowSpec{
			ArtifactGC: &WorkflowLevelArtifactGC{
				ArtifactGC: ArtifactGC{
					Strategy: ArtifactGCOnWorkflowCompletion,
				},
			},
			Templates: []Template{
				{
					Name: "template-foo",
					Outputs: Outputs{
						Artifacts: Artifacts{
							Artifact{Name: "artifact-foo"},
							Artifact{Name: "artifact-bar", ArtifactGC: &ArtifactGC{Strategy: ArtifactGCOnWorkflowDeletion}},
						},
					},
				},
				{
					Name: "template-bar",
					Outputs: Outputs{
						Artifacts: Artifacts{
							Artifact{Name: "artifact-foobar"},
						},
					},
				},
			},
		},
		Status: WorkflowStatus{
			Nodes: Nodes{
				"test-foo": NodeStatus{
					ID:           "node-foo",
					TemplateName: "template-foo",
					Outputs: &Outputs{
						Artifacts: Artifacts{
							Artifact{Name: "artifact-foo"},
							Artifact{Name: "artifact-bar", ArtifactGC: &ArtifactGC{Strategy: ArtifactGCOnWorkflowDeletion}},
						},
					},
				},
				"test-bar": NodeStatus{
					ID:           "node-bar",
					TemplateName: "template-bar",
					Outputs: &Outputs{
						Artifacts: Artifacts{
							Artifact{Name: "artifact-foobar"},
						},
					},
				},
			},
		},
	}

	query := NewArtifactSearchQuery()

	countArtifactName := func(ars ArtifactSearchResults, name string) int {
		count := 0
		for _, ar := range ars {
			if ar.Artifact.Name == name {
				count++
			}
		}
		return count
	}
	countNodeID := func(ars ArtifactSearchResults, nodeID string) int {
		count := 0
		for _, ar := range ars {
			if ar.NodeID == nodeID {
				count++
			}
		}
		return count
	}

	// no filters
	queriedArtifactSearchResults := wf.SearchArtifacts(query)
	assert.NotNil(t, queriedArtifactSearchResults)
	assert.Len(t, queriedArtifactSearchResults, 3)
	assert.Equal(t, 1, countArtifactName(queriedArtifactSearchResults, "artifact-foo"))
	assert.Equal(t, 1, countArtifactName(queriedArtifactSearchResults, "artifact-bar"))
	assert.Equal(t, 1, countArtifactName(queriedArtifactSearchResults, "artifact-foobar"))
	assert.Equal(t, 2, countNodeID(queriedArtifactSearchResults, "node-foo"))
	assert.Equal(t, 1, countNodeID(queriedArtifactSearchResults, "node-bar"))

	// artifactGC strategy: OnWorkflowCompletion
	query.ArtifactGCStrategies[ArtifactGCOnWorkflowCompletion] = true
	queriedArtifactSearchResults = wf.SearchArtifacts(query)
	assert.NotNil(t, queriedArtifactSearchResults)
	assert.Len(t, queriedArtifactSearchResults, 2)
	assert.Equal(t, 1, countArtifactName(queriedArtifactSearchResults, "artifact-foo"))
	assert.Equal(t, 0, countArtifactName(queriedArtifactSearchResults, "artifact-bar"))
	assert.Equal(t, 1, countArtifactName(queriedArtifactSearchResults, "artifact-foobar"))
	assert.Equal(t, 1, countNodeID(queriedArtifactSearchResults, "node-foo"))
	assert.Equal(t, 1, countNodeID(queriedArtifactSearchResults, "node-bar"))

	// artifactGC strategy: OnWorkflowDeletion
	query = NewArtifactSearchQuery()
	query.ArtifactGCStrategies[ArtifactGCOnWorkflowDeletion] = true
	queriedArtifactSearchResults = wf.SearchArtifacts(query)
	assert.NotNil(t, queriedArtifactSearchResults)
	assert.Len(t, queriedArtifactSearchResults, 1)
	assert.Equal(t, 0, countArtifactName(queriedArtifactSearchResults, "artifact-foo"))
	assert.Equal(t, 1, countArtifactName(queriedArtifactSearchResults, "artifact-bar"))
	assert.Equal(t, 0, countArtifactName(queriedArtifactSearchResults, "artifact-foobar"))
	assert.Equal(t, 1, countNodeID(queriedArtifactSearchResults, "node-foo"))
	assert.Equal(t, 0, countNodeID(queriedArtifactSearchResults, "node-bar"))

	// template name
	query = NewArtifactSearchQuery()
	query.TemplateName = "template-bar"
	queriedArtifactSearchResults = wf.SearchArtifacts(query)
	assert.NotNil(t, queriedArtifactSearchResults)
	assert.Len(t, queriedArtifactSearchResults, 1)
	assert.Equal(t, "artifact-foobar", queriedArtifactSearchResults[0].Artifact.Name)
	assert.Equal(t, "node-bar", queriedArtifactSearchResults[0].NodeID)

	// artifact name
	query = NewArtifactSearchQuery()
	query.ArtifactName = "artifact-foo"
	queriedArtifactSearchResults = wf.SearchArtifacts(query)
	assert.NotNil(t, queriedArtifactSearchResults)
	assert.Len(t, queriedArtifactSearchResults, 1)
	assert.Equal(t, "artifact-foo", queriedArtifactSearchResults[0].Artifact.Name)
	assert.Equal(t, "node-foo", queriedArtifactSearchResults[0].NodeID)

	// node id
	query = NewArtifactSearchQuery()
	query.NodeId = "node-foo"
	queriedArtifactSearchResults = wf.SearchArtifacts(query)
	assert.NotNil(t, queriedArtifactSearchResults)
	assert.Len(t, queriedArtifactSearchResults, 2)
	assert.Equal(t, 1, countArtifactName(queriedArtifactSearchResults, "artifact-foo"))
	assert.Equal(t, 1, countArtifactName(queriedArtifactSearchResults, "artifact-bar"))
	assert.Equal(t, 2, countNodeID(queriedArtifactSearchResults, "node-foo"))

	// bad query
	query = NewArtifactSearchQuery()
	query.NodeId = "node-foobar"
	queriedArtifactSearchResults = wf.SearchArtifacts(query)
	assert.Nil(t, queriedArtifactSearchResults)
	assert.Len(t, queriedArtifactSearchResults, 0)

	// template and artifact name
	query = NewArtifactSearchQuery()
	query.TemplateName = "template-foo"
	query.ArtifactName = "artifact-foo"
	queriedArtifactSearchResults = wf.SearchArtifacts(query)
	assert.NotNil(t, queriedArtifactSearchResults)
	assert.Len(t, queriedArtifactSearchResults, 1)
	assert.Equal(t, "artifact-foo", queriedArtifactSearchResults[0].Artifact.Name)
	assert.Equal(t, "node-foo", queriedArtifactSearchResults[0].NodeID)
}

func TestWorkflowSpec_GetArtifactGC(t *testing.T) {
	spec := WorkflowSpec{}

	assert.NotNil(t, spec.GetArtifactGC())
	assert.Equal(t, &ArtifactGC{Strategy: ArtifactGCStrategyUndefined}, spec.GetArtifactGC())
}

func TestWorkflowSpec_GetVolumeGC(t *testing.T) {
	spec := WorkflowSpec{}

	assert.NotNil(t, spec.GetVolumeClaimGC())
	assert.Equal(t, &VolumeClaimGC{Strategy: VolumeClaimGCOnSuccess}, spec.GetVolumeClaimGC())
}

func TestGetTTLStrategy(t *testing.T) {
	spec := WorkflowSpec{TTLStrategy: &TTLStrategy{SecondsAfterCompletion: pointer.Int32Ptr(20)}}
	ttl := spec.GetTTLStrategy()
	assert.Equal(t, int32(20), *ttl.SecondsAfterCompletion)
}

func TestWfGetTTLStrategy(t *testing.T) {
	wf := Workflow{}

	wf.Status.StoredWorkflowSpec = &WorkflowSpec{TTLStrategy: &TTLStrategy{SecondsAfterCompletion: pointer.Int32Ptr(20)}}
	result := wf.GetTTLStrategy()
	assert.Equal(t, int32(20), *result.SecondsAfterCompletion)

	wf.Spec.TTLStrategy = &TTLStrategy{SecondsAfterCompletion: pointer.Int32Ptr(30)}
	result = wf.GetTTLStrategy()
	assert.Equal(t, int32(30), *result.SecondsAfterCompletion)
}

func TestWorkflow_GetSemaphoreKeys(t *testing.T) {
	assert := assert.New(t)
	wf := Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: WorkflowSpec{
			Synchronization: &Synchronization{
				Semaphore: &SemaphoreRef{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test",
					},
				}},
			},
		},
	}
	keys := wf.GetSemaphoreKeys()
	assert.Len(keys, 1)
	assert.Contains(keys, "test/test")
	wf.Spec.Templates = []Template{
		{
			Name: "t1",
			Synchronization: &Synchronization{
				Semaphore: &SemaphoreRef{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "template",
					},
				}},
			},
		},
		{
			Name: "t1",
			Synchronization: &Synchronization{
				Semaphore: &SemaphoreRef{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "template1",
					},
				}},
			},
		},
		{
			Name: "t2",
			Synchronization: &Synchronization{
				Semaphore: &SemaphoreRef{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "template",
					},
				}},
			},
		},
	}
	keys = wf.GetSemaphoreKeys()
	assert.Len(keys, 3)
	assert.Contains(keys, "test/test")
	assert.Contains(keys, "test/template")
	assert.Contains(keys, "test/template1")

	spec := wf.Spec.DeepCopy()
	wf.Spec = WorkflowSpec{
		WorkflowTemplateRef: &WorkflowTemplateRef{
			Name: "test",
		},
	}
	wf.Status.StoredWorkflowSpec = spec
	keys = wf.GetSemaphoreKeys()
	assert.Len(keys, 3)
	assert.Contains(keys, "test/test")
	assert.Contains(keys, "test/template")
	assert.Contains(keys, "test/template1")
}

func TestTemplate_IsMainContainerNamed(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		x := &Template{}
		assert.True(t, x.IsMainContainerName("main"))
	})
	t.Run("ContainerSet", func(t *testing.T) {
		x := &Template{ContainerSet: &ContainerSetTemplate{Containers: []ContainerNode{{Container: corev1.Container{Name: "foo"}}}}}
		assert.False(t, x.IsMainContainerName("main"))
		assert.True(t, x.IsMainContainerName("foo"))
	})
}

func TestTemplate_GetMainContainer(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		x := &Template{}
		assert.Equal(t, []string{"main"}, x.GetMainContainerNames())
	})
	t.Run("ContainerSet", func(t *testing.T) {
		x := &Template{ContainerSet: &ContainerSetTemplate{Containers: []ContainerNode{{Container: corev1.Container{Name: "foo"}}}}}
		assert.Equal(t, []string{"foo"}, x.GetMainContainerNames())
	})
}

func TestTemplate_HasSequencedContainers(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		x := &Template{}
		assert.False(t, x.HasSequencedContainers())
	})
	t.Run("ContainerSet", func(t *testing.T) {
		x := &Template{ContainerSet: &ContainerSetTemplate{Containers: []ContainerNode{{Dependencies: []string{""}}}}}
		assert.True(t, x.HasSequencedContainers())
	})
}

func TestTemplate_GetVolumeMounts(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		x := &Template{}
		assert.Empty(t, x.GetVolumeMounts())
	})
	t.Run("Container", func(t *testing.T) {
		x := &Template{Container: &corev1.Container{VolumeMounts: []corev1.VolumeMount{{}}}}
		assert.NotEmpty(t, x.GetVolumeMounts())
	})
	t.Run("ContainerSet", func(t *testing.T) {
		x := &Template{ContainerSet: &ContainerSetTemplate{VolumeMounts: []corev1.VolumeMount{{}}}}
		assert.NotEmpty(t, x.GetVolumeMounts())
	})
	t.Run("Script", func(t *testing.T) {
		x := &Template{Script: &ScriptTemplate{Container: corev1.Container{VolumeMounts: []corev1.VolumeMount{{}}}}}
		assert.NotEmpty(t, x.GetVolumeMounts())
	})
}

func TestTemplate_HasOutputs(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		x := &Template{}
		assert.False(t, x.HasOutput())
	})
	t.Run("Container", func(t *testing.T) {
		x := &Template{Container: &corev1.Container{}}
		assert.True(t, x.HasOutput())
	})
	t.Run("ContainerSet", func(t *testing.T) {
		t.Run("NoMain", func(t *testing.T) {
			x := &Template{ContainerSet: &ContainerSetTemplate{}}
			assert.False(t, x.HasOutput())
		})
		t.Run("Main", func(t *testing.T) {
			x := &Template{ContainerSet: &ContainerSetTemplate{Containers: []ContainerNode{{Container: corev1.Container{Name: "main"}}}}}
			assert.True(t, x.HasOutput())
		})
	})
	t.Run("Script", func(t *testing.T) {
		x := &Template{Script: &ScriptTemplate{}}
		assert.True(t, x.HasOutput())
	})
	t.Run("Data", func(t *testing.T) {
		x := &Template{Data: &Data{}}
		assert.True(t, x.HasOutput())
	})
	t.Run("Resource", func(t *testing.T) {
		x := &Template{Resource: &ResourceTemplate{}}
		assert.False(t, x.HasOutput())
	})
	t.Run("Plugin", func(t *testing.T) {
		x := &Template{Plugin: &Plugin{}}
		assert.True(t, x.HasOutput())
	})
}

func TestTemplate_SaveLogsAsArtifact(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		x := &Template{}
		assert.False(t, x.SaveLogsAsArtifact())
	})
	t.Run("IsArchiveLogs", func(t *testing.T) {
		x := &Template{ArchiveLocation: &ArtifactLocation{ArchiveLogs: pointer.BoolPtr(true)}}
		assert.True(t, x.SaveLogsAsArtifact())
	})
}

func TestTemplate_ExcludeTemplateTypes(t *testing.T) {
	steps := ParallelSteps{
		[]WorkflowStep{
			{
				Name:     "Test",
				Template: "testtmpl",
			},
		},
	}
	tmpl := Template{
		Name:      "step",
		Steps:     []ParallelSteps{steps},
		Script:    &ScriptTemplate{Source: "test"},
		Container: &corev1.Container{Name: "container"},
		DAG:       &DAGTemplate{FailFast: pointer.BoolPtr(true)},
		Resource:  &ResourceTemplate{Action: "Create"},
		Data:      &Data{Source: DataSource{ArtifactPaths: &ArtifactPaths{}}},
		Suspend:   &SuspendTemplate{Duration: "10s"},
	}

	t.Run("StepTemplateType", func(t *testing.T) {
		stepTmpl := tmpl.DeepCopy()
		stepTmpl.SetType(TemplateTypeSteps)
		assert.NotNil(t, stepTmpl.Steps)
		assert.Nil(t, stepTmpl.Script)
		assert.Nil(t, stepTmpl.Resource)
		assert.Nil(t, stepTmpl.Data)
		assert.Nil(t, stepTmpl.DAG)
		assert.Nil(t, stepTmpl.Container)
		assert.Nil(t, stepTmpl.Suspend)
	})

	t.Run("DAGTemplateType", func(t *testing.T) {
		dagTmpl := tmpl.DeepCopy()
		dagTmpl.SetType(TemplateTypeDAG)
		assert.NotNil(t, dagTmpl.DAG)
		assert.Nil(t, dagTmpl.Script)
		assert.Nil(t, dagTmpl.Resource)
		assert.Nil(t, dagTmpl.Data)
		assert.Len(t, dagTmpl.Steps, 0)
		assert.Nil(t, dagTmpl.Container)
		assert.Nil(t, dagTmpl.Suspend)
	})

	t.Run("ScriptTemplateType", func(t *testing.T) {
		scriptTmpl := tmpl.DeepCopy()
		scriptTmpl.SetType(TemplateTypeScript)
		assert.NotNil(t, scriptTmpl.Script)
		assert.Nil(t, scriptTmpl.DAG)
		assert.Nil(t, scriptTmpl.Resource)
		assert.Nil(t, scriptTmpl.Data)
		assert.Len(t, scriptTmpl.Steps, 0)
		assert.Nil(t, scriptTmpl.Container)
		assert.Nil(t, scriptTmpl.Suspend)
	})

	t.Run("ResourceTemplateType", func(t *testing.T) {
		resourceTmpl := tmpl.DeepCopy()
		resourceTmpl.SetType(TemplateTypeResource)
		assert.NotNil(t, resourceTmpl.Resource)
		assert.Nil(t, resourceTmpl.Script)
		assert.Nil(t, resourceTmpl.DAG)
		assert.Nil(t, resourceTmpl.Data)
		assert.Len(t, resourceTmpl.Steps, 0)
		assert.Nil(t, resourceTmpl.Container)
		assert.Nil(t, resourceTmpl.Suspend)
	})
	t.Run("ContainerTemplateType", func(t *testing.T) {
		containerTmpl := tmpl.DeepCopy()
		containerTmpl.SetType(TemplateTypeContainer)
		assert.NotNil(t, containerTmpl.Container)
		assert.Nil(t, containerTmpl.Script)
		assert.Nil(t, containerTmpl.DAG)
		assert.Nil(t, containerTmpl.Data)
		assert.Len(t, containerTmpl.Steps, 0)
		assert.Nil(t, containerTmpl.Resource)
		assert.Nil(t, containerTmpl.Suspend)
	})
	t.Run("DataTemplateType", func(t *testing.T) {
		dataTmpl := tmpl.DeepCopy()
		dataTmpl.SetType(TemplateTypeData)
		assert.NotNil(t, dataTmpl.Data)
		assert.Nil(t, dataTmpl.Script)
		assert.Nil(t, dataTmpl.DAG)
		assert.Nil(t, dataTmpl.Container)
		assert.Len(t, dataTmpl.Steps, 0)
		assert.Nil(t, dataTmpl.Resource)
		assert.Nil(t, dataTmpl.Suspend)
	})
	t.Run("SuspendTemplateType", func(t *testing.T) {
		suspendTmpl := tmpl.DeepCopy()
		suspendTmpl.SetType(TemplateTypeSuspend)
		assert.NotNil(t, suspendTmpl.Suspend)
		assert.Nil(t, suspendTmpl.Script)
		assert.Nil(t, suspendTmpl.DAG)
		assert.Nil(t, suspendTmpl.Container)
		assert.Len(t, suspendTmpl.Steps, 0)
		assert.Nil(t, suspendTmpl.Resource)
		assert.Nil(t, suspendTmpl.Data)
	})
}

func TestDAGTask_GetExitTemplate(t *testing.T) {
	args := Arguments{
		Parameters: []Parameter{
			{
				Name:  "test",
				Value: AnyStringPtr("welcome"),
			},
		},
	}
	task := DAGTask{
		Hooks: map[LifecycleEvent]LifecycleHook{
			ExitLifecycleEvent: LifecycleHook{
				Template:  "test",
				Arguments: args,
			},
		},
	}
	existTmpl := task.GetExitHook(Arguments{})
	assert.NotNil(t, existTmpl)
	assert.Equal(t, "test", existTmpl.Template)
	assert.Equal(t, args, existTmpl.Arguments)
	task = DAGTask{OnExit: "test-tmpl"}
	existTmpl = task.GetExitHook(args)
	assert.NotNil(t, existTmpl)
	assert.Equal(t, "test-tmpl", existTmpl.Template)
	assert.Equal(t, args, existTmpl.Arguments)
}

func TestStep_GetExitTemplate(t *testing.T) {
	args := Arguments{
		Parameters: []Parameter{
			{
				Name:  "test",
				Value: AnyStringPtr("welcome"),
			},
		},
	}
	task := WorkflowStep{
		Hooks: map[LifecycleEvent]LifecycleHook{
			ExitLifecycleEvent: LifecycleHook{
				Template:  "test",
				Arguments: args,
			},
		},
	}
	existTmpl := task.GetExitHook(Arguments{})
	assert.NotNil(t, existTmpl)
	assert.Equal(t, "test", existTmpl.Template)
	assert.Equal(t, args, existTmpl.Arguments)
	task = WorkflowStep{OnExit: "test-tmpl"}
	existTmpl = task.GetExitHook(args)
	assert.NotNil(t, existTmpl)
	assert.Equal(t, "test-tmpl", existTmpl.Template)
	assert.Equal(t, args, existTmpl.Arguments)
}

func TestHasChild(t *testing.T) {
	node := NodeStatus{
		Children: []string{"a", "b"},
	}
	assert.True(t, node.HasChild("a"))
	assert.False(t, node.HasChild("c"))
	assert.False(t, node.HasChild(""))
}

func TestParameterGetValue(t *testing.T) {
	empty := Parameter{}
	defaultVal := Parameter{Default: AnyStringPtr("Default")}
	value := Parameter{Value: AnyStringPtr("Test")}

	valueFrom := Parameter{ValueFrom: &ValueFrom{}}
	assert.False(t, empty.HasValue())
	assert.Empty(t, empty.GetValue())
	assert.True(t, defaultVal.HasValue())
	assert.NotEmpty(t, defaultVal.GetValue())
	assert.Equal(t, "Default", defaultVal.GetValue())
	assert.True(t, value.HasValue())
	assert.NotEmpty(t, value.GetValue())
	assert.Equal(t, "Test", value.GetValue())
	assert.True(t, valueFrom.HasValue())

}

func TestTemplateIsLeaf(t *testing.T) {
	tmpls := []Template{
		{
			HTTP: &HTTP{URL: "test.com"},
		},
		{
			ContainerSet: &ContainerSetTemplate{},
		},
		{
			Container: &corev1.Container{},
		},
		{
			Script: &ScriptTemplate{},
		},
		{
			Resource: &ResourceTemplate{},
		},
	}
	for _, tmpl := range tmpls {
		assert.True(t, tmpl.IsLeaf())
	}
	tmpl := Template{
		DAG: &DAGTemplate{},
	}
	assert.False(t, tmpl.IsLeaf())
	tmpl = Template{
		Steps: []ParallelSteps{},
	}
	assert.False(t, tmpl.IsLeaf())

}

func TestTemplateGetType(t *testing.T) {
	tmpl := Template{HTTP: &HTTP{}}
	assert.Equal(t, TemplateTypeHTTP, tmpl.GetType())
}

func TestWfSpecGetExitHook(t *testing.T) {
	wfSpec := WorkflowSpec{OnExit: "test"}
	hooks := wfSpec.GetExitHook(wfSpec.Arguments)
	assert.Equal(t, "test", hooks.Template)
	wfSpec = WorkflowSpec{Hooks: LifecycleHooks{"exit": LifecycleHook{Template: "hook"}}}
	hooks = wfSpec.GetExitHook(wfSpec.Arguments)
	assert.Equal(t, "hook", hooks.Template)
}

func TestDagSpecGetExitHook(t *testing.T) {
	dagTask := DAGTask{Name: "A", OnExit: "test"}
	hooks := dagTask.GetExitHook(dagTask.Arguments)
	assert.Equal(t, "test", hooks.Template)
	dagTask = DAGTask{Name: "A", Hooks: LifecycleHooks{"exit": LifecycleHook{Template: "hook"}}}
	hooks = dagTask.GetExitHook(dagTask.Arguments)
	assert.Equal(t, "hook", hooks.Template)
}

func TestStepSpecGetExitHook(t *testing.T) {
	step := WorkflowStep{Name: "A", OnExit: "test"}
	hooks := step.GetExitHook(step.Arguments)
	assert.Equal(t, "test", hooks.Template)
	step = WorkflowStep{Name: "A", Hooks: LifecycleHooks{"exit": LifecycleHook{Template: "hook"}}}
	hooks = step.GetExitHook(step.Arguments)
	assert.Equal(t, "hook", hooks.Template)

}

func TestTemplate_RetryStrategy(t *testing.T) {
	tmpl := Template{}
	strategy, err := tmpl.GetRetryStrategy()
	assert.Nil(t, err)
	assert.Equal(t, wait.Backoff{Steps: 1}, strategy)
}

func TestGetExecSpec(t *testing.T) {
	wf := Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: WorkflowSpec{
			Templates: []Template{
				{Name: "spec-template"},
			},
		},
		Status: WorkflowStatus{
			StoredWorkflowSpec: &WorkflowSpec{
				Templates: []Template{
					{Name: "stored-spec-template"},
				},
			},
		},
	}

	assert.Equal(t, wf.GetExecSpec().Templates[0].Name, "stored-spec-template")

	wf = Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: WorkflowSpec{
			Templates: []Template{
				{Name: "spec-template"},
			},
		},
	}

	assert.Equal(t, wf.GetExecSpec().Templates[0].Name, "spec-template")

	wf.Status.StoredWorkflowSpec = nil

	assert.Equal(t, wf.GetExecSpec().Templates[0].Name, "spec-template")
}

// Check that inline tasks and steps are properly recovered from the store
func TestInlineStore(t *testing.T) {
	tests := map[ResourceScope]bool{
		ResourceScopeLocal:      false,
		ResourceScopeNamespaced: true,
		ResourceScopeCluster:    true,
	}

	for scope, shouldStore := range tests {
		t.Run(string(scope), func(t *testing.T) {
			wf := Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: WorkflowSpec{
					Templates: []Template{
						{
							Name: "dag-template",
							DAG: &DAGTemplate{
								Tasks: []DAGTask{
									{
										Name: "hello1",
										Inline: &Template{
											Script: &ScriptTemplate{
												Source: "abc",
											},
										},
									}, {
										Name: "hello2",
										Inline: &Template{
											Script: &ScriptTemplate{
												Source: "def",
											},
										},
									},
								},
							},
						},
						{
							Name: "step-template",
							Steps: []ParallelSteps{
								ParallelSteps{
									[]WorkflowStep{
										{
											Name: "hello1",
											Inline: &Template{
												Script: &ScriptTemplate{
													Source: "ghi",
												},
											},
										}, {
											Name: "hello2",
											Inline: &Template{
												Script: &ScriptTemplate{
													Source: "jkl",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			dagtmpl1 := &wf.Spec.Templates[0].DAG.Tasks[0]
			dagtmpl2 := &wf.Spec.Templates[0].DAG.Tasks[1]
			steptmpl1 := &wf.Spec.Templates[1].Steps[0].Steps[0]
			steptmpl2 := &wf.Spec.Templates[1].Steps[0].Steps[1]
			stored, err := wf.SetStoredTemplate(scope, "dag-template", dagtmpl1, dagtmpl1.Inline)
			assert.Equal(t, shouldStore, stored, "DAG template 1 should be stored for non local scopes")
			assert.Nil(t, err, "SetStoredTemplate for DAG1 should not return an error")
			stored, err = wf.SetStoredTemplate(scope, "dag-template", dagtmpl2, dagtmpl2.Inline)
			assert.Equal(t, shouldStore, stored, "DAG template 2 should be stored for non local scopes")
			assert.Nil(t, err, "SetStoredTemplate for DAG2 should not return an error")
			stored, err = wf.SetStoredTemplate(scope, "step-template", steptmpl1, steptmpl1.Inline)
			assert.Equal(t, shouldStore, stored, "Step template 1 should be stored for non local scopes")
			assert.Nil(t, err, "SetStoredTemplate for Step 1 should not return an error")
			stored, err = wf.SetStoredTemplate(scope, "step-template", steptmpl2, steptmpl2.Inline)
			assert.Equal(t, shouldStore, stored, "Step template 2 should be stored for non local scopes")
			assert.Nil(t, err, "SetStoredTemplate for Step 2 should not return an error")
			// For cases where we can store we should be able to retrieve and check
			if shouldStore {
				dagretrieved1 := wf.GetStoredTemplate(scope, "dag-template", dagtmpl1)
				assert.NotNil(t, dagretrieved1, "We should retrieve DAG Template 1")
				assert.Equal(t, dagtmpl1.Inline, dagretrieved1, "DAG template 1 should match what we stored")
				dagretrieved2 := wf.GetStoredTemplate(scope, "dag-template", dagtmpl2)
				assert.NotNil(t, dagretrieved2, "We should retrieve DAG Template 2")
				assert.Equal(t, dagtmpl2.Inline, dagretrieved2, "DAG template 2 should match what we stored")
				assert.NotEqual(t, dagretrieved1, dagretrieved2, "DAG template 1 and 2 should be different")

				stepretrieved1 := wf.GetStoredTemplate(scope, "step-template", steptmpl1)
				assert.NotNil(t, stepretrieved1, "We should retrieve Step Template 1")
				assert.Equal(t, steptmpl1.Inline, stepretrieved1, "Step template 1 should match what we stored")
				stepretrieved2 := wf.GetStoredTemplate(scope, "step-template", steptmpl2)
				assert.NotNil(t, stepretrieved2, "We should retrieve Step Template 2")
				assert.Equal(t, steptmpl2.Inline, stepretrieved2, "Step template 2 should match what we stored")
				assert.NotEqual(t, stepretrieved1, stepretrieved2, "Step template 1 and 2 should be different")
			}
		})
	}
}
