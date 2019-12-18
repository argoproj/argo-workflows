package workflow

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/argoproj/argo/workflow/templateresolution"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	apisrvcmn "github.com/argoproj/argo/cmd/server/common"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/config"
	"github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/argo/workflow/validate"
)

type WorkflowServer struct {
	Namespace        string
	WfClientset      versioned.Interface
	KubeClientset    kubernetes.Interface
	EnableClientAuth bool
	Config           *config.WorkflowControllerConfig
	WfDBService      *DBService
	WfKubeService    *KubeService
}

func NewWorkflowServer(namespace string, wfClientset versioned.Interface, kubeClientSet kubernetes.Interface, config *config.WorkflowControllerConfig, enableClientAuth bool) *WorkflowServer {
	wfServer := WorkflowServer{Namespace: namespace, WfClientset: wfClientset, KubeClientset: kubeClientSet, EnableClientAuth: enableClientAuth}
	if config != nil && config.Persistence != nil {
		var err error
		wfServer.WfDBService, err = NewDBService(kubeClientSet, namespace, config.Persistence)
		if err != nil {
			wfServer.WfDBService = nil
			log.Errorf("Error Creating DB Context. %v", err)
		} else {
			log.Infof("DB Context created successfully")
		}
	}

	return &wfServer
}

func (s *WorkflowServer) CreatePersistenceContext(namespace string, kubeClientSet *kubernetes.Clientset, config *config.PersistConfig) (*sqldb.WorkflowDBContext, error) {
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

func (s *WorkflowServer) GetWFClient(ctx context.Context) (versioned.Interface, kubernetes.Interface, error) {
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

func (s *WorkflowServer) Create(ctx context.Context, wfReq *WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	if wfReq.Workflow == nil {
		return nil, fmt.Errorf("workflow body not specified")
	}
	namespace := s.Namespace
	if wfReq.Namespace != "" {
		namespace = wfReq.Namespace
	}

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(namespace))

	err = validate.ValidateWorkflow(wftmplGetter, wfReq.Workflow, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}

	dryRun := wfReq.CreateOptions != nil && len(wfReq.CreateOptions.DryRun) > 0
	log.WithField("dryRun", dryRun).Info("Creating workflow")
	if !dryRun {
		wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Create(wfReq.Workflow)
		if err != nil {
			log.Errorf("Create request is failed. Error: %s", err)
			return nil, err
		}
		log.Infof("Workflow '%s' created successfully", wf.Name)
		return wf, nil
	} else {
		return wfReq.Workflow, nil
	}
}

func (s *WorkflowServer) Get(ctx context.Context, wfReq *WorkflowGetRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	var wf *v1alpha1.Workflow
	namespace := s.Namespace
	if wfReq.Namespace != "" {
		namespace = wfReq.Namespace
	}

	if s.WfDBService != nil {
		wf, err = s.WfDBService.Get(wfReq.WorkflowName, wfReq.Namespace)
	} else {

		wf, err = wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, v1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	return wf, err
}

func (s *WorkflowServer) List(ctx context.Context, wfReq *WorkflowListRequest) (*v1alpha1.WorkflowList, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	var wfList *v1alpha1.WorkflowList
	var listOption v1.ListOptions = v1.ListOptions{}
	namespace := s.Namespace
	if wfReq.Namespace != "" {
		namespace = wfReq.Namespace
	}

	if wfReq.ListOptions != nil {
		listOption = *wfReq.ListOptions
	}

	if s.WfDBService != nil {
		var pagesize uint = 0
		if wfReq.ListOptions != nil {
			pagesize = uint(wfReq.ListOptions.Limit)
		}

		wfList, err = s.WfDBService.List(namespace, pagesize, "")
	} else {
		wfList, err = wfClient.ArgoprojV1alpha1().Workflows(namespace).List(listOption)
	}
	if err != nil {
		return nil, err
	}

	return wfList, nil
}

func (s *WorkflowServer) Delete(ctx context.Context, wfReq *WorkflowDeleteRequest) (*WorkflowDeleteResponse, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	namespace := s.Namespace
	if wfReq.Namespace != "" {
		namespace = wfReq.Namespace
	}

	if s.WfDBService != nil {
		err = s.WfDBService.Delete(wfReq.WorkflowName, wfReq.Namespace)
		if err != nil {
			return nil, err
		}
	}

	err = wfClient.ArgoprojV1alpha1().Workflows(namespace).Delete(wfReq.WorkflowName, &v1.DeleteOptions{})
	if err != nil {
		return nil, err
	}

	return &WorkflowDeleteResponse{
		WorkflowName: wfReq.WorkflowName,
		Status:       "Deleted",
	}, nil
}

func (s *WorkflowServer) Retry(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	namespace := s.Namespace
	if wfReq.Namespace != "" {
		namespace = wfReq.Namespace
	}

	wfClient, kubeClient, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, v1.GetOptions{})

	if err != nil {
		return nil, err
	}

	wf, err = util.RetryWorkflow(kubeClient, wfClient.ArgoprojV1alpha1().Workflows(namespace), wf)

	if err != nil {
		return nil, err
	}
	return wf, err
}

func (s *WorkflowServer) Resubmit(ctx context.Context, in *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	newWF, err := util.FormulateResubmitWorkflow(wf, in.Memoized)
	if err != nil {
		return nil, err
	}
	created, err := util.SubmitWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), wfClient, namespace, newWF, nil)
	if err != nil {
		return nil, err
	}

	return created, err
}

