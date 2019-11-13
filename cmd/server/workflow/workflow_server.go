package workflow

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

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

type WorkflowServer struct {
	Namespace        string
	WfClientset      *versioned.Clientset
	KubeClientset    *kubernetes.Clientset
	EnableClientAuth bool
	Config           *config.WorkflowControllerConfig
	WfDBService      *DBService
	WfKubeService    *KubeService
}

func NewWorkflowServer(namespace string, wfClientset *versioned.Clientset, kubeClientSet *kubernetes.Clientset, config *config.WorkflowControllerConfig, enableClientAuth bool) *WorkflowServer {

	wfServer := WorkflowServer{Namespace: namespace, WfClientset: wfClientset, KubeClientset: kubeClientSet, EnableClientAuth: enableClientAuth}
	var err error
	if config != nil && config.Persistence != nil {
		wfServer.WfDBService.wfDBctx, err = wfServer.CreatePersistenceContext(namespace, kubeClientSet, config.Persistence)
	}
	if err != nil {
		log.Errorf("Error Creating DB Context. %v", err)
		return nil
	}
	return &wfServer
}

func (s *WorkflowServer) CreatePersistenceContext(namespace string, kubeClientSet *kubernetes.Clientset, config *config.PersistConfig) (*sqldb.WorkflowDBContext, error) {

	var wfDBCtx sqldb.WorkflowDBContext
	var err error

	//wfDBCtx.TableName = wfc.Config.Persistence.TableName
	wfDBCtx.NodeStatusOffload = config.NodeStatusOffload

	wfDBCtx.Session, wfDBCtx.TableName, err = sqldb.CreateDBSession(kubeClientSet, namespace, config)

	if err != nil {
		log.Errorf("Error in createPersistenceContext. %v", err)
		return nil, err
	}

	return &wfDBCtx, nil
}

func (s *WorkflowServer) GetWFClient(ctx context.Context) (*versioned.Clientset, *kubernetes.Clientset, error) {

	md, _ := metadata.FromIncomingContext(ctx)

	if !s.EnableClientAuth {
		return s.WfClientset, s.KubeClientset, nil
	}

	var restConfigStr, bearerToken string
	if len(md.Get(CLIENT_REST_CONFIG)) == 0 {
		return nil, nil, errors.New("Client kubeconfig is not found")
	}
	restConfigStr = md.Get(CLIENT_REST_CONFIG)[0]

	if len(md.Get(AUTH_TOKEN)) > 0 {
		bearerToken = md.Get(AUTH_TOKEN)[0]
	}

	restConfig := rest.Config{}

	err := json.Unmarshal([]byte(restConfigStr), &restConfig)
	if err != nil {
		return nil, nil, err
	}
	restConfig.BearerToken = string(bearerToken)

	// create the clientset
	wfClientset, err := wfclientset.NewForConfig(&restConfig)

	// create the clientset
	clientset, err := kubernetes.NewForConfig(&restConfig)

	if err != nil {
		log.Warnf("Failure to create WfClientset. ClientConfig: %s, Error: %s", restConfig, err)
		return nil, nil, err
	}

	return wfClientset, clientset, nil
}

func (s *WorkflowServer) Create(ctx context.Context, in *WorkflowCreateRequest) (*v1alpha1.Workflow, error) {

	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}
	if in.Workflow == nil {
		return nil, errors.New("Workflow body not found")
	}

	in.Workflow.Namespace = namespace

	wf, err := s.ApplyWorkflowOptions(in.Workflow, in.SubmitOptions)
	if err != nil {
		return nil, err
	}

	err = validate.ValidateWorkflow(wfClient, namespace, wf, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}

	if in.SubmitOptions != nil && in.SubmitOptions.ServerDryRun {
		return util.CreateServerDryRun(wf, wfClient)
	}

	wf, err = s.WfKubeService.Create(wfClient, namespace, in.Workflow)

	if err != nil {
		log.Warnf("Create request is failed. Error: %s", err)
		return nil, err
	}
	log.Info("Workflow created successfully. Name: %s", wf.Name)
	return wf, nil
}

func (s *WorkflowServer) Get(ctx context.Context, in *WorkflowGetRequest) (*v1alpha1.Workflow, error) {

	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	var wf *v1alpha1.Workflow

	if s.WfDBService != nil {
		wf, err = s.WfDBService.Get(in.WorkflowName, in.Namespace)
	} else {
		wf, err = wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.WorkflowName, v1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}

	return wf, err
}

