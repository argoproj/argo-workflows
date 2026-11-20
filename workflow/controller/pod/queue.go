// Package pod reconciles pods and takes care of gc events
package pod

import (
	"context"
	"slices"
	"syscall"
	"time"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/signal"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
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

	// Continue cleaning up existing finalizers when the capture flag is disabled.
	if pod.Finalizers != nil {
		finalizers := slices.Clone(pod.Finalizers)
		finalizers = slices.DeleteFunc(finalizers,
			func(s string) bool { return s == common.FinalizerPodStatus })
		if len(finalizers) != len(pod.Finalizers) {
			un.SetFinalizers(finalizers)
			un.SetResourceVersion(pod.ResourceVersion)
			if pod.UID != "" {
				un.SetUID(pod.UID)
			}
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
		_ = signal.Container(ctx, c.restConfig, pod, container.Name, sig)
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

func (c *Controller) patchPodForCleanupByUID(ctx context.Context, pods typedv1.PodInterface, podName, uid string) (bool, error) {
	if uid == "" {
		return false, nil
	}
	pod, err := pods.Get(ctx, podName, metav1.GetOptions{})
	if apierr.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	instanceRequirement := util.InstanceIDRequirement(c.config.InstanceID)
	if string(pod.UID) != uid || !instanceRequirement.Matches(labels.Set(pod.Labels)) {
		return false, nil
	}

	patch, err := c.getPodCleanupPatch(pod, false)
	if err != nil {
		return false, err
	}
	if patch == nil {
		return true, nil
	}

	_, err = pods.Patch(ctx, podName, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil && !apierr.IsNotFound(err) {
		return false, err
	}
	return true, nil
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

	namespace, podName, action, uid := parsePodCleanupKey(key)
	ctx, log := c.log.WithFields(logging.Fields{"key": key, "action": action, "namespace": namespace, "podName": podName}).InContext(ctx)
	log.Info(ctx, "cleaning up pod")
	err := func() error {
		switch action {
		case terminateContainers:
			pod, err := c.GetPod(namespace, podName)
			if err == nil && pod != nil && pod.Status.Phase == apiv1.PodPending {
				c.queuePodForCleanup(ctx, namespace, podName, deletePod)
			} else if terminationGracePeriod, err := c.signalContainers(ctx, namespace, podName, syscall.SIGTERM); err != nil {
				return err
			} else if terminationGracePeriod > 0 {
				c.queuePodForCleanupAfter(ctx, namespace, podName, killContainers, terminationGracePeriod)
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
		case deletePodByUID:
			pods := c.kubeclientset.CoreV1().Pods(namespace)
			if matched, err := c.patchPodForCleanupByUID(ctx, pods, podName, uid); err != nil || !matched {
				return err
			}
			propagation := metav1.DeletePropagationBackground
			podUID := types.UID(uid)
			err := pods.Delete(ctx, podName, metav1.DeleteOptions{
				PropagationPolicy:  &propagation,
				GracePeriodSeconds: c.config.PodGCGracePeriodSeconds,
				Preconditions:      &metav1.Preconditions{UID: &podUID},
			})
			if err != nil && !apierr.IsNotFound(err) {
				return err
			}
		case removeFinalizer:
			pods := c.kubeclientset.CoreV1().Pods(namespace)
			var err error
			if uid != "" {
				_, err = c.patchPodForCleanupByUID(ctx, pods, podName, uid)
			} else {
				err = c.patchPodForCleanup(ctx, pods, namespace, podName, false)
			}
			if err != nil {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		log.WithError(err).Warn(ctx, "failed to clean-up pod")
		if errorsutil.IsTransientErr(ctx, err) || apierr.IsConflict(err) {
			c.workqueue.AddRateLimited(key)
		}
	}
	return true
}

func (c *Controller) queuePodForCleanup(ctx context.Context, namespace string, podName string, action podCleanupAction) {
	c.log.WithFields(logging.Fields{"namespace": namespace, "podName": podName, "action": action}).Debug(ctx, "queueing pod for cleanup")
	c.workqueue.AddRateLimited(newPodCleanupKey(namespace, podName, action))
}

func (c *Controller) queuePodForCleanupAfter(ctx context.Context, namespace string, podName string, action podCleanupAction, duration time.Duration) {
	logCtx := c.log.WithFields(logging.Fields{"namespace": namespace, "podName": podName, "action": action, "after": duration})
	if duration > 0 {
		logCtx.Debug(ctx, "queueing pod for cleanup after")
		c.workqueue.AddAfter(newPodCleanupKey(namespace, podName, action), duration)
	} else {
		logCtx.Warn(ctx, "queueing pod for cleanup now, rather than delayed")
		c.workqueue.AddRateLimited(newPodCleanupKey(namespace, podName, action))
	}
}

func (c *Controller) queuePodForCleanupByUID(ctx context.Context, namespace string, podName string, action podCleanupAction, uid string) {
	c.log.WithFields(logging.Fields{"namespace": namespace, "podName": podName, "uid": uid, "action": action}).Info(ctx, "queueing pod for cleanup by UID")
	c.workqueue.AddRateLimited(newPodCleanupKeyWithUID(namespace, podName, action, uid))
}
