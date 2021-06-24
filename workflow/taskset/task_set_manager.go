package taskset

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	informer "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	argowait "github.com/argoproj/argo-workflows/v3/util/wait"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
)

type QueueWorkflowFunc func(string)

type WorkflowTaskSetManager struct {
	wfTaskSetClient   v1alpha1.ArgoprojV1alpha1Interface
	wfTaskSetInformer informer.WorkflowTaskSetInformer
	queueWorkflowFunc QueueWorkflowFunc
	wfTaskSetQueue    workqueue.RateLimitingInterface
}

func NewWorkflowTaskSetManager(client v1alpha1.ArgoprojV1alpha1Interface, informer informer.WorkflowTaskSetInformer, queueFunc QueueWorkflowFunc, metrics *metrics.Metrics) *WorkflowTaskSetManager {
	return &WorkflowTaskSetManager{
		wfTaskSetClient:   client,
		wfTaskSetInformer: informer,
		queueWorkflowFunc: queueFunc,
		wfTaskSetQueue:    metrics.RateLimiterWithBusyWorkers(workqueue.DefaultControllerRateLimiter(), "wf_task_set_queue"),
	}
}

func (wfts WorkflowTaskSetManager) CreateTaskSet(ctx context.Context, wf *wfv1.Workflow, nodeId string, tmpl wfv1.Template) error {
	key := fmt.Sprintf("%s/%s", wf.Namespace, wf.Name)
	log.WithField("workflow", wf.Name).WithField("namespace", wf.Namespace).WithField("TaskSet", key).Infof("Creating TaskSet")

	taskSet := wfv1.WorkflowTaskSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       workflow.WorkflowTaskSetKind,
			APIVersion: workflow.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: wf.Namespace,
			Name:      wf.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: wf.APIVersion,
					Kind:       wf.Kind,
					UID:        wf.UID,
					Name:       wf.Name,
				},
			},
		},
		Spec: wfv1.WorkflowTaskSetSpec{
			Tasks: []wfv1.Task{
				{
					NodeID:   nodeId,
					Template: tmpl,
				},
			},
		},
	}
	log.WithField("workflow", wf.Name).WithField("namespace", wf.Namespace).WithField("TaskSet", key).Debug("creating new taskset")
	err := argowait.Backoff(retry.DefaultBackoff, func() (bool, error) {
		var err error
		_, err = wfts.wfTaskSetClient.WorkflowTaskSets(wf.Namespace).Create(ctx, &taskSet, metav1.CreateOptions{})
		return err == nil || apierr.IsConflict(err) || apierr.IsAlreadyExists(err), err
	})
	if apierr.IsConflict(err) || apierr.IsAlreadyExists(err) {
		taskSet, err := wfts.wfTaskSetClient.WorkflowTaskSets(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Error(fmt.Errorf("failed to get TaskSet '%s' from informer index", key))
			return err
		}
		task := wfv1.Task{
			NodeID:   nodeId,
			Template: tmpl,
		}
		if existing := taskSet.Spec.Tasks.GetTask(nodeId); existing == nil {
			log.WithField("workflow", wf.Name).WithField("namespace", wf.Namespace).WithField("TaskSet", key).Debug("updating existing taskset")
			taskSet.Spec.Tasks = append(taskSet.Spec.Tasks, task)
			err = argowait.Backoff(retry.DefaultBackoff, func() (bool, error) {
				var err error
				_, err = wfts.wfTaskSetClient.WorkflowTaskSets(wf.Namespace).Update(ctx, taskSet, metav1.UpdateOptions{})
				return err == nil, err
			})

			if err != nil {
				return fmt.Errorf("failed to update TaskSet. %v", err)
			}
		}
	} else if err != nil {
		log.WithError(err).WithField("workflow", wf.Name).WithField("namespace", wf.Namespace).Error("Failed to create WorkflowTaskSet")
		return err
	}
	return nil
}

func (wfts WorkflowTaskSetManager) Run(ctx context.Context) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)
	defer wfts.wfTaskSetQueue.ShutDown()
	log.Infof("Starting TaskSet manager")
	wfts.wfTaskSetInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					wfts.queueWorkflowFunc(key)
				}
			},
		})
	go wfts.wfTaskSetInformer.Informer().Run(ctx.Done())

	<-ctx.Done()
}
