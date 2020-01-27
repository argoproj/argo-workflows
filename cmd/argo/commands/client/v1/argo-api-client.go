package v1

import (
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/workflow"
	"github.com/argoproj/argo/server/workflowarchive"
)

type argoAPIClient struct {
	*grpc.ClientConn
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

func newArgoAPIClient() Interface {
	return &argoAPIClient{client.GetClientConn()}
}

func (a *argoAPIClient) Get(namespace, name string) (*wfv1.Workflow, error) {
	return workflow.NewWorkflowServiceClient(a.ClientConn).GetWorkflow(client.GetContext(), &workflow.WorkflowGetRequest{
		Name:      name,
		Namespace: namespace,
	})
}

func (a *argoAPIClient) List(namespace string, opts metav1.ListOptions) (*wfv1.WorkflowList, error) {
	return workflow.NewWorkflowServiceClient(a.ClientConn).ListWorkflows(client.GetContext(), &workflow.WorkflowListRequest{
		Namespace:   namespace,
		ListOptions: &opts,
	})
}

func (a *argoAPIClient) Submit(namespace string, wf *wfv1.Workflow, dryRun bool) (*wfv1.Workflow, error) {
	return workflow.NewWorkflowServiceClient(a.ClientConn).CreateWorkflow(client.GetContext(), &workflow.WorkflowCreateRequest{
		Namespace:    namespace,
		Workflow:     wf,
		ServerDryRun: dryRun,
	})
}

func (a *argoAPIClient) GetToken() (string, error) {
	return client.GetToken(), nil
}
