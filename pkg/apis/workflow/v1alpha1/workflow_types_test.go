package v1alpha1

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	assert.True(t, a.HasLocation())
	assert.NoError(t, a.SetKey("my-key"))
	key, err := a.GetKey()
	assert.NoError(t, err)
	assert.Equal(t, "http://my-host/my-key", a.URL)
	assert.Equal(t, "/my-key", key, "has leading slash")
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
	assert.Nil(t, l.Get())
	assert.Nil(t, (&ArtifactLocation{}).Get())
	assert.IsType(t, &GitArtifact{}, (&ArtifactLocation{Git: &GitArtifact{}}).Get())
	assert.IsType(t, &GCSArtifact{}, (&ArtifactLocation{GCS: &GCSArtifact{}}).Get())
	assert.IsType(t, &HDFSArtifact{}, (&ArtifactLocation{HDFS: &HDFSArtifact{}}).Get())
	assert.IsType(t, &HTTPArtifact{}, (&ArtifactLocation{HTTP: &HTTPArtifact{}}).Get())
	assert.IsType(t, &OSSArtifact{}, (&ArtifactLocation{OSS: &OSSArtifact{}}).Get())
	assert.IsType(t, &RawArtifact{}, (&ArtifactLocation{Raw: &RawArtifact{}}).Get())
	assert.IsType(t, &S3Artifact{}, (&ArtifactLocation{S3: &S3Artifact{}}).Get())
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

func TestTemplate_GetSidecarNames(t *testing.T) {
	m := &Template{
		Sidecars: []UserContainer{
			{Container: corev1.Container{Name: "sidecar-0"}},
		},
	}
	assert.ElementsMatch(t, []string{"sidecar-0"}, m.GetSidecarNames())
}
