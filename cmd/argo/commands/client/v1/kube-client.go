package v1

import (
	"fmt"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/packer"
	"github.com/argoproj/argo/workflow/util"
)

var misuseError = fmt.Errorf("this doesn't make sense unless you are using argo server, perhaps something like `export ARGO_SERVER=localhost:2746` would help")

// This client communicates with Argo using the K8S API.
// This is useful if you to speak to a system that does not have the Argo Server, but does not
// support some features, such as offloading large workflows or the workflow archive.
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

func newKubeClient() (Interface, error) {
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

func (k *kubeClient) GetWorkflow(namespace, name string) (*wfv1.Workflow, error) {
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

func (k *kubeClient) ListWorkflows(namespace string, opts metav1.ListOptions) (*wfv1.WorkflowList, error) {
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

func (k *kubeClient) Submit(namespace string, wf *wfv1.Workflow, dryRun, serverDryRun bool) (*wfv1.Workflow, error) {
	if dryRun {
		return wf, nil
	}
	if serverDryRun {
		ok, err := k.checkServerVersionForDryRun()
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("server-dry-run is not available for server api versions older than v1.12")
		}
		// kind of gross code, but find
		return util.CreateServerDryRun(wf, k.Interface)
	}
	return k.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
}

func (k *kubeClient) Token() (string, error) {
	return "", misuseError
}

func (k *kubeClient) checkServerVersionForDryRun() (bool, error) {
	serverVersion, err := k.Discovery().ServerVersion()
	if err != nil {
		return false, err
	}
	majorVersion, err := strconv.Atoi(serverVersion.Major)
	if err != nil {
		return false, err
	}
	minorVersion, err := strconv.Atoi(serverVersion.Minor)
	if err != nil {
		return false, err
	}
	if majorVersion < 1 {
		return false, nil
	} else if majorVersion == 1 && minorVersion < 12 {
		return false, nil
	}
	return true, nil
}