func (s *WorkflowServer) List(ctx context.Context, in *WorkflowListRequest) (*v1alpha1.WorkflowList, error) {

	namespace := s.Namespace

	if in.Namespace != "" {
		namespace = in.Namespace
	}
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	listOpt := in.ListOptions
	var wfList *v1alpha1.WorkflowList
	if s.WfDBService != nil {
		wfList, err = s.WfDBService.List(namespace, uint(listOpt.Limit), "")
	} else {

		wfList, err = wfClient.ArgoprojV1alpha1().Workflows(namespace).List(v1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	return wfList, nil

}

func (s *WorkflowServer) Delete(ctx context.Context, in *WorkflowDeleteRequest) (*WorkflowDeleteResponse, error) {
	namespace := s.Namespace

	if in.Namespace != "" {
		namespace = in.Namespace
	}
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	if s.WfDBService != nil {
		err = s.WfDBService.Delete(in.WorkflowName, in.Namespace)
		if err != nil {
			return nil, err
		}
	}
	err = wfClient.ArgoprojV1alpha1().Workflows(namespace).Delete(in.WorkflowName, &v1.DeleteOptions{})

	if err != nil {
		return nil, err
	}
	var rsp WorkflowDeleteResponse
	rsp.WorkflowName = in.WorkflowName
	rsp.Status = "Deleted"

	return &rsp, nil
}

func (s *WorkflowServer) Retry(ctx context.Context, in *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}

	wfClient, kubeClient, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.WorkflowName, v1.GetOptions{})

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
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}

	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.WorkflowName, v1.GetOptions{})

	newWF, err := util.FormulateResubmitWorkflow(wf, in.Memoized)

	created, err := util.SubmitWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), wfClient, namespace, newWF, nil)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return created, err
}

func (s *WorkflowServer) Resume(ctx context.Context, in *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}

	err = util.ResumeWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), in.WorkflowName)
	if err != nil {
		log.Warnf("Failed to resume %s: %+v", in.WorkflowName, err)
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.WorkflowName, v1.GetOptions{})

	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *WorkflowServer) Suspend(ctx context.Context, in *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}

	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	err = util.SuspendWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), in.WorkflowName)

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *WorkflowServer) Terminate(ctx context.Context, in *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	err = util.TerminateWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), in.WorkflowName)

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *WorkflowServer) Lint(ctx context.Context, in *WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
	wfClient, _, err := s.GetWFClient(ctx)
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}

	err = validate.ValidateWorkflow(wfClient, namespace, in.Workflow, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}
	return in.Workflow, nil
}

func (s *WorkflowServer) Watch(in *WorkflowGetRequest, ws WorkflowService_WatchServer) error {
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}
	wfClient, _, err := s.GetWFClient(ws.Context())

	if err != nil {
		return err
	}
	wfs, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Watch(v1.ListOptions{})
	if err != nil {
		return err
	}
	done := make(chan bool)
	go func() {
		for next := range wfs.ResultChan() {
			a := *next.Object.(*v1alpha1.Workflow)
			if in.WorkflowName == "" || in.WorkflowName == a.Name {

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

func (s *WorkflowServer) PodLogs(in *WorkflowLogRequest, log WorkflowService_PodLogsServer) error {

	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}
	containerName := "main"
	if in.Container != "" {
		containerName = in.Container
	}
	_, kubeClient, err := s.GetWFClient(log.Context())

	stream, err := kubeClient.CoreV1().Pods(namespace).GetLogs(in.PodName, &corev1.PodLogOptions{
		Container:    containerName,
		Follow:       in.LogOptions.Follow,
		Timestamps:   true,
		SinceSeconds: in.LogOptions.SinceSeconds,
		SinceTime:    in.LogOptions.SinceTime,
		TailLines:    in.LogOptions.TailLines,
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
						log.Send(&cnt)
					}
				}
			} else {
				cnt := LogEntry{Content: line, TimeStamp: &logTime}
				log.Send(&cnt)
			}
		}
	}
	return err
}

func (s *WorkflowServer) ApplyWorkflowOptions(wf *v1alpha1.Workflow, opts *SubmitOptions) (*v1alpha1.Workflow, error) {
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
