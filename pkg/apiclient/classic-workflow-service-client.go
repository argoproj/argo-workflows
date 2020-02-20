package apiclient

import (
	"context"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/util/help"
	"github.com/argoproj/argo/workflow/packer"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/argo/workflow/validate"
)

var (
	offloadNodeStatusNotSupportedWarning = fmt.Sprintf("offload node status is not supported, see %s", help.CLI)
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

func (k *classicWorkflowServiceClient) GetWorkflow(_ context.Context, req *workflowpkg.WorkflowGetRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return k.getWorkflow(req.Namespace, req.Name, req.GetOptions)
}

func (k *classicWorkflowServiceClient) getWorkflow(namespace, name string, options *metav1.GetOptions) (*v1alpha1.Workflow, error) {
	if options == nil {
		options = &metav1.GetOptions{}
	}
	wf, err := k.Interface.ArgoprojV1alpha1().Workflows(namespace).Get(name, *options)
	if err != nil {
		return nil, err
	}
	err = packer.DecompressWorkflow(wf)
	if err != nil {
		return nil, err
	}
	if wf.Status.IsOffloadNodeStatus() {
		log.Warn(offloadNodeStatusNotSupportedWarning)
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
			log.Warn(offloadNodeStatusNotSupportedWarning)
		}
	}
	return list, nil
}

func (k *classicWorkflowServiceClient) WatchWorkflows(_ context.Context, _ *workflowpkg.WatchWorkflowsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchWorkflowsClient, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) DeleteWorkflow(_ context.Context, in *workflowpkg.WorkflowDeleteRequest, _ ...grpc.CallOption) (*workflowpkg.WorkflowDeleteResponse, error) {
	err := k.ArgoprojV1alpha1().Workflows(in.Namespace).Delete(in.Name, in.DeleteOptions)
	if err != nil {
		return nil, err
	}
	return &workflowpkg.WorkflowDeleteResponse{}, nil
}

func (k *classicWorkflowServiceClient) RetryWorkflow(_ context.Context, _ *workflowpkg.WorkflowRetryRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) ResubmitWorkflow(_ context.Context, req *workflowpkg.WorkflowResubmitRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	wf, err := k.getWorkflow(req.Namespace, req.Name, nil)
	if err != nil {
		return nil, err
	}
	newWF, err := util.FormulateResubmitWorkflow(wf, req.Memoized)
	if err != nil {
		return nil, err
	}
	created, err := util.SubmitWorkflow(k.Interface.ArgoprojV1alpha1().Workflows(req.Namespace), k.Interface, req.Namespace, newWF, &util.SubmitOpts{})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (k *classicWorkflowServiceClient) ResumeWorkflow(_ context.Context, _ *workflowpkg.WorkflowResumeRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) SuspendWorkflow(_ context.Context, _ *workflowpkg.WorkflowSuspendRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) TerminateWorkflow(_ context.Context, _ *workflowpkg.WorkflowTerminateRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	panic("implement me")
}

func (k *classicWorkflowServiceClient) LintWorkflow(_ context.Context, in *workflowpkg.WorkflowLintRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	templateGetter := templateresolution.WrapWorkflowTemplateInterface(k.Interface.ArgoprojV1alpha1().WorkflowTemplates(in.Namespace))
	err := validate.ValidateWorkflow(templateGetter, in.Workflow, validate.ValidateOpts{Lint: true})
	if err != nil {
		return nil, err
	}
	return in.Workflow, nil
}

func (k *classicWorkflowServiceClient) PodLogs(_ context.Context, _ *workflowpkg.WorkflowLogRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_PodLogsClient, error) {
	panic("implement me")
}
