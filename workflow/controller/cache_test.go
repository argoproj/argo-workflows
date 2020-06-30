package controller

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var MockParamValue string = "Hello world"

var MockParam = wfv1.Parameter{
	Name: "hello",
	Value: &MockParamValue,
}

var workflowCached = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: memoized-workflow-test
spec:
  entrypoint: whalesay
  arguments:
    parameters:
    - name: message
      value: hi there world
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
    memoize:
      key: "{{inputs.parameters.message}}"
      maxAge: 1d
      cache:
        configMapName:
          name: whalesay-cache
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["sleep 10; cowsay {{inputs.parameters.message}} > /tmp/hello_world.txt"]
    outputs:
      parameters:
      - name: hello
        valueFrom:
          path: /tmp/hello_world.txt
`


var sampleConfigMapCacheEntry = v1.ConfigMap{
	Data: map[string]string{
		"hi-there-world": `{"ExpiresAt":"2020-06-18T17:11:05Z","NodeID":"memoize-abx4124-123129321123","Outputs":{"parameters":[{"name":"hello","value":"\n__________ \n\u003c hi there \u003e\n ---------- \n    \\\n     \\\n      \\     \n                    ##        .            \n              ##\n## ##       ==            \n           ## ## ## ##      ===            \n       /\"\"\"\"\"\"\"\"\"\"\"\"\"\"\"\"___/\n===        \n  ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~   \n       \\______ o          __/            \n        \\    \\        __/             \n          \\____\\______/   ","valueFrom":{"path":"/tmp/hello_world.txt"}}],"artifacts":[{"name":"main-logs","archiveLogs":true,"s3":{"endpoint":"minio:9000","bucket":"my-bucket","insecure":true,"accessKeySecret":{"name":"my-minio-cred","key":"accesskey"},"secretKeySecret":{"name":"my-minio-cred","key":"secretkey"},"key":"memoized-workflow-btfmf/memoized-workflow-btfmf/main.log"}}]}}`,
	},
	TypeMeta: metav1.TypeMeta{
		Kind: "ConfigMap",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "whalesay-cache",
		ResourceVersion: "1630732",
	},
}

var sampleOutput string = "\n__________ \n\u003c hi there \u003e\n ---------- \n    \\\n     \\\n      \\     \n                    ##        .            \n              ##\n## ##       ==            \n           ## ## ## ##      ===            \n       /\"\"\"\"\"\"\"\"\"\"\"\"\"\"\"\"___/\n===        \n  ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~   \n       \\______ o          __/            \n        \\    \\        __/             \n          \\____\\______/   "

func TestConfigMapCacheLoadOperate(t *testing.T) {
	wf := unmarshalWF(workflowCached)
	cancel, controller := newController()
	defer cancel()

	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(wf)
	assert.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(&sampleConfigMapCacheEntry)
	assert.NoError(t, err)

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()

	status := woc.wf.Status
	outputs := status.Nodes[""].Outputs

	assert.NotNil(t, outputs)
	assert.Equal(t, "hello", outputs.Parameters[0].Name)
	assert.Equal(t, sampleOutput, *outputs.Parameters[0].Value)
}
//
//func TestConfigMapCacheSaveOperate(t *testing.T) {
//	// create a workflow that's at the moment before the pod finished
//	// simulate the pod finishing so the controller reads the outputs and saves to cache
//	// assert that info was actually saved
//	wf := unmarshalWF(workflowCached)
//	woc := newWoc(*wf)
//	woc.operate()
//
//	outputs := wfv1.Outputs{}
//	outputs.Parameters = append(outputs.Parameters, MockParam)
//	ok := woc.controller.cache.Save("hello", &outputs)
//	assert.False(t, ok)
//}
//
//func TestConfigMapCacheLoad() {
//	wfclientset := fakewfclientset.NewSimpleClientset(objects...)
//	NewConfigMapcache(..., wfclientset)
//
//}
//
//func TestConfigMapCacheSave() {
//
//}

