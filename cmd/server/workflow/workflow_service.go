package workflow

import (
	"bufio"
	"encoding/json"
	"errors"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/cmd/server/common"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/util"
)

type kubeService struct {
	Namespace        string
	WfClientset      *versioned.Clientset
	KubeClientset    *kubernetes.Clientset
	EnableClientAuth bool
}

func NewKubeServer(Namespace string, wfClientset *wfclientset.Clientset, kubeClientSet *kubernetes.Clientset, enableClientAuth bool) *kubeService {
	return &kubeService{
		Namespace:        Namespace,
		WfClientset:      wfClientset,
		KubeClientset:    kubeClientSet,
		EnableClientAuth: enableClientAuth,
	}
}

func (s *kubeService) GetWFClient(ctx context.Context) (*versioned.Clientset, *kubernetes.Clientset, error) {
	md, _ := metadata.FromIncomingContext(ctx)

	if s.EnableClientAuth {
		return s.WfClientset, s.KubeClientset, nil
	}

	var restConfigStr, bearerToken string
	if len(md.Get(common.CLIENT_REST_CONFIG)) == 0 {
		return nil, nil, errors.New("Client kubeconfig is not found")
	}
	restConfigStr = md.Get(common.CLIENT_REST_CONFIG)[0]

	if len(md.Get(common.AUTH_TOKEN)) > 0 {
		bearerToken = md.Get(common.AUTH_TOKEN)[0]
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

func (s *kubeService) Create(wfClient versioned.Interface, namespace string, wf *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	createdWf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Create(wf)
	if err != nil {
		log.Warnf("Create request is failed. Error: %s", err)
		return nil, err
	}

	log.Infof("Workflow created successfully. Name: %s", createdWf.Name)
	return createdWf, nil
}

func (s *kubeService) Get(wfClient versioned.Interface, namespace string, workflowName string, getOption *v1.GetOptions) (*v1alpha1.Workflow, error) {
	wfGetOption := v1.GetOptions{}
	if getOption != nil {
		wfGetOption = *getOption
	}
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(workflowName, wfGetOption)
	if err != nil {
		return nil, err
	}
	return wf, err
}

func (s *kubeService) List(wfClient versioned.Interface, namespace string, wfReq *WorkflowListRequest) (*v1alpha1.WorkflowList, error) {

	var listOption v1.ListOptions = v1.ListOptions{}
	if wfReq.ListOptions != nil {
		listOption = *wfReq.ListOptions
	}

	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).List(listOption)
	if err != nil {
		return nil, err
	}
	return wfList, nil
}

func (s *kubeService) Delete(wfClient versioned.Interface, namespace string, wfReq *WorkflowDeleteRequest) (*WorkflowDeleteResponse, error) {
	err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Delete(wfReq.WorkflowName, &v1.DeleteOptions{})
	if err != nil {
		return nil, err
	}
	return &WorkflowDeleteResponse{WorkflowName: wfReq.WorkflowName, Status: "deleted"}, nil
}

func (s *kubeService) Retry(wfClient versioned.Interface, kubeClient kubernetes.Interface, namespace string, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
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

func (s *kubeService) Resubmit(wfClient versioned.Interface, namespace string, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
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

func (s *kubeService) Resume(wfClient versioned.Interface, namespace string, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
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

func (s *kubeService) Suspend(wfClient versioned.Interface, namespace string, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
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

func (s *kubeService) Terminate(wfClient versioned.Interface, namespace string, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
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

func (s *kubeService) WatchWorkflow(wfClient versioned.Interface, wfReq *WorkflowGetRequest, ws WorkflowService_WatchWorkflowServer) error {
	wfs, err := wfClient.ArgoprojV1alpha1().Workflows(wfReq.Namespace).Watch(v1.ListOptions{})
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
					break
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

func (s *kubeService) PodLogs(kubeClient kubernetes.Interface, wfReq *WorkflowLogRequest, log WorkflowService_PodLogsServer) error {
	stream, err := kubeClient.CoreV1().Pods(wfReq.Namespace).GetLogs(wfReq.PodName, &corev1.PodLogOptions{
		Container:    wfReq.Container,
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
