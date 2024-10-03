package sync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
)

func Test_NamespaceBucket(t *testing.T) {
	assert.Equal(t, "a", NamespaceBucket("a/b"))
}

func TestNoParallelismSamePriority(t *testing.T) {
	throttler := NewThrottler(0, SingleBucket, nil)

	throttler.Add("c", 0, time.Now().Add(2*time.Hour))
	throttler.Add("b", 0, time.Now().Add(1*time.Hour))
	throttler.Add("a", 0, time.Now())

	assert.True(t, throttler.Admit("a"))
	assert.True(t, throttler.Admit("b"))
	assert.True(t, throttler.Admit("c"))
}

func TestNoParallelismMultipleBuckets(t *testing.T) {
	throttler := NewThrottler(1, func(key Key) BucketKey {
		namespace, _, _ := cache.SplitMetaNamespaceKey(key)
		return namespace
	}, func(key string) {})

	throttler.Add("a/0", 0, time.Now())
	throttler.Add("a/1", 0, time.Now())
	throttler.Add("b/0", 0, time.Now())
	throttler.Add("b/1", 0, time.Now())

	assert.True(t, throttler.Admit("a/0"))
	assert.False(t, throttler.Admit("a/1"))
	assert.True(t, throttler.Admit("b/0"))
	throttler.Remove("a/0")
	assert.True(t, throttler.Admit("a/1"))
}

func TestWithParallelismLimitAndPriority(t *testing.T) {
	queuedKey := ""
	throttler := NewThrottler(2, SingleBucket, func(key string) { queuedKey = key })

	throttler.Add("a", 1, time.Now())
	throttler.Add("b", 2, time.Now())
	throttler.Add("c", 3, time.Now())
	throttler.Add("d", 4, time.Now())

	assert.True(t, throttler.Admit("a"), "is started, even though low priority")
	assert.True(t, throttler.Admit("b"), "is started, even though low priority")
	assert.False(t, throttler.Admit("c"), "cannot start")
	assert.False(t, throttler.Admit("d"), "cannot start")
	assert.Equal(t, "b", queuedKey)
	queuedKey = ""

	throttler.Remove("a")
	assert.True(t, throttler.Admit("b"), "stays running")
	assert.True(t, throttler.Admit("d"), "top priority")
	assert.False(t, throttler.Admit("c"))
	assert.Equal(t, "d", queuedKey)
	queuedKey = ""

	throttler.Remove("b")
	assert.True(t, throttler.Admit("d"), "top priority")
	assert.True(t, throttler.Admit("c"), "now running too")
	assert.Equal(t, "c", queuedKey)
}

func TestInitWithWorkflows(t *testing.T) {
	queuedKey := ""
	throttler := NewThrottler(1, SingleBucket, func(key string) { queuedKey = key })
	ctx := context.Background()

	wfclientset := fakewfclientset.NewSimpleClientset(
		wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  labels:
    workflows.argoproj.io/phase: Running
  name: a
  namespace: default
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
status:
  phase: Running
  startedAt: "2020-06-19T17:37:05Z"
`),
		wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  labels:
    workflows.argoproj.io/phase: Running
  name: b
  namespace: default
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
status:
  phase: Running
  startedAt: "2020-06-19T17:37:05Z"
`))
	wfList, err := wfclientset.ArgoprojV1alpha1().Workflows("default").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	err = throttler.Init(wfList.Items)
	require.NoError(t, err)
	assert.True(t, throttler.Admit("default/a"))
	assert.True(t, throttler.Admit("default/b"))

	throttler.Add("default/c", 0, time.Now())
	throttler.Add("default/d", 0, time.Now())
	assert.False(t, throttler.Admit("default/c"))
	assert.False(t, throttler.Admit("default/d"))

	throttler.Remove("default/a")
	assert.Equal(t, "", queuedKey)
	assert.False(t, throttler.Admit("default/c"))
	assert.False(t, throttler.Admit("default/d"))

	queuedKey = ""
	throttler.Remove("default/b")
	assert.Equal(t, "default/c", queuedKey)
	assert.True(t, throttler.Admit("default/c"))
	assert.False(t, throttler.Admit("default/d"))

	queuedKey = ""
	throttler.Remove("default/c")
	assert.Equal(t, "default/d", queuedKey)
	assert.True(t, throttler.Admit("default/d"))
}
