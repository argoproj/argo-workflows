package v1

import (
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/workflowarchive"
)

// This client communicates with Argo using the Argo Server API.
// This supports all features, but requires you to install the Argo Server.
type argoAPIClient struct {
	*grpc.ClientConn
}

func newArgoAPIClient() Interface {
	return &argoAPIClient{client.GetClientConn()}
}

func (a *argoAPIClient) Namespace() (string, error) {
	namespace, _, err := client.Config.Namespace()
	return namespace, err
}

func (a *argoAPIClient) ListArchivedWorkflows(namespace string) (*wfv1.WorkflowList, error) {
	return workflowarchive.NewArchivedWorkflowServiceClient(a.ClientConn).ListArchivedWorkflows(client.GetContext(), &workflowarchive.ListArchivedWorkflowsRequest{
		ListOptions: &metav1.ListOptions{FieldSelector: "metadata.namespace=" + namespace},
	})
}

func (a *argoAPIClient) GetArchivedWorkflow(uid string) (*wfv1.Workflow, error) {
	return workflowarchive.NewArchivedWorkflowServiceClient(a.ClientConn).GetArchivedWorkflow(client.GetContext(), &workflowarchive.GetArchivedWorkflowRequest{
		Uid: uid,
	})
}

func (a *argoAPIClient) DeleteArchivedWorkflow(uid string) error {
	_, err := workflowarchive.NewArchivedWorkflowServiceClient(a.ClientConn).DeleteArchivedWorkflow(client.GetContext(), &workflowarchive.DeleteArchivedWorkflowRequest{
		Uid: uid,
	})
	return err
}
func (a *argoAPIClient) GetWorkflow(namespace, name string) (*wfv1.Workflow, error) {
	return workflowpkg.NewWorkflowServiceClient(a.ClientConn).GetWorkflow(client.GetContext(), &workflowpkg.WorkflowGetRequest{
		Name:      name,
		Namespace: namespace,
	})
}

func (a *argoAPIClient) ListWorkflows(namespace string, opts metav1.ListOptions) (*wfv1.WorkflowList, error) {
	return workflowpkg.NewWorkflowServiceClient(a.ClientConn).ListWorkflows(client.GetContext(), &workflowpkg.WorkflowListRequest{
		Namespace:   namespace,
		ListOptions: &opts,
	})
}

func (a *argoAPIClient) Submit(namespace string, wf *wfv1.Workflow, dryRun, serverDryRun bool) (*wfv1.Workflow, error) {
	if dryRun {
		return wf, nil
	}
	return workflowpkg.NewWorkflowServiceClient(a.ClientConn).CreateWorkflow(client.GetContext(), &workflowpkg.WorkflowCreateRequest{
		Namespace:    namespace,
		Workflow:     wf,
		ServerDryRun: serverDryRun,
	})
}

func (a *argoAPIClient) Token() (string, error) {
	return client.GetBearerToken(), nil
}
