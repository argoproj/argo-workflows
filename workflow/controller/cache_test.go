package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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
	},
}

func TestConfigMapCacheLoadHit(t *testing.T) {
	cancel, controller := newController()
	defer cancel()

	ctx := context.Background()
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &sampleConfigMapCacheEntry, metav1.CreateOptions{})
	assert.NoError(t, err)
	c := cache.NewConfigMapCache("default", controller.kubeclientset, "whalesay-cache")

	cm, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, sampleConfigMapCacheEntry.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Nil(t, cm.Labels)

	entry, err := c.Load(ctx, "hi-there-world")
	assert.NoError(t, err)
	assert.True(t, entry.LastHitTimestamp.Time.After(entry.CreationTimestamp.Time))

	outputs := entry.Outputs
	assert.NoError(t, err)
	if assert.Len(t, outputs.Parameters, 1) {
		assert.Equal(t, "hello", outputs.Parameters[0].Name)
		assert.Equal(t, "foobar", outputs.Parameters[0].Value.String())
	}
}

func TestConfigMapCacheLoadMiss(t *testing.T) {
	cancel, controller := newController()
	defer cancel()

	ctx := context.Background()
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &sampleConfigMapEmptyCacheEntry, metav1.CreateOptions{})
	assert.NoError(t, err)
	c := cache.NewConfigMapCache("default", controller.kubeclientset, "whalesay-cache")
	entry, err := c.Load(ctx, "hi-there-world")
	assert.NoError(t, err)
	assert.Nil(t, entry)
}

func TestConfigMapCacheSave(t *testing.T) {
	var MockParamValue string = "Hello world"
	MockParam := wfv1.Parameter{
		Name:  "hello",
		Value: wfv1.AnyStringPtr(MockParamValue),
	}
	cancel, controller := newController()
	defer cancel()
	c := cache.NewConfigMapCache("default", controller.kubeclientset, "whalesay-cache")

	ctx := context.Background()
	outputs := wfv1.Outputs{}
	outputs.Parameters = append(outputs.Parameters, MockParam)
	err := c.Save(ctx, "hi-there-world", "", &outputs)
	assert.NoError(t, err)

	cm, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, "whalesay-cache", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, cm)
	var entry cache.Entry
	wfv1.MustUnmarshal([]byte(cm.Data["hi-there-world"]), &entry)
	assert.Equal(t, entry.LastHitTimestamp.Time, entry.CreationTimestamp.Time)
}
