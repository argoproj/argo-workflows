package workflow

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/util"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type Server struct {
	Namespace        string
	Clientset        versioned.Clientset
	EnableClientAuth bool
}

func NewServer(Namespace string, clientset versioned.Clientset, enableClientAuth bool) WorkflowServiceServer {
	return &Server{Namespace: Namespace, Clientset: clientset, EnableClientAuth: enableClientAuth}
}

func (s *Server) GetClientSet(md metadata.MD) (*versioned.Clientset, error) {

	if s.EnableClientAuth {
		return &s.Clientset, nil
	}

	var restConfigStr, bearerToken string

	restConfigStr = md.Get(CLIENT_REST_CONFIG)[0]

	bearerToken = md.Get(AUTH_TOKEN)[0]

	restConfig := rest.Config{}
	err := json.Unmarshal([]byte(restConfigStr), &restConfig)
	if err != nil {
		return nil, err
	}
	restConfig.BearerToken = string(bearerToken)
	//restConfig :=rest.Config{
	//	// TODO: switch to using cluster DNS.
	//	Host:            host,
	//	TLSClientConfig: tlsClientConfig,
	//	BearerToken:     string(bearerToken),
	//
	//	}

	fmt.Println(restConfigStr)
	// create the clientset
	clientset, err := wfclientset.NewForConfig(&restConfig)

	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func (s *Server) Create(ctx context.Context, in *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {

	md, _ := metadata.FromIncomingContext(ctx)
	clientset, err := s.GetClientSet(md)

	if clientset == nil {
		return nil, nil
	}
	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}
	wf, err := s.Clientset.ArgoprojV1alpha1().Workflows(namespace).Create(in)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return wf, nil
}

func (s *Server) Get(ctx context.Context, in *WorkflowQuery) (*v1alpha1.Workflow, error) {

	namespace := s.Namespace
	if in.Namespace != "" {
		namespace = in.Namespace
	}
	md, _ := metadata.FromIncomingContext(ctx)

	clientset, err := s.GetClientSet(md)

	if clientset == nil {
		return nil, nil

	}
	wf, err := clientset.ArgoprojV1alpha1().Workflows(namespace).Get(in.Name, v1.GetOptions{})

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return wf, err
}

func (s *Server) List(ctx context.Context, in *WorkflowQuery) (*WorkflowListResponse, error) {

	namespace := s.Namespace

	if in.Namespace != "" {
		namespace = in.Namespace
	}

	wfList, err := s.Clientset.ArgoprojV1alpha1().Workflows(namespace).List(v1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}

	//fmt.Println(wfList)
	var wfListItem []*v1alpha1.Workflow
	for idx, _ := range wfList.Items {
		wfListItem = append(wfListItem, &wfList.Items[idx])
	}
	var wfListRsp = WorkflowListResponse{}
	wfListRsp.Workflows = wfListItem
	fmt.Println(wfListRsp)
	return &wfListRsp, nil

}

func (s *Server) Delete(ctx context.Context, in *WorkflowQuery) (*WorkflowResponse, error) {

	return nil, nil
}

func (s *Server) Retry(ctx context.Context, in *WorkflowUpdateQuery) (*v1alpha1.Workflow, error) {
	//namespace := s.Namespace
	//if in.Workflow.Namespace != "" {
	//	namespace = in.Workflow.Namespace
	//}
	//kubeClient := commonutil.InitKubeClient()
	//
	////wf, err :=  util.RetryWorkflow(kubeClient., s.Clientset.ArgoprojV1alpha1().Workflows(namespace),in.Workflow)
	//
	//if err != nil {
	//	fmt.Println(err)
	//	return nil, err
	//}
	//
	//return wf, err
	return nil, nil
}

func (s *Server) Resubmit(ctx context.Context, in *WorkflowUpdateQuery) (*v1alpha1.Workflow, error) {
	namespace := s.Namespace
	if in.Workflow.Namespace != "" {
		namespace = in.Workflow.Namespace
	}

	var wfClientset *versioned.Clientset

	wf, err := s.Clientset.ArgoprojV1alpha1().Workflows(namespace).Get(in.Workflow.Name, v1.GetOptions{})
	//errors.CheckError(err)
	newWF, err := util.FormulateResubmitWorkflow(wf, in.Memoized)
	//errors.CheckError(err)
	created, err := util.SubmitWorkflow(s.Clientset.ArgoprojV1alpha1().Workflows(namespace), wfClientset, namespace, newWF, nil)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return created, err
}

func (s *Server) Resume(ctx context.Context, in *WorkflowUpdateQuery) (*v1alpha1.Workflow, error) {
	return nil, nil
}

func (s *Server) Suspend(ctx context.Context, in *WorkflowUpdateQuery) (*v1alpha1.Workflow, error) {
	return nil, nil
}

func (s *Server) Terminate(ctx context.Context, in *WorkflowUpdateQuery) (*v1alpha1.Workflow, error) {
	return nil, nil
}
