package workflow

import (
	"encoding/json"
	"errors"
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
	return &KubeService{
		Namespace:        Namespace,
		WfClientset:      wfClientset,
		KubeClientset:    kubeClientSet,
		EnableClientAuth: enableClientAuth,
	}
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

func (s *KubeService) Create(wfClient *versioned.Clientset, namespace string, wf *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	createdWf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
	if err != nil {
		log.Warnf("Create request is failed. Error: %s", err)
		return nil, err
	}

	log.Infof("Workflow created successfully. Name: %s", createdWf.Name)
	return createdWf, nil
}

func (s *KubeService) Get(wfClient *versioned.Clientset, namespace string, wfReq *WorkflowGetRequest) (*v1alpha1.Workflow, error) {
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, err
}

func (s *KubeService) List(wfClient *versioned.Clientset, namespace string, wfReq *WorkflowListRequest) (*v1alpha1.WorkflowList, error) {
	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return wfList, nil
}

func (s *KubeService) Delete(wfClient *versioned.Clientset, namespace string, wfReq *WorkflowDeleteRequest) (*WorkflowDeleteResponse, error) {
	err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Delete(wfReq.WorkflowName, &v1.DeleteOptions{})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *KubeService) Retry(wfClient *versioned.Clientset, kubeClient *kubernetes.Clientset, namespace string, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
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

func (s *KubeService) Resubmit(wfClient *versioned.Clientset, namespace string, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	newWF, err := util.FormulateResubmitWorkflow(wf, wfReq.Memoized)
	if err != nil {
		return nil, err
	}

	created, err := util.SubmitWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), wfClient, namespace, newWF, nil)
	if err != nil {
		return nil, err
	}

	return created, err
}

func (s *KubeService) Resume(wfClient *versioned.Clientset, namespace string, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	err := util.ResumeWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), wfReq.WorkflowName)
	if err != nil {
		log.Warnf("Failed to resume %s: %+v", wfReq.WorkflowName, err)
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *KubeService) Suspend(wfClient *versioned.Clientset, namespace string, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	err := util.SuspendWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), wfReq.WorkflowName)
	if err != nil {
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *KubeService) Terminate(wfClient *versioned.Clientset, namespace string, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	err := util.TerminateWorkflow(wfClient.ArgoprojV1alpha1().Workflows(namespace), wfReq.WorkflowName)
	if err != nil {
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}
