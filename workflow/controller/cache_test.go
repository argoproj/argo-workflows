package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/cache"
)

var sampleConfigMapCacheEntry = apiv1.ConfigMap{
	Data: map[string]string{
		"hi-there-world": `{"nodeID":"memoized-simple-workflow-5wj2p","outputs":{"parameters":[{"name":"hello","value":"foobar","valueFrom":{"path":"/tmp/hello_world.txt"}}],"artifacts":[{"name":"main-logs","archiveLogs":true,"s3":{"endpoint":"minio:9000","bucket":"my-bucket","insecure":true,"accessKeySecret":{"name":"my-minio-cred","key":"accesskey"},"secretKeySecret":{"name":"my-minio-cred","key":"secretkey"},"key":"memoized-simple-workflow-5wj2p/memoized-simple-workflow-5wj2p/main.log"}}]},"creationTimestamp":"2020-09-21T18:12:56Z"}`,
	},
	TypeMeta: metav1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:            "whalesay-cache",
		ResourceVersion: "1630732",
		Labels: map[string]string{
			common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache,
		},
	},
}

var sampleConfigMapEmptyCacheEntry = apiv1.ConfigMap{
	TypeMeta: metav1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:            "whalesay-cache",
		ResourceVersion: "1630732",
		Labels: map[string]string{
			common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache,
		},
	},
}

func TestConfigMapCacheLoadHit(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &sampleConfigMapCacheEntry, metav1.CreateOptions{})
	require.NoError(t, err)
	c := cache.NewConfigMapCache("default", controller.kubeclientset, "whalesay-cache")

	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, sampleConfigMapCacheEntry.Name, metav1.GetOptions{})
	require.NoError(t, err)

	entry, err := c.Load(ctx, "hi-there-world")
	require.NoError(t, err)
	assert.True(t, entry.LastHitTimestamp.After(entry.CreationTimestamp.Time))

	outputs := entry.Outputs
	require.NoError(t, err)
	require.Len(t, outputs.Parameters, 1)
	assert.Equal(t, "hello", outputs.Parameters[0].Name)
	assert.Equal(t, "foobar", outputs.Parameters[0].Value.String())
}

func TestConfigMapCacheLoadMiss(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &sampleConfigMapEmptyCacheEntry, metav1.CreateOptions{})
	require.NoError(t, err)
	c := cache.NewConfigMapCache("default", controller.kubeclientset, "whalesay-cache")
	entry, err := c.Load(ctx, "hi-there-world")
	require.NoError(t, err)
	assert.Nil(t, entry)
}

func TestConfigMapCacheSave(t *testing.T) {
	var MockParamValue = "Hello world"
	MockParam := wfv1.Parameter{
		Name:  "hello",
		Value: wfv1.AnyStringPtr(MockParamValue),
	}
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	c := cache.NewConfigMapCache("default", controller.kubeclientset, "whalesay-cache")

	outputs := wfv1.Outputs{}
	outputs.Parameters = append(outputs.Parameters, MockParam)
	err := c.Save(ctx, "hi-there-world", "", &outputs)
	require.NoError(t, err)

	cm, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, "whalesay-cache", metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, cm)
	var entry cache.Entry
	wfv1.MustUnmarshal([]byte(cm.Data["hi-there-world"]), &entry)
	assert.Equal(t, entry.LastHitTimestamp.Time, entry.CreationTimestamp.Time)
}
