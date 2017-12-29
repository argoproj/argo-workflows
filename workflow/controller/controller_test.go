package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apimachinery"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/fake"
	fakerest "k8s.io/client-go/rest/fake"
	"k8s.io/kubernetes/pkg/api"
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
	scheme := runtime.NewScheme()
	wfv1.AddToScheme(scheme)
	api.AddToScheme(scheme)
	registry, _ := registered.NewAPIRegistrationManager("v1")

	registry.RegisterGroup(apimachinery.GroupMeta{
		GroupVersion: wfv1.SchemeGroupVersion,
	})
	// The following prevents the "The legacy v1 API is not registered." panic from the fake RESTClient
	registry.RegisterGroup(apimachinery.GroupMeta{
		GroupVersion: schema.GroupVersion{Group: "", Version: ""},
	})
	return &WorkflowController{
		Config: WorkflowControllerConfig{
			ExecutorImage: "executor:latest",
		},
		clientset: fake.NewSimpleClientset(),
		scheme:    scheme,
		restClient: &fakerest.RESTClient{
			APIRegistry:          registry,
			NegotiatedSerializer: serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)},
			GroupName:            wfv1.CRDGroup,
			VersionedAPIPath:     wfv1.CRDVersion,
			Client: fakerest.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
				var wf wfv1.Workflow
				return &http.Response{StatusCode: 200, Header: defaultHeader(), Body: marshallBody(wf)}, nil
			}),
		},
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
	controller.clientset = nil
	wf := unmarshalWF(helloWorldWf)
	controller.operateWorkflow(wf)
}
