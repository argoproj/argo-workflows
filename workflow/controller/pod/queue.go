// Package pod reconciles pods and takes care of gc events
package pod

import (
	"context"
	"os"
	"slices"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/signal"
)

func (c *Controller) runPodCleanup(ctx context.Context) {
	for c.processNextPodCleanupItem(ctx) {
	}
}

func (c *Controller) getPodCleanupPatch(pod *apiv1.Pod, labelPodCompleted bool) ([]byte, error) {
	un := unstructured.Unstructured{}
	if labelPodCompleted {
		un.SetLabels(map[string]string{common.LabelKeyCompleted: "true"})
	}

	finalizerEnabled := os.Getenv(common.EnvVarPodStatusCaptureFinalizer) == "true"
	if finalizerEnabled && pod.Finalizers != nil {
		finalizers := slices.Clone(pod.Finalizers)
		finalizers = slices.DeleteFunc(finalizers,
			func(s string) bool { return s == common.FinalizerPodStatus })
		if len(finalizers) != len(pod.Finalizers) {
			un.SetFinalizers(finalizers)
			un.SetResourceVersion(pod.ObjectMeta.ResourceVersion)
		}
	}

	// if there was nothing to patch (no-op)
	if len(un.Object) == 0 {
		return nil, nil
	}

	return un.MarshalJSON()
}

// signalContainers signals all containers of a pod
func (c *Controller) signalContainers(ctx context.Context, namespace string, podName string, sig syscall.Signal) (time.Duration, error) {
	pod, err := c.GetPod(namespace, podName)
	if pod == nil || err != nil {
		return 0, err
	}

	for _, container := range pod.Status.ContainerStatuses {
		if container.State.Running == nil {
			continue
		}
		// problems are already logged at info level, so we just ignore errors here
		_ = signal.SignalContainer(ctx, c.restConfig, pod, container.Name, sig)
	}
	if pod.Spec.TerminationGracePeriodSeconds == nil {
		return 30 * time.Second, nil
	}
	return time.Duration(*pod.Spec.TerminationGracePeriodSeconds) * time.Second, nil
}

func (c *Controller) patchPodForCleanup(ctx context.Context, pods typedv1.PodInterface, namespace, podName string, labelPodCompleted bool) error {
	pod, err := c.GetPod(namespace, podName)
	// err is always nil in all kind of caches for now
	if err != nil {
		return err
	}
	// if pod is nil, it must have been deleted
	if pod == nil {
		return nil
	}

	patch, err := c.getPodCleanupPatch(pod, labelPodCompleted)
	if err != nil {
		return err
	}
	if patch == nil {
		return nil
	}

	_, err = pods.Patch(ctx, podName, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil && !apierr.IsNotFound(err) {
		return err
	}

	return nil
}

// all pods will ultimately be cleaned up by either deleting them, or labelling them
func (c *Controller) processNextPodCleanupItem(ctx context.Context) bool {
	key, quit := c.workqueue.Get()
	if quit {
		return false
	}

	defer func() {
		c.workqueue.Forget(key)
		c.workqueue.Done(key)
	}()

	namespace, podName, action := parsePodCleanupKey(podCleanupKey(key))
	logCtx := c.log.WithFields(logrus.Fields{"key": key, "action": action, "namespace": namespace, "podName": podName})
	logCtx.Info("cleaning up pod")
	err := func() error {
		switch action {
		case terminateContainers:
			pod, err := c.GetPod(namespace, podName)
			if err == nil && pod != nil && pod.Status.Phase == apiv1.PodPending {
				c.queuePodForCleanup(namespace, podName, deletePod)
			} else if terminationGracePeriod, err := c.signalContainers(ctx, namespace, podName, syscall.SIGTERM); err != nil {
				return err
			} else if terminationGracePeriod > 0 {
				c.queuePodForCleanupAfter(namespace, podName, killContainers, terminationGracePeriod)
			}
		case killContainers:
			if _, err := c.signalContainers(ctx, namespace, podName, syscall.SIGKILL); err != nil {
				return err
			}
		case labelPodCompleted:
			pods := c.kubeclientset.CoreV1().Pods(namespace)
			if err := c.patchPodForCleanup(ctx, pods, namespace, podName, true); err != nil {
				return err
			}
		case deletePod:
			pods := c.kubeclientset.CoreV1().Pods(namespace)
			if err := c.patchPodForCleanup(ctx, pods, namespace, podName, false); err != nil {
				return err
			}
			propagation := metav1.DeletePropagationBackground
			err := pods.Delete(ctx, podName, metav1.DeleteOptions{
				PropagationPolicy:  &propagation,
				GracePeriodSeconds: c.config.PodGCGracePeriodSeconds,
			})
			if err != nil && !apierr.IsNotFound(err) {
				return err
			}
		case removeFinalizer:
			pods := c.kubeclientset.CoreV1().Pods(namespace)
			if err := c.patchPodForCleanup(ctx, pods, namespace, podName, false); err != nil {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		logCtx.WithError(err).Warn("failed to clean-up pod")
		if errorsutil.IsTransientErr(err) || apierr.IsConflict(err) {
			c.workqueue.AddRateLimited(key)
		}
	}
	return true
}

func (c *Controller) queuePodForCleanup(namespace string, podName string, action podCleanupAction) {
	c.log.WithFields(logrus.Fields{"namespace": namespace, "podName": podName, "action": action}).Info("queueing pod for cleanup")
	c.workqueue.AddRateLimited(newPodCleanupKey(namespace, podName, action))
}

func (c *Controller) queuePodForCleanupAfter(namespace string, podName string, action podCleanupAction, duration time.Duration) {
	logCtx := c.log.WithFields(logrus.Fields{"namespace": namespace, "podName": podName, "action": action, "after": duration})
	if duration > 0 {
		logCtx.Info("queueing pod for cleanup after")
		c.workqueue.AddAfter(newPodCleanupKey(namespace, podName, action), duration)
	} else {
		logCtx.Warn("queueing pod for cleanup now, rather than delayed")
		c.workqueue.AddRateLimited(newPodCleanupKey(namespace, podName, action))
	}
}
