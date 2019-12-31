package workflow

import (
	"bufio"
	"strings"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/util"
)

type kubeService struct {
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

func (s *kubeService) Get(wfClient versioned.Interface, namespace string, workflowName string, getOption *metav1.GetOptions) (*v1alpha1.Workflow, error) {
	wfGetOption := metav1.GetOptions{}
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
	var listOption = metav1.ListOptions{}
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
	err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Delete(wfReq.WorkflowName, &metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}
	return &WorkflowDeleteResponse{WorkflowName: wfReq.WorkflowName, Status: "Deleted"}, nil
}

func (s *kubeService) Retry(wfClient versioned.Interface, kubeClient kubernetes.Interface, namespace string, wfReq *WorkflowUpdateRequest) (*v1alpha1.Workflow, error) {
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, metav1.GetOptions{})
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
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, metav1.GetOptions{})
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

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, metav1.GetOptions{})
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

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, metav1.GetOptions{})
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

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(wfReq.WorkflowName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *kubeService) PodLogs(kubeClient kubernetes.Interface, wfReq *WorkflowLogRequest, log WorkflowService_PodLogsServer) error {
	stream, err := kubeClient.CoreV1().Pods(wfReq.Namespace).GetLogs(wfReq.PodName, wfReq.LogOptions).Stream()

	if err == nil {
		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, " ")
			//logTime, err := time.Parse(time.RFC3339, parts[0])
			byt := []byte(parts[0])
			var logTime metav1.Time
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
