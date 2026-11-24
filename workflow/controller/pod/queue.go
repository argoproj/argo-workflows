// Package pod reconciles pods and takes care of gc events
package pod

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"syscall"
	"time"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/signal"
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
			un.SetResourceVersion(pod.ResourceVersion)
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

// podCleanupRequest tracks one logical cleanup key across ready, running, and delayed entries.
type podCleanupRequest struct {
	entry      string
	readyAt    time.Time
	pendingAt  time.Time
	processing bool
	retrying   bool
}

func (c *Controller) addPodCleanup(key string, delay time.Duration) {
	c.cleanupMu.Lock()
	defer c.cleanupMu.Unlock()

	readyAt := time.Now().Add(max(delay, 0))
	if request := c.cleanupRequests[key]; request != nil {
		switch {
		case request.retrying:
			return
		case request.processing:
			if request.pendingAt.IsZero() || readyAt.Before(request.pendingAt) {
				request.pendingAt = readyAt
			}
			return
		case !readyAt.Before(request.readyAt):
			return
		case delay > 0:
			request.readyAt = readyAt
			c.workqueue.AddAfter(request.entry, time.Until(readyAt))
			return
		}
	}
	c.schedulePodCleanup(key, readyAt, false)
}

// schedulePodCleanup must hold cleanupMu. A new entry lets workers ignore timers
// left behind when immediate work supersedes a delay or a failure starts a retry.
func (c *Controller) schedulePodCleanup(key string, readyAt time.Time, retry bool) {
	if c.cleanupRequests == nil {
		c.cleanupRequests = make(map[string]*podCleanupRequest)
	}
	c.cleanupSequence++
	entry := fmt.Sprintf("%d:%s", c.cleanupSequence, key)
	c.cleanupRequests[key] = &podCleanupRequest{entry: entry, readyAt: readyAt, retrying: retry}
	delay := time.Until(readyAt)
	switch {
	case retry:
		c.workqueue.AddRateLimited(entry)
		// Preserve the cleanup queue's existing per-attempt limiter reset.
		c.workqueue.Forget(entry)
	case delay > 0:
		c.workqueue.AddAfter(entry, delay)
	default:
		c.workqueue.Add(entry)
	}
}

func (c *Controller) startPodCleanup(key, entry string) bool {
	c.cleanupMu.Lock()
	defer c.cleanupMu.Unlock()

	request := c.cleanupRequests[key]
	if request == nil || request.entry != entry {
		return false
	}
	request.processing = true
	request.retrying = false
	return true
}

func (c *Controller) finishPodCleanup(key string, retry bool) {
	c.cleanupMu.Lock()
	defer c.cleanupMu.Unlock()

	pendingAt := c.cleanupRequests[key].pendingAt
	delete(c.cleanupRequests, key)
	if retry || !pendingAt.IsZero() {
		c.schedulePodCleanup(key, pendingAt, retry)
	}
}

// all pods will ultimately be cleaned up by either deleting them, or labelling them
func (c *Controller) processNextPodCleanupItem(ctx context.Context) bool {
	entry, quit := c.workqueue.Get()
	if quit {
		return false
	}
	defer c.workqueue.Done(entry)

	_, key, _ := strings.Cut(entry, ":")
	if !c.startPodCleanup(key, entry) {
		return true
	}
	retry := false
	defer func() { c.finishPodCleanup(key, retry) }()

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
			if err := c.patchPodForCleanup(ctx, pods, namespace, podName, false); err != nil {
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
			if err := c.patchPodForCleanup(ctx, pods, namespace, podName, false); err != nil {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		log.WithError(err).Warn(ctx, "failed to clean-up pod")
		retry = errorsutil.IsTransientErr(ctx, err) || apierr.IsConflict(err)
	}
	return true
}

func (c *Controller) queuePodForCleanup(ctx context.Context, namespace string, podName string, action podCleanupAction) {
	c.log.WithFields(logging.Fields{"namespace": namespace, "podName": podName, "action": action}).Debug(ctx, "queueing pod for cleanup")
	// AddRateLimited reserves retry tokens before the queue deduplicates pod events.
	c.addPodCleanup(newPodCleanupKey(namespace, podName, action), 0)
}

func (c *Controller) queuePodForCleanupAfter(ctx context.Context, namespace string, podName string, action podCleanupAction, duration time.Duration) {
	logCtx := c.log.WithFields(logging.Fields{"namespace": namespace, "podName": podName, "action": action, "after": duration})
	if duration > 0 {
		logCtx.Debug(ctx, "queueing pod for cleanup after")
		c.addPodCleanup(newPodCleanupKey(namespace, podName, action), duration)
	} else {
		logCtx.Warn(ctx, "queueing pod for cleanup now, rather than delayed")
		c.addPodCleanup(newPodCleanupKey(namespace, podName, action), 0)
	}
}

func (c *Controller) queuePodForCleanupByUID(ctx context.Context, namespace string, podName string, uid string) {
	c.log.WithFields(logging.Fields{"namespace": namespace, "podName": podName, "uid": uid, "action": deletePodByUID}).Info(ctx, "queueing pod for cleanup by UID")
	c.addPodCleanup(newPodCleanupKeyWithUID(namespace, podName, deletePodByUID, uid), 0)
}
