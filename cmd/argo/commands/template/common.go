package template

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/templateresolution"
)

// Global variables
var (
	// DEPRECATED
	restConfig *rest.Config
	// DEPRECATED
	clientset *kubernetes.Clientset
	// DEPRECATED
	wfClientset *versioned.Clientset
	// DEPRECATED
	wftmplClient v1alpha1.WorkflowTemplateInterface
	// DEPRECATED
	namespace string
)

// DEPRECATED
func initKubeClient() *kubernetes.Clientset {
	if clientset != nil {
		return clientset
	}
	var err error
	restConfig, err = client.Config.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}

	// create the clientset
	clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatal(err)
	}
	return clientset
}

// InitWorkflowTemplateClient creates a new client for the Kubernetes WorkflowTemplate CRD.
// DEPRECATED
func InitWorkflowTemplateClient(ns ...string) v1alpha1.WorkflowTemplateInterface {
	if wftmplClient != nil {
		return wftmplClient
	}
	initKubeClient()
	var err error
	if len(ns) > 0 {
		namespace = ns[0]
	} else {
		namespace, _, err = client.Config.Namespace()
		if err != nil {
			log.Fatal(err)
		}
	}
	wfClientset = versioned.NewForConfigOrDie(restConfig)
	wftmplClient = wfClientset.ArgoprojV1alpha1().WorkflowTemplates(namespace)
	return wftmplClient
}

// LazyWorkflowTemplateGetter is a wrapper of v1alpha1.WorkflowTemplateInterface which
// supports lazy initialization.
// DEPRECATED
type LazyWorkflowTemplateGetter struct{}

// Get initializes it just before it's actually used and returns a retrieved workflow template.
// DEPRECATED
func (c LazyWorkflowTemplateGetter) Get(name string) (*wfv1.WorkflowTemplate, error) {
	if wftmplClient == nil {
		_ = InitWorkflowTemplateClient()
	}
	return templateresolution.WrapWorkflowTemplateInterface(wftmplClient).Get(name)
}

// DEPRECATED
var _ templateresolution.WorkflowTemplateNamespacedGetter = &LazyWorkflowTemplateGetter{}

// DEPRECATED
func GetWFtmplApiServerGRPCClient(conn *grpc.ClientConn) (workflowtemplate.WorkflowTemplateServiceClient, context.Context) {
	return workflowtemplate.NewWorkflowTemplateServiceClient(conn), client.GetContext()
}
