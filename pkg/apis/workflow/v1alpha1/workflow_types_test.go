package v1alpha1

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestWorkflows(t *testing.T) {
	wfs := Workflows{
		{ObjectMeta: v1.ObjectMeta{Name: "3"}, Status: WorkflowStatus{FinishedAt: v1.NewTime(time.Time{}.Add(1))}},
		{ObjectMeta: v1.ObjectMeta{Name: "2"}, Status: WorkflowStatus{FinishedAt: v1.NewTime(time.Time{}.Add(0))}},
		{ObjectMeta: v1.ObjectMeta{Name: "1"}, Status: WorkflowStatus{StartedAt: v1.NewTime(time.Time{}.Add(0))}},
		{ObjectMeta: v1.ObjectMeta{Name: "0"}, Status: WorkflowStatus{StartedAt: v1.NewTime(time.Time{}.Add(1))}},
	}
	sort.Sort(wfs)
	assert.Equal(t, "0", wfs[0].Name)
	assert.Equal(t, "1", wfs[1].Name)
	assert.Equal(t, "2", wfs[2].Name)
	assert.Equal(t, "3", wfs[3].Name)
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
	wfCond := WorkflowConditions{WorkflowCondition{Type: WorkflowConditionCompleted, Message: "Hello"}}
	wfCond.UpsertConditionMessage(WorkflowCondition{Type: WorkflowConditionCompleted, Message: "world!"})
	assert.Equal(t, "Hello, world!", wfCond[0].Message)
}

func TestShutdownStrategy_ShouldExecute(t *testing.T) {
	assert.False(t, ShutdownStrategyTerminate.ShouldExecute(true))
	assert.False(t, ShutdownStrategyTerminate.ShouldExecute(false))
	assert.False(t, ShutdownStrategyStop.ShouldExecute(false))
	assert.True(t, ShutdownStrategyStop.ShouldExecute(true))
}
