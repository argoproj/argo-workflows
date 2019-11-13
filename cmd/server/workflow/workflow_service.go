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

type KubeService struct {
	Namespace        string
	WfClientset      *versioned.Clientset
	KubeClientset    *kubernetes.Clientset
	EnableClientAuth bool
}

func NewKubeServer(Namespace string, wfClientset *wfclientset.Clientset, kubeClientSet *kubernetes.Clientset, enableClientAuth bool) *KubeService {
	return &KubeService{Namespace: Namespace, WfClientset: wfClientset, KubeClientset: kubeClientSet, EnableClientAuth: enableClientAuth}
}

func (s *KubeService) GetWFClient(ctx context.Context) (*versioned.Clientset, *kubernetes.Clientset, error) {

	md, _ := metadata.FromIncomingContext(ctx)

	if s.EnableClientAuth {
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

func (s *KubeService) Create(wfClient *versioned.Clientset, namespace string, in *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Create(in)
	if err != nil {
		log.Warnf("Create request is failed. Error: %s", err)
		return nil, err
	}
	log.Info("Workflow created successfully. Name: %s", wf.Name)
	return wf, nil
}

func (s *KubeService) Get(wfClient *versioned.Clientset, namespace string, in *WorkflowGetRequest) (*v1alpha1.Workflow, error) {

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.WorkflowName, v1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return wf, err
}

func (s *KubeService) List(wfClient *versioned.Clientset, namespace string, in *WorkflowListRequest) (*v1alpha1.WorkflowList, error) {

	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).List(v1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}

	return wfList, nil

}

func (s *KubeService) Delete(wfClient *versioned.Clientset, namespace string, in *WorkflowDeleteRequest) (*WorkflowDeleteResponse, error) {

	err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Delete(in.WorkflowName, &v1.DeleteOptions{})
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	//fmt.Sprint("Workflow '%s' deleted\n", in.WorkflowName)
	return nil , nil
}

func (s *KubeService) Retry(wfClient *versioned.Clientset, kubeClient *kubernetes.Clientset, namespace string, in *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {

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

func (s *KubeService) Resubmit(wfClient *versioned.Clientset, namespace string, in *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.WorkflowName, v1.GetOptions{})

	newWF, err := util.FormulateResubmitWorkflow(wf, in.Memoized)

	created, err := util.SubmitWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), wfClient, namespace, newWF, nil)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return created, err
}

func (s *KubeService) Resume(wfClient *versioned.Clientset, namespace string, in *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {

	err := util.ResumeWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), in.WorkflowName)
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

func (s *KubeService) Suspend(wfClient *versioned.Clientset, namespace string, in *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {

	err := util.SuspendWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), in.WorkflowName)

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *KubeService) Terminate(wfClient *versioned.Clientset, namespace string, in *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {

	err := util.TerminateWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), in.WorkflowName)

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(in.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}
