package v1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type Interface interface {
	Submit(namespace string, wf *wfv1.Workflow, dryRun bool) (*wfv1.Workflow, error)
	List(namespace string, opts v1.ListOptions) (*wfv1.WorkflowList, error)
	Get(namespace, name string) (*wfv1.Workflow, error)
	GetToken() (string, error)
	DeleteArchivedWorkflow(uid string) error
	GetArchivedWorkflow(uid string) (*wfv1.Workflow, error)
	ListArchivedWorkflows(namespace string) (*wfv1.WorkflowList, error)
	Namespace() (string, error)
}

func GetClient() (Interface, error) {
	if client.ArgoServer != "" {
		return newArgoAPIClient(), nil
	} else {
		return newKubeImpl()
	}
}
