package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
)

func TestMultiNoParallelismSamePriority(t *testing.T) {
	throttler := NewMultiThrottler(0, 0, func(Key) {})

	throttler.Add("default/c", 0, time.Now().Add(2*time.Hour))
	throttler.Add("default/b", 0, time.Now().Add(1*time.Hour))
	throttler.Add("default/a", 0, time.Now())

	assert.True(t, throttler.Admit("default/a"))
	assert.True(t, throttler.Admit("default/b"))
	assert.True(t, throttler.Admit("default/c"))
}

func TestMultiNoParallelismMultipleBuckets(t *testing.T) {
	throttler := NewMultiThrottler(1, 1, func(Key) {})
	throttler.Add("a/0", 0, time.Now())
	throttler.Add("a/1", 0, time.Now().Add(-1*time.Second))
	throttler.Add("b/0", 0, time.Now().Add(-2*time.Second))
	throttler.Add("b/1", 0, time.Now().Add(-3*time.Second))

	assert.True(t, throttler.Admit("a/0"))
	assert.False(t, throttler.Admit("a/1"))
	assert.False(t, throttler.Admit("b/0"))
	assert.False(t, throttler.Admit("b/1"))
	throttler.Remove("a/0")
	assert.True(t, throttler.Admit("b/1"))
}

func TestMultiWithParallelismLimitAndPriority(t *testing.T) {
	queuedKey := ""
	throttler := NewMultiThrottler(2, 0, func(key string) { queuedKey = key })

	throttler.Add("default/a", 1, time.Now())
	throttler.Add("default/b", 2, time.Now())
	throttler.Add("default/c", 3, time.Now())
	throttler.Add("default/d", 4, time.Now())

	assert.True(t, throttler.Admit("default/a"), "is started, even though low priority")
	assert.True(t, throttler.Admit("default/b"), "is started, even though low priority")
	assert.False(t, throttler.Admit("default/c"), "cannot start")
	assert.False(t, throttler.Admit("default/d"), "cannot start")
	assert.Equal(t, "default/b", queuedKey)
	queuedKey = ""

	throttler.Remove("default/a")
	assert.True(t, throttler.Admit("default/b"), "stays running")
	assert.True(t, throttler.Admit("default/d"), "top priority")
	assert.False(t, throttler.Admit("default/c"))
	assert.Equal(t, "default/d", queuedKey)
	queuedKey = ""

	throttler.Remove("default/b")
	assert.True(t, throttler.Admit("default/d"), "top priority")
	assert.True(t, throttler.Admit("default/c"), "now running too")
	assert.Equal(t, "default/c", queuedKey)
}

func TestMultiInitWithWorkflows(t *testing.T) {
	queuedKey := ""
	throttler := NewMultiThrottler(1, 1, func(key string) { queuedKey = key })
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
	assert.Empty(t, queuedKey)
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

func TestTotalAllowNamespaceLimit(t *testing.T) {
	namespaceLimits := make(map[string]int)
	namespaceLimits["a"] = 2
	namespaceLimits["b"] = 1
	throttler := &multiThrottler{
		queue:                       func(key Key) {},
		namespaceParallelism:        namespaceLimits,
		namespaceParallelismDefault: 6,
		totalParallelism:            4,
		running:                     make(map[Key]bool),
		pending:                     make(map[string]*priorityQueue),
		lock:                        &sync.Mutex{},
	}
	throttler.Add("a/0", 1, time.Now())
	throttler.Add("b/0", 2, time.Now())
	throttler.Add("a/1", 3, time.Now())
	throttler.Add("a/2", 4, time.Now())
	throttler.Add("a/3", 5, time.Now())
	throttler.Add("a/4", 6, time.Now())
	throttler.Add("b/1", 7, time.Now())

	assert.True(t, throttler.Admit("a/0"))
	assert.True(t, throttler.Admit("b/0"))
	assert.True(t, throttler.Admit("a/1"))

	assert.False(t, throttler.Admit("a/2"))
	assert.False(t, throttler.Admit("a/3"))
	assert.False(t, throttler.Admit("a/4"))
	assert.False(t, throttler.Admit("b/1"))

	throttler.Add("c/0", 8, time.Now())
	assert.True(t, throttler.Admit("c/0"))
}

func TestPriorityAcrossNamespaces(t *testing.T) {
	throttler := NewMultiThrottler(3, 1, func(Key) {})
	throttler.Add("a/0", 0, time.Now())
	throttler.Add("a/1", 0, time.Now())
	throttler.Add("a/2", 0, time.Now())
	throttler.Add("b/0", 1, time.Now())
	throttler.Add("b/1", 1, time.Now())
	throttler.Add("b/2", 1, time.Now())

	assert.True(t, throttler.Admit("a/0"))
	assert.True(t, throttler.Admit("b/0"))
	assert.False(t, throttler.Admit("a/1"))
	assert.False(t, throttler.Admit("a/2"))
	assert.True(t, throttler.Admit("b/0"))
	assert.False(t, throttler.Admit("b/1"))
	assert.False(t, throttler.Admit("b/2"))
	throttler.Remove("a/0")
	assert.False(t, throttler.Admit("b/1"))
	assert.True(t, throttler.Admit("a/1"))
	throttler.Remove("b/0")
	assert.True(t, throttler.Admit("b/1"))
	assert.False(t, throttler.Admit("a/2"))
}
