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
		Status:     WorkflowStatus{FinishedAt: metav1.Time{Time: t1}}}))
	assert.False(t, WorkflowRanBetween(t1, t2)(Workflow{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t0}},
		Status:     WorkflowStatus{FinishedAt: metav1.Time{Time: t1}}}))
	assert.False(t, WorkflowRanBetween(t2, t3)(Workflow{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t0}},
		Status:     WorkflowStatus{FinishedAt: metav1.Time{Time: t1}}}))
	assert.False(t, WorkflowRanBetween(t0, t1)(Workflow{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t1}},
		Status:     WorkflowStatus{FinishedAt: metav1.Time{Time: t2}}}))
	assert.False(t, WorkflowRanBetween(t2, t3)(Workflow{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t1}},
		Status:     WorkflowStatus{FinishedAt: metav1.Time{Time: t2}}}))
	assert.True(t, WorkflowRanBetween(t0, t3)(Workflow{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: t1}},
		Status:     WorkflowStatus{FinishedAt: metav1.Time{Time: t2}}}))
}

func TestArtifactLocation_HasLocation(t *testing.T) {
	assert.False(t, (&ArtifactLocation{}).HasLocation())
	assert.False(t, (&ArtifactLocation{ArchiveLogs: pointer.BoolPtr(true)}).HasLocation())
	assert.True(t, (&ArtifactLocation{S3: &S3Artifact{Key: "my-key", S3Bucket: S3Bucket{Endpoint: "my-endpoint", Bucket: "my-bucket"}}}).HasLocation())
	assert.True(t, (&ArtifactLocation{Git: &GitArtifact{Repo: "my-repo"}}).HasLocation())
	assert.True(t, (&ArtifactLocation{HTTP: &HTTPArtifact{URL: "my-url"}}).HasLocation())
	assert.True(t, (&ArtifactLocation{Artifactory: &ArtifactoryArtifact{URL: "my-url"}}).HasLocation())
	assert.True(t, (&ArtifactLocation{Raw: &RawArtifact{Data: "my-data"}}).HasLocation())
	assert.True(t, (&ArtifactLocation{HDFS: &HDFSArtifact{HDFSConfig: HDFSConfig{Addresses: []string{"my-address"}}}}).HasLocation())
	assert.True(t, (&ArtifactLocation{OSS: &OSSArtifact{Key: "my-key", OSSBucket: OSSBucket{Endpoint: "my-endpoint", Bucket: "my-bucket"}}}).HasLocation())
	assert.True(t, (&ArtifactLocation{GCS: &GCSArtifact{Key: "my-key", GCSBucket: GCSBucket{Bucket: "my-bucket"}}}).HasLocation())
}

func TestArtifactLocation_GetType(t *testing.T) {
	assert.Equal(t, ArtifactLocationUnknown, (&ArtifactLocation{}).GetType())
	assert.Equal(t, ArtifactLocationS3, (&ArtifactLocation{S3: &S3Artifact{Key: "my-key", S3Bucket: S3Bucket{Endpoint: "my-endpoint", Bucket: "my-bucket"}}}).GetType())
	assert.Equal(t, ArtifactLocationGit, (&ArtifactLocation{Git: &GitArtifact{Repo: "my-repo"}}).GetType())
	assert.Equal(t, ArtifactLocationHTTP, (&ArtifactLocation{HTTP: &HTTPArtifact{URL: "my-url"}}).GetType())
	assert.Equal(t, ArtifactLocationArtifactory, (&ArtifactLocation{Artifactory: &ArtifactoryArtifact{URL: "my-url"}}).GetType())
	assert.Equal(t, ArtifactLocationRaw, (&ArtifactLocation{Raw: &RawArtifact{Data: "my-data"}}).GetType())
	assert.Equal(t, ArtifactLocationHDFS, (&ArtifactLocation{HDFS: &HDFSArtifact{HDFSConfig: HDFSConfig{Addresses: []string{"my-address"}}}}).GetType())
	assert.Equal(t, ArtifactLocationOSS, (&ArtifactLocation{OSS: &OSSArtifact{Key: "my-key", OSSBucket: OSSBucket{Endpoint: "my-endpoint", Bucket: "my-bucket"}}}).GetType())
	assert.Equal(t, ArtifactLocationGCS, (&ArtifactLocation{GCS: &GCSArtifact{Key: "my-key", GCSBucket: GCSBucket{Bucket: "my-bucket"}}}).GetType())
}

