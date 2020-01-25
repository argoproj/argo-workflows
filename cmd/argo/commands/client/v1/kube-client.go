package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/packer"
)

var misuseError = fmt.Errorf("this doesn't make sense unless you are using argo server, perhaps something like `export ARGO_SERVER=localhost:2746` would help")

type kubeClient struct {
	versioned.Interface
}

func (k *kubeClient) Namespace() (string, error) {
	namespace, _, err := client.Config.Namespace()
	return namespace, err
}

func (k *kubeClient) ListArchivedWorkflows(_ string) (*wfv1.WorkflowList, error) {
	return nil, misuseError
}

func (k *kubeClient) GetArchivedWorkflow(_ string) (*wfv1.Workflow, error) {
	return nil, misuseError
}

func (k *kubeClient) DeleteArchivedWorkflow(_ string) error {
	return misuseError
}

func newKubeImpl() (Interface, error) {
	restConfig, err := client.Config.ClientConfig()
	if err != nil {
		return nil, err
	}
	wfClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return &kubeClient{wfClient}, nil
}

func (k *kubeClient) Get(namespace, name string) (*wfv1.Workflow, error) {
	wf, err := k.ArgoprojV1alpha1().Workflows(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = packer.DecompressWorkflow(wf)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (k *kubeClient) List(namespace string, opts metav1.ListOptions) (*wfv1.WorkflowList, error) {
	list, err := k.ArgoprojV1alpha1().Workflows(namespace).List(opts)
	if err != nil {
		return nil, err
	}
	for _, wf := range list.Items {
		err = packer.DecompressWorkflow(&wf)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (k *kubeClient) Submit(namespace string, wf *wfv1.Workflow, dryRun bool) (*wfv1.Workflow, error) {
	if dryRun {
		return nil, fmt.Errorf("dryRun not implemented")
	}
	return k.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
}

func (k *kubeClient) GetToken() (string, error) {
	return "", misuseError
}
