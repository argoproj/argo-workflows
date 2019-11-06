package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/config"
	"github.com/argoproj/argo/workflow/util"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type WorkflowServer struct {
	Namespace        string
	WfClientset      *versioned.Clientset
	KubeClientset    *kubernetes.Clientset
	EnableClientAuth bool
	Config 			 *config.WorkflowControllerConfig
	WfDBService      *DBService
	WfKubeService    *KubeService
}



func NewWorkflowServer(namespace string, wfClientset *versioned.Clientset, kubeClientSet *kubernetes.Clientset, config *config.WorkflowControllerConfig, enableClientAuth bool) (*WorkflowServer) {

	wfServer := WorkflowServer{Namespace: namespace, WfClientset: wfClientset, KubeClientset: kubeClientSet, EnableClientAuth: enableClientAuth}
	var err error
	wfServer.WfDBService.wfDBctx, err = wfServer.CreatePersistenceContext(namespace, kubeClientSet,config.Persistence)

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

	if s.EnableClientAuth {
		return s.WfClientset, s.KubeClientset, nil
	}

	var restConfigStr, bearerToken string
	if len(md.Get(CLIENT_REST_CONFIG)) == 0 {
		return nil,nil, errors.New("Client kubeconfig is not found")
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
	wf, err := s.WfKubeService.Create(wfClient,namespace,in.Workflows)

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
	}else {
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
		wfList, err = s.WfDBService.List(namespace, uint(listOpt.Limit),"")
	}else {
		wfList, err = wfClient.ArgoprojV1alpha1().Workflows(namespace).List(*listOpt)
	}
	if err != nil {
		fmt.Println(err)
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
	err = wfClient.ArgoprojV1alpha1().Workflows(namespace).Delete(in.WorkflowName, &v1.DeleteOptions{})
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	//msgStr := fmt.Sprint("Workflow '%s' deleted\n", in.WorkflowName)
	return nil, nil
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