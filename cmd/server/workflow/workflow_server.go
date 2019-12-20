package workflow

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	commonserver "github.com/argoproj/argo/cmd/server/common"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/config"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/argo/workflow/validate"
)

type workflowServer struct {
	*commonserver.Server
	wfDBService   *DBService
	wfKubeService *kubeService
}

func NewWorkflowServer(namespace string, wfClientset versioned.Interface, kubeClientset kubernetes.Interface, config *config.WorkflowControllerConfig, enableClientAuth bool) *workflowServer {
	wfServer := workflowServer{Server: commonserver.NewServer(enableClientAuth, namespace, wfClientset, kubeClientset)}
	if config != nil && config.Persistence != nil {
		var err error
		wfServer.wfDBService, err = NewDBService(kubeClientset, namespace, config.Persistence)
		if err != nil {
			wfServer.wfDBService = nil
			log.Errorf("Error Creating DB Context. %v", err)
		} else {
			log.Infof("DB Context created successfully")
		}
	}

	return &wfServer
}

func (s *workflowServer) CreatePersistenceContext(namespace string, kubeClientSet *kubernetes.Clientset, config *config.PersistConfig) (*sqldb.WorkflowDBContext, error) {
	var wfDBCtx sqldb.WorkflowDBContext
	var err error
	wfDBCtx.NodeStatusOffload = config.NodeStatusOffload
	wfDBCtx.Session, wfDBCtx.TableName, err = sqldb.CreateDBSession(kubeClientSet, namespace, config)

	if err != nil {
		log.Errorf("Error in createPersistenceContext: %s", err)
		return nil, err
	}

	return &wfDBCtx, nil
}

func (s *workflowServer) CreateWorkflow(ctx context.Context, wfReq *WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	if wfReq.Workflow == nil {
		return nil, fmt.Errorf("workflow body not specified")
	}

	if wfReq.Workflow.Namespace == "" {
		wfReq.Workflow.Namespace = wfReq.Namespace
	}

	if wfReq.InstanceID != "" {
		labels := wfReq.Workflow.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[common.LabelKeyControllerInstanceID] = wfReq.InstanceID
		wfReq.Workflow.SetLabels(labels)
	}

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(wfReq.Namespace))

	err = validate.ValidateWorkflow(wftmplGetter, wfReq.Workflow, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}

	if wfReq.ServerDryRun {
		return util.CreateServerDryRun(wfReq.Workflow, wfClient)
	}

	wf, err := s.wfKubeService.Create(wfClient, wfReq.Namespace, wfReq.Workflow)

	if err != nil {
		log.Errorf("Create request is failed. Error: %s", err)
		return nil, err

	}
	return wf, nil
}

func (s *workflowServer) GetWorkflow(ctx context.Context, wfReq *WorkflowGetRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	wf, err := s.wfKubeService.Get(wfClient, wfReq.Namespace, wfReq.WorkflowName, wfReq.GetOptions)
	if err != nil {
		return nil, err
	}

	if wf.Status.OffloadNodeStatus {
		offloaded, err := s.wfDBService.Get(wfReq.WorkflowName, wfReq.Namespace)
		if err != nil {
			return nil, err
		}
		wf.Status.Nodes = offloaded.Status.Nodes
		wf.Status.CompressedNodes = offloaded.Status.CompressedNodes
	}

	return wf, err
}

func (s *workflowServer) ListWorkflows(ctx context.Context, wfReq *WorkflowListRequest) (*v1alpha1.WorkflowList, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	wfList, err := s.wfKubeService.List(wfClient, wfReq.Namespace, wfReq)
	if err != nil {
		return nil, err
	}

	if s.wfDBService != nil {
		offloadedWorkflows, err := s.wfDBService.List(wfReq.Namespace, 0, "")
		if err != nil {
			return nil, err
		}
		status := map[types.UID]v1alpha1.WorkflowStatus{}
		for _, item := range offloadedWorkflows.Items {
			status[item.UID] = item.Status
		}
		for _, wf := range wfList.Items {
			if wf.Status.OffloadNodeStatus {
				status, ok := status[wf.UID]
				if ok {
					wf.Status.Nodes = status.Nodes
					wf.Status.CompressedNodes = status.CompressedNodes
				} else {
					return nil, fmt.Errorf("unable to find offloaded workflow status for %s/%s", wfReq.Namespace, wf.UID)
				}
			}
		}
	}

	return wfList, nil
}

func (s *workflowServer) DeleteWorkflow(ctx context.Context, wfReq *WorkflowDeleteRequest) (*WorkflowDeleteResponse, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	wf, err := s.wfKubeService.Get(wfClient, wfReq.Namespace, wfReq.WorkflowName, nil)
	if err != nil {
		return nil, err
	}

	if wf.Status.OffloadNodeStatus {
		err = s.wfDBService.Delete(wfReq.WorkflowName, wfReq.Namespace)
		if err != nil {
			return nil, err
		}
	}

	return s.wfKubeService.Delete(wfClient, wfReq.Namespace, wfReq)
}

func (s *workflowServer) RetryWorkflow(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {

	wfClient, kubeClient, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	return s.wfKubeService.Retry(wfClient, kubeClient, wfReq.Namespace, wfReq)

}

func (s *workflowServer) ResubmitWorkflow(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}

	return s.wfKubeService.Resubmit(wfClient, wfReq.Namespace, wfReq)
}

func (s *workflowServer) ResumeWorkflow(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	return s.wfKubeService.Resume(wfClient, wfReq.Namespace, wfReq)

}

func (s *workflowServer) SuspendWorkflow(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	return s.wfKubeService.Suspend(wfClient, wfReq.Namespace, wfReq)
}

func (s *workflowServer) TerminateWorkflow(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	return s.wfKubeService.Terminate(wfClient, wfReq.Namespace, wfReq)

}

func (s *workflowServer) LintWorkflow(ctx context.Context, wfReq *WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(wfReq.Namespace))

	err = validate.ValidateWorkflow(wftmplGetter, wfReq.Workflow, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}

	return wfReq.Workflow, nil
}

func (s *workflowServer) WatchWorkflow(wfReq *WorkflowGetRequest, ws WorkflowService_WatchWorkflowServer) error {
	wfClient, _, err := s.GetWFClient(ws.Context())
	if err != nil {
		return err
	}
	return s.wfKubeService.WatchWorkflow(wfClient, wfReq, ws)
}

func (s *workflowServer) PodLogs(wfReq *WorkflowLogRequest, log WorkflowService_PodLogsServer) error {
	_, kubeClient, err := s.GetWFClient(log.Context())
	if err != nil {
		return err
	}

	return s.wfKubeService.PodLogs(kubeClient, wfReq, log)
}
