package client

import (
	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type WorkflowClient struct {
	cl        *rest.RESTClient
	namespace string
}

// NewRESTClient returns a generic RESTClient that operates on kubernetes-like APIs
func NewRESTClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := wfv1.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}

	config := *cfg
	config.GroupVersion = &wfv1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{
		CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}
	return client, scheme, nil
}

func NewWorkflowClient(cl *rest.RESTClient, namespace string) *WorkflowClient {
	return &WorkflowClient{
		cl:        cl,
		namespace: namespace,
	}
}

func (f *WorkflowClient) CreateWorkflow(obj *wfv1.Workflow) (*wfv1.Workflow, error) {
	var result wfv1.Workflow
	err := f.cl.Post().
		Namespace(f.namespace).Resource(wfv1.CRDPlural).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *WorkflowClient) UpdateWorkflow(obj *wfv1.Workflow) (*wfv1.Workflow, error) {
	var result wfv1.Workflow
	err := f.cl.Put().
		Name(obj.ObjectMeta.Name).
		Namespace(f.namespace).Resource(wfv1.CRDPlural).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *WorkflowClient) DeleteWorkflow(name string, options *metav1.DeleteOptions) error {
	return f.cl.Delete().
		Name(name).
		Namespace(f.namespace).Resource(wfv1.CRDPlural).
		Body(options).Do().
		Error()
}

func (f *WorkflowClient) GetWorkflow(name string) (*wfv1.Workflow, error) {
	var result wfv1.Workflow
	err := f.cl.Get().
		Namespace(f.namespace).Resource(wfv1.CRDPlural).
		Name(name).Do().Into(&result)
	return &result, err
}

func (f *WorkflowClient) ListWorkflows(opts metav1.ListOptions) (*wfv1.WorkflowList, error) {
	var result wfv1.WorkflowList
	err := f.cl.Get().
		Namespace(f.namespace).Resource(wfv1.CRDPlural).
		//VersionedParams(&opts, f.codec).
		Do().Into(&result)
	return &result, err
}

func (f *WorkflowClient) NewListWatch() *cache.ListWatch {
	return cache.NewListWatchFromClient(
		f.cl,
		wfv1.CRDPlural,
		apiv1.NamespaceAll,
		fields.Everything())
}
