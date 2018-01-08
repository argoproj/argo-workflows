package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

var helloWorldWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

func newController() *WorkflowController {
	return &WorkflowController{
		Config: WorkflowControllerConfig{
			ExecutorImage: "executor:latest",
		},
		kubeclientset: fake.NewSimpleClientset(),
		wfclientset:   fakewfclientset.NewSimpleClientset(),
	}
}
func defaultHeader() http.Header {
	header := http.Header{}
	header.Set("Content-Type", runtime.ContentTypeJSON)
	return header
}

func marshallBody(b interface{}) io.ReadCloser {
	result, err := json.Marshal(b)
	if err != nil {
		panic(err)
	}
	return ioutil.NopCloser(bytes.NewReader(result))
}

func unmarshalWF(yamlStr string) *wfv1.Workflow {
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(yamlStr), &wf)
	if err != nil {
		panic(err)
	}
	return &wf
}

// TestOperateWorkflowPanicRecover ensures we can recover from unexpected panics
func TestOperateWorkflowPanicRecover(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fail()
		}
	}()
	controller := newController()
	// intentionally set clientset to nil to induce panic
	controller.kubeclientset = nil
	wf := unmarshalWF(helloWorldWf)
	controller.operateWorkflow(wf)
}