func (s *WorkflowServer) Resume(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	namespace := s.Namespace
	if wfReq.Namespace != "" {
		namespace = wfReq.Namespace
	}

	err = util.ResumeWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), wfReq.WorkflowName)
	if err != nil {
		log.Warnf("Failed to resume '%s': %s", wfReq.WorkflowName, err)
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *WorkflowServer) Suspend(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	namespace := s.Namespace
	if wfReq.Namespace != "" {
		namespace = wfReq.Namespace
	}

	err = util.SuspendWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), wfReq.WorkflowName)
	if err != nil {
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *WorkflowServer) Terminate(ctx context.Context, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	namespace := s.Namespace
	if wfReq.Namespace != "" {
		namespace = wfReq.Namespace
	}

	err = util.TerminateWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), wfReq.WorkflowName)
	if err != nil {
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *WorkflowServer) Lint(ctx context.Context, wfReq *WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	namespace := s.Namespace
	if wfReq.Namespace != "" {
		namespace = wfReq.Namespace
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(namespace))

	err = validate.ValidateWorkflow(wftmplGetter, wfReq.Workflow, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}

	return wfReq.Workflow, nil
}

func (s *WorkflowServer) Watch(wfReq *WorkflowGetRequest, ws WorkflowService_WatchServer) error {
	wfClient, _, err := s.GetWFClient(ws.Context())
	if err != nil {
		return err
	}

	namespace := s.Namespace
	if wfReq.Namespace != "" {
		namespace = wfReq.Namespace
	}

	wfs, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Watch(v1.ListOptions{})
	if err != nil {
		return err
	}

	done := make(chan bool)
	go func() {
		for next := range wfs.ResultChan() {
			a := *next.Object.(*v1alpha1.Workflow)
			if wfReq.WorkflowName == "" || wfReq.WorkflowName == a.Name {
				err = ws.Send(&a)
				if err != nil {
					log.Warnf("Unable to send stream message: %v", err)
				}
			}
		}
		done <- true
	}()

	select {
	case <-ws.Context().Done():
		wfs.Stop()
	case <-done:
		wfs.Stop()
	}

	return nil
}

func (s *WorkflowServer) PodLogs(wfReq *WorkflowLogRequest, log WorkflowService_PodLogsServer) error {
	_, kubeClient, err := s.GetWFClient(log.Context())
	if err != nil {
		return err
	}

	namespace := s.Namespace
	if wfReq.Namespace != "" {
		namespace = wfReq.Namespace
	}
	containerName := "main"
	if wfReq.Container != "" {
		containerName = wfReq.Container
	}

	stream, err := kubeClient.CoreV1().Pods(namespace).GetLogs(wfReq.PodName, &corev1.PodLogOptions{
		Container:    containerName,
		Follow:       wfReq.LogOptions.Follow,
		Timestamps:   true,
		SinceSeconds: wfReq.LogOptions.SinceSeconds,
		SinceTime:    wfReq.LogOptions.SinceTime,
		TailLines:    wfReq.LogOptions.TailLines,
	}).Stream()

	if err == nil {
		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, " ")
			//logTime, err := time.Parse(time.RFC3339, parts[0])
			byt := []byte(parts[0])
			var logTime v1.Time
			err := logTime.UnmarshalText(byt)
			if err == nil {
				lines := strings.Join(parts[1:], " ")
				for _, line := range strings.Split(lines, "\r") {
					if line != "" {
						cnt := LogEntry{Content: line, TimeStamp: &logTime}
						_ = log.Send(&cnt)
					}
				}
			} else {
				cnt := LogEntry{Content: line, TimeStamp: &logTime}
				_ = log.Send(&cnt)
			}
		}
	}
	return err
}
