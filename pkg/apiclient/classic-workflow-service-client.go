package apiclient

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/util/help"
	"github.com/argoproj/argo/workflow/packer"
	"github.com/argoproj/argo/workflow/util"
)

var (
	offloadError = fmt.Errorf("you cannot use the classic client because you have offload node states, see %s", help.CLI)
)

type classicWorkflowServiceClient struct {
	versioned.Interface
}

func (k *classicWorkflowServiceClient) CreateWorkflow(_ context.Context, in *workflowpkg.WorkflowCreateRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	wf := in.Workflow
	dryRun := len(in.CreateOptions.DryRun) > 0
	serverDryRun := in.ServerDryRun
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
		// kind of gross code, but fine
		return util.CreateServerDryRun(wf, k)
	}
	return k.ArgoprojV1alpha1().Workflows(in.Namespace).Create(wf)
}

func (k *classicWorkflowServiceClient) checkServerVersionForDryRun() (bool, error) {
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

func (k *classicWorkflowServiceClient) GetWorkflow(_ context.Context, in *workflowpkg.WorkflowGetRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	options := metav1.GetOptions{}
	if in.GetOptions != nil {
		options = *in.GetOptions
	}
	wf, err := k.ArgoprojV1alpha1().Workflows(in.Namespace).Get(in.Name, options)
	if err != nil {
		return nil, err
	}
	err = packer.DecompressWorkflow(wf)
	if err != nil {
		return nil, err
	}
	if wf.Status.IsOffloadNodeStatus() {
		return nil, offloadError
	}
	return wf, nil
}

func (k *classicWorkflowServiceClient) ListWorkflows(_ context.Context, in *workflowpkg.WorkflowListRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowList, error) {
	list, err := k.ArgoprojV1alpha1().Workflows(in.Namespace).List(*in.ListOptions)
	if err != nil {
		return nil, err
	}
	for _, wf := range list.Items {
		err = packer.DecompressWorkflow(&wf)
		if err != nil {
			return nil, err
		}
		if wf.Status.IsOffloadNodeStatus() {
			return nil, offloadError
		}
	}
	return list, nil
}

func (k *classicWorkflowServiceClient) WatchWorkflows(ctx context.Context, in *workflowpkg.WatchWorkflowsRequest, opts ...grpc.CallOption) (workflowpkg.WorkflowService_WatchWorkflowsClient, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) DeleteWorkflow(ctx context.Context, in *workflowpkg.WorkflowDeleteRequest, opts ...grpc.CallOption) (*workflowpkg.WorkflowDeleteResponse, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) RetryWorkflow(ctx context.Context, in *workflowpkg.WorkflowRetryRequest, opts ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) ResubmitWorkflow(ctx context.Context, in *workflowpkg.WorkflowResubmitRequest, opts ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) ResumeWorkflow(ctx context.Context, in *workflowpkg.WorkflowResumeRequest, opts ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) SuspendWorkflow(ctx context.Context, in *workflowpkg.WorkflowSuspendRequest, opts ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) TerminateWorkflow(ctx context.Context, in *workflowpkg.WorkflowTerminateRequest, opts ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) LintWorkflow(ctx context.Context, in *workflowpkg.WorkflowLintRequest, opts ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) PodLogs(ctx context.Context, in *workflowpkg.WorkflowLogRequest, opts ...grpc.CallOption) (workflowpkg.WorkflowService_PodLogsClient, error) {
	panic("implement me")
}
