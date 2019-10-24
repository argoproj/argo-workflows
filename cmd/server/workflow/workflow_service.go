package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/util"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Server struct {
	Namespace        string
	WfClientset      *versioned.Clientset
	KubeClientset    *kubernetes.Clientset
	EnableClientAuth bool
}

func NewServer(Namespace string, wfClientset *versioned.Clientset, kubeClientSet *kubernetes.Clientset, enableClientAuth bool) WorkflowServiceServer {
	return &Server{Namespace: Namespace, WfClientset: wfClientset, KubeClientset: kubeClientSet, EnableClientAuth: enableClientAuth}
}

func (s *Server) GetWFClient(ctx context.Context) (*versioned.Clientset, *kubernetes.Clientset, error) {

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

	fmt.Println(restConfigStr)
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

func (s *Server) Create(ctx context.Context, in *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {

	wfClient, _, err := s.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Create(in)
	if err != nil {
		log.Warnf("Create request is failed. Error: %s", err)
		return nil, err
	}
	log.Info("Workflow created successfully. Name: %s", wf.Name)
	return wf, nil
}

func (s *Server) Get(ctx context.Context, in *WorkflowGetRequest) (*v1alpha1.Workflow, error) {

	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.Wfname, v1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return wf, err
}

func (s *Server) List(ctx context.Context, in *WorkflowListRequest) (*v1alpha1.WorkflowList, error) {

	namespace := s.Namespace

	if in.Namespace != "" {
		namespace = in.Namespace
	}
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}

	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).List(v1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}

	return wfList, nil

}



func (s *Server) Delete(ctx context.Context, in *WorkflowDeleteRequest) (*WorkflowDeleteResponse, error) {
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

	return fmt.Sprint("Workflow '%s' deleted\n", in.WorkflowName), nil
}

func (s *Server) Retry(ctx context.Context, in *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}

	wfClient, kubeClient, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.Wfname, v1.GetOptions{})

	if err != nil {
		return nil, err
	}

	wf, err = util.RetryWorkflow(kubeClient, wfClient.ArgoprojV1alpha1().Workflows(namespace), wf)

	if err != nil {
		return nil, err
	}
	return wf, err
}

func (s *Server) Resubmit(ctx context.Context, in *WorkflowUpdateQuery) (*v1alpha1.Workflow, error) {
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}

	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.Workflow.Name, v1.GetOptions{})

	newWF, err := util.FormulateResubmitWorkflow(wf, in.Memoized)

	created, err := util.SubmitWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), wfClient, namespace, newWF, nil)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return created, err
}

func (s *Server) Resume(ctx context.Context, in *WorkflowUpdateQuery) (*v1alpha1.Workflow, error) {
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}

	err = util.ResumeWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), in.Wfname)
	if err != nil {
		log.Warnf("Failed to resume %s: %+v", in.Wfname, err)
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.Wfname, v1.GetOptions{})

	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *Server) Suspend(ctx context.Context, in *WorkflowUpdateQuery) (*v1alpha1.Workflow, error) {
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}

	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	err = util.SuspendWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), in.Wfname)

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.Wfname, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *Server) Terminate(ctx context.Context, in *WorkflowUpdateQuery) (*v1alpha1.Workflow, error) {
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}
	wfClient, _, err := s.GetWFClient(ctx)

	if err != nil {
		return nil, err
	}
	err = util.TerminateWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), in.Wfname)

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.Wfname, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}
