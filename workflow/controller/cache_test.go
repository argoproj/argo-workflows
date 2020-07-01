package controller

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"

	apiv1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var sampleOutput string = "\n__________ \n\u003c hi there \u003e\n ---------- \n    \\\n     \\\n      \\     \n                    ##        .            \n              ##\n## ##       ==            \n           ## ## ## ##      ===            \n       /\"\"\"\"\"\"\"\"\"\"\"\"\"\"\"\"___/\n===        \n  ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~   \n       \\______ o          __/            \n        \\    \\        __/             \n          \\____\\______/   "

var sampleConfigMapCacheEntry = apiv1.ConfigMap{
	Data: map[string]string{
		"hi-there-world": `{"ExpiresAt":"2020-06-18T17:11:05Z","NodeID":"memoize-abx4124-123129321123","Outputs":{"parameters":[{"name":"hello","value":"\n__________ \n\u003c hi there \u003e\n ---------- \n    \\\n     \\\n      \\     \n                    ##        .            \n              ##\n## ##       ==            \n           ## ## ## ##      ===            \n       /\"\"\"\"\"\"\"\"\"\"\"\"\"\"\"\"___/\n===        \n  ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~   \n       \\______ o          __/            \n        \\    \\        __/             \n          \\____\\______/   ","valueFrom":{"path":"/tmp/hello_world.txt"}}],"artifacts":[{"name":"main-logs","archiveLogs":true,"s3":{"endpoint":"minio:9000","bucket":"my-bucket","insecure":true,"accessKeySecret":{"name":"my-minio-cred","key":"accesskey"},"secretKeySecret":{"name":"my-minio-cred","key":"secretkey"},"key":"memoized-workflow-btfmf/memoized-workflow-btfmf/main.log"}}]}}`,
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

func TestConfigMapCacheLoad(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(&sampleConfigMapCacheEntry)
	assert.NoError(t, err)
	c := NewConfigMapCache("whalesay-cache", "default", controller.kubeclientset)
	entry, err := c.Load("hi-there-world")
	assert.NoError(t, err)
	assert.Equal(t, "hello", entry.Parameters[0].Name)
	assert.Equal(t, sampleOutput, *entry.Parameters[0].Value)
}

func TestConfigMapCacheSave(t *testing.T) {
	var MockParamValue string = "Hello world"

	var MockParam = wfv1.Parameter{
		Name:  "hello",
		Value: &intstr.IntOrString{StrVal: MockParamValue},
	}
	cancel, controller := newController()
	defer cancel()
	c := NewConfigMapCache("whalesay-cache", "default", controller.kubeclientset)
	outputs := wfv1.Outputs{}
	outputs.Parameters = append(outputs.Parameters, MockParam)
	err := c.Save("hi-there-world", "", &outputs)
	assert.NoError(t, err)
	cm, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get("whalesay-cache", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, cm)
}
