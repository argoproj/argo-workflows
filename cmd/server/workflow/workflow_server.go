package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/argoproj/argo/workflow/templateresolution"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	apisrvcmn "github.com/argoproj/argo/cmd/server/common"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/config"
	"github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/argo/workflow/validate"
)

type workflowServer struct {
	Namespace        string
	WfClientset      versioned.Interface
	KubeClientset    kubernetes.Interface
	EnableClientAuth bool
	Config           *config.WorkflowControllerConfig
	WfDBService      *DBService
	WfKubeService    *kubeService
}

func NewWorkflowServer(namespace string, wfClientset versioned.Interface, kubeClientSet kubernetes.Interface, config *config.WorkflowControllerConfig, enableClientAuth bool) *workflowServer {
	wfServer := workflowServer{Namespace: namespace, WfClientset: wfClientset, KubeClientset: kubeClientSet, EnableClientAuth: enableClientAuth}
	if config != nil && config.Persistence != nil {
		var err error
		wfServer.WfDBService, err = NewDBService(kubeClientSet, namespace, config.Persistence)
		if err != nil {
			wfServer.WfDBService = nil
			log.Errorf("Error Creating DB Context. %v", err)
		}else {
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

func (s *workflowServer) GetWFClient(ctx context.Context) (versioned.Interface, kubernetes.Interface, error) {
	md, _ := metadata.FromIncomingContext(ctx)

	if !s.EnableClientAuth {
		return s.WfClientset, s.KubeClientset, nil
	}

	var restConfigStr, bearerToken string
	if len(md.Get(apisrvcmn.CLIENT_REST_CONFIG)) == 0 {
		return nil, nil, errors.New("Client kubeconfig is not found")
	}
	restConfigStr = md.Get(apisrvcmn.CLIENT_REST_CONFIG)[0]

	if len(md.Get(apisrvcmn.AUTH_TOKEN)) > 0 {
		bearerToken = md.Get(apisrvcmn.AUTH_TOKEN)[0]
	}

	restConfig := rest.Config{}

	err := json.Unmarshal([]byte(restConfigStr), &restConfig)
	if err != nil {
		return nil, nil, err
	}

	restConfig.BearerToken = bearerToken

	wfClientset, err := wfclientset.NewForConfig(&restConfig)
	if err != nil {
		log.Errorf("Failure to create WfClientset with ClientConfig '%+v': %s", restConfig, err)
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(&restConfig)
	if err != nil {
		log.Errorf("Failure to create KubeClientset with ClientConfig '%+v': %s", restConfig, err)
		return nil, nil, err
	}

	return wfClientset, clientset, nil
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

	wf, err := s.ApplyWorkflowOptions(wfReq.Workflow, wfReq.SubmitOptions)
	if err != nil {
		return nil, err
	}

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(wfReq.Namespace))

	err = validate.ValidateWorkflow(wftmplGetter, wf, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}

	if wfReq.SubmitOptions != nil && wfReq.SubmitOptions.ServerDryRun {
		return util.CreateServerDryRun(wf, wfClient)
	}

	wf, err = s.WfKubeService.Create(wfClient, wfReq.Namespace, wfReq.Workflow)

	if err != nil {
		log.Errorf("Create request is failed. Error: %s", err)
		return nil, err
	}
	log.Infof("Workflow '%s' created successfully", wf.Name)
	return wf, nil
}

func (s *workflowServer) GetWorkflow(ctx context.Context, wfReq *WorkflowGetRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	var wf *v1alpha1.Workflow

	if s.WfDBService != nil {
		wf, err = s.WfDBService.Get(wfReq.WorkflowName, wfReq.Namespace)
	} else {

		wf, err = s.WfKubeService.Get(wfClient, wfReq.Namespace, wfReq.WorkflowName, wfReq.GetOptions)
		//wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, v1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	return wf, err
}

func (s *workflowServer) ListWorkflows(ctx context.Context, wfReq *WorkflowListRequest) (*v1alpha1.WorkflowList, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	var wfList *v1alpha1.WorkflowList

	if s.WfDBService != nil {
		var pagesize uint = 0
		if wfReq.ListOptions != nil {
			pagesize = uint(wfReq.ListOptions.Limit)
		}

		wfList, err = s.WfDBService.List(wfReq.Namespace, pagesize, "")
	} else {
		wfList, err = s.WfKubeService.List(wfClient, wfReq.Namespace, wfReq)
	}
	if err != nil {
		return nil, err
	}

	return wfList, nil
}

func (s *workflowServer) DeleteWorkflow(ctx context.Context, wfReq *WorkflowDeleteRequest) (*WorkflowDeleteResponse, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	if s.WfDBService != nil {
		err = s.WfDBService.Delete(wfReq.WorkflowName, wfReq.Namespace)
		if err != nil {
			return nil, err
		}
	}

	wfDelRes, err := s.WfKubeService.Delete(wfClient, wfReq.Namespace, wfReq)
	if err != nil {
		return nil, err
	}

	return wfDelRes, nil
}

func (s *workflowServer) RetryWorkflow(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {

	wfClient, kubeClient, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	return s.WfKubeService.Retry(wfClient, kubeClient, wfReq.Namespace, wfReq)

}

func (s *workflowServer) ResubmitWorkflow(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}

	return s.WfKubeService.Resubmit(wfClient, wfReq.Namespace, wfReq)
}

func (s *workflowServer) ResumeWorkflow(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	return s.WfKubeService.Resubmit(wfClient, wfReq.Namespace, wfReq)

}

func (s *workflowServer) SuspendWorkflow(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	return s.WfKubeService.Suspend(wfClient, wfReq.Namespace, wfReq)
}

func (s *workflowServer) TerminateWorkflow(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	return s.WfKubeService.Terminate(wfClient, wfReq.Namespace, wfReq)

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
	return s.WfKubeService.WatchWorkflow(wfClient, wfReq, ws)
}

func (s *workflowServer) PodLogs(wfReq *WorkflowLogRequest, log WorkflowService_PodLogsServer) error {
	_, kubeClient, err := s.GetWFClient(log.Context())
	if err != nil {
		return err
	}

	return s.WfKubeService.PodLogs(kubeClient, wfReq, log)
}

func (s *workflowServer) ApplyWorkflowOptions(wf *v1alpha1.Workflow, opts *SubmitOptions) (*v1alpha1.Workflow, error) {
	if opts == nil {
		return wf, nil
	}
	if opts.Entrypoint != "" {
		wf.Spec.Entrypoint = opts.Entrypoint
	}
	if opts.ServiceAccount != "" {
		wf.Spec.ServiceAccountName = opts.ServiceAccount
	}
	labels := wf.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	if opts.Labels != "" {
		passedLabels, err := cmdutil.ParseLabels(opts.Labels)
		if err != nil {
			return nil, fmt.Errorf("Expected labels of the form: NAME1=VALUE2,NAME2=VALUE2. Received: %s", opts.Labels)
		}
		for k, v := range passedLabels {
			labels[k] = v
		}
	}
	if opts.InstanceID != "" {
		labels[common.LabelKeyControllerInstanceID] = opts.InstanceID
	}
	wf.SetLabels(labels)
	if len(opts.Parameters) > 0 {
		newParams := make([]v1alpha1.Parameter, 0)
		passedParams := make(map[string]bool)
		for _, paramStr := range opts.Parameters {
			parts := strings.SplitN(paramStr, "=", 2)
			if len(parts) == 1 {
				return nil, fmt.Errorf("Expected parameter of the form: NAME=VALUE. Received: %s", paramStr)
			}
			param := v1alpha1.Parameter{
				Name:  parts[0],
				Value: &parts[1],
			}
			newParams = append(newParams, param)
			passedParams[param.Name] = true
		}

		for _, param := range wf.Spec.Arguments.Parameters {
			if _, ok := passedParams[param.Name]; ok {
				// this parameter was overridden via command line
				continue
			}
			newParams = append(newParams, param)
		}
		wf.Spec.Arguments.Parameters = newParams
	}
	if opts.GenerateName != "" {
		wf.ObjectMeta.GenerateName = opts.GenerateName
	}
	if opts.Name != "" {
		wf.ObjectMeta.Name = opts.Name
	}
	if opts.OwnerReference != nil {
		wf.SetOwnerReferences(append(wf.GetOwnerReferences(), *opts.OwnerReference))
	}
	return wf, nil
}