func TestArtifact_GetArchive(t *testing.T) {
	assert.NotNil(t, (&Artifact{}).GetArchive())
	assert.Equal(t, &ArchiveStrategy{None: &NoneStrategy{}}, (&Artifact{Archive: &ArchiveStrategy{None: &NoneStrategy{}}}).GetArchive())
}

func TestTemplate_IsResubmitAllowed(t *testing.T) {
	assert.False(t, (&Template{}).IsResubmitPendingPods())
	assert.True(t, (&Template{ResubmitPendingPods: true}).IsResubmitPendingPods())
}

func TestNodes_FindByDisplayName(t *testing.T) {
	assert.Nil(t, Nodes{}.FindByDisplayName(""))
	assert.NotNil(t, Nodes{"": NodeStatus{DisplayName: "foo"}}.FindByDisplayName("foo"))
}

func TestNodes_Any(t *testing.T) {
	assert.False(t, Nodes{"": NodeStatus{Name: "foo"}}.Any(func(node NodeStatus) bool { return node.Name == "bar" }))
	assert.True(t, Nodes{"": NodeStatus{Name: "foo"}}.Any(func(node NodeStatus) bool { return node.Name == "foo" }))
}

func TestResourcesDuration(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		assert.Equal(t, ResourcesDuration{}.String(), "")
		assert.Equal(t, ResourcesDuration{corev1.ResourceMemory: NewResourceDuration(1 * time.Second)}.String(), "1s*(100Mi memory)")
	})
	t.Run("Add", func(t *testing.T) {
		assert.Equal(t, ResourcesDuration{}.Add(ResourcesDuration{}).String(), "")
		assert.Equal(t, ResourcesDuration{corev1.ResourceMemory: NewResourceDuration(1 * time.Second)}.
			Add(ResourcesDuration{corev1.ResourceMemory: NewResourceDuration(1 * time.Second)}).
			String(), "2s*(100Mi memory)")
	})
	t.Run("CPUAndMemory", func(t *testing.T) {
		assert.Equal(t, ResourcesDuration{}.Add(ResourcesDuration{}).String(), "")
		s := ResourcesDuration{corev1.ResourceCPU: NewResourceDuration(2 * time.Second)}.
			Add(ResourcesDuration{corev1.ResourceMemory: NewResourceDuration(1 * time.Second)}).
			String()
		assert.Contains(t, s, "1s*(100Mi memory)")
		assert.Contains(t, s, "2s*(1 cpu)")
	})
}

func TestResourceDuration(t *testing.T) {
	assert.Equal(t, ResourceDuration(1), NewResourceDuration(1*time.Second))
	assert.Equal(t, "1s", NewResourceDuration(1*time.Second).String())
}

func TestNodes_GetResourcesDuration(t *testing.T) {
	assert.Equal(t, ResourcesDuration{}, Nodes{}.GetResourcesDuration())
	assert.Equal(t, ResourcesDuration{corev1.ResourceMemory: 3}, Nodes{
		"foo": NodeStatus{ResourcesDuration: ResourcesDuration{corev1.ResourceMemory: 1}},
		"bar": NodeStatus{ResourcesDuration: ResourcesDuration{corev1.ResourceMemory: 2}},
	}.GetResourcesDuration())
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

	assert.NotNil(t, spec.GetVolumeGC())
	assert.Equal(t, &VolumeGC{Strategy: VolumeGCOnSuccess}, spec.GetVolumeGC())
}
