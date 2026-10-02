// Package pod reconciles pods and takes care of gc events
package pod

import (
	"context"
	"errors"
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
	"k8s.io/client-go/rest"

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
		}
	}

	// if there was nothing to patch (no-op)
	if len(un.Object) == 0 {
		return nil, nil
	}
	// Guard the mutation itself, not just a subsequent DELETE. A resourceVersion
	// obtained from a replacement must never authorize an old Pod's cleanup.
	if pod.UID == "" || pod.ResourceVersion == "" {
		return nil, fmt.Errorf("cannot clean up pod without UID and resourceVersion")
	}
	un.SetUID(pod.UID)
	un.SetResourceVersion(pod.ResourceVersion)

	return un.MarshalJSON()
}

type containerSignalFunc func(context.Context, *rest.Config, *apiv1.Pod, string, syscall.Signal) error

// signalContainers signals all running containers, preserving delivery failures
// without allowing one failed or already-exited container to block the others.
func (c *Controller) signalContainers(ctx context.Context, pod *apiv1.Pod, sig syscall.Signal) (time.Duration, error) {
	signalContainer := c.signalContainer
	if signalContainer == nil {
		signalContainer = signal.Container
	}
	var failures []error
	for _, container := range pod.Status.ContainerStatuses {
		if container.State.Running == nil {
			continue
		}
		if err := signalContainer(ctx, c.restConfig, pod, container.Name, sig); err != nil {
			failures = append(failures, fmt.Errorf("signal %s to container %q: %w", sig, container.Name, err))
		}
	}
	if pod.Spec.TerminationGracePeriodSeconds == nil {
		return 30 * time.Second, errors.Join(failures...)
	}
	return time.Duration(*pod.Spec.TerminationGracePeriodSeconds) * time.Second, errors.Join(failures...)
}

func (c *Controller) patchPodForCleanup(ctx context.Context, pods typedv1.PodInterface, pod *apiv1.Pod, labelPodCompleted bool) error {
	patch, err := c.getPodCleanupPatch(pod, labelPodCompleted)
	if err != nil {
		return err
	}
	if patch == nil {
		return nil
	}

	_, err = pods.Patch(ctx, pod.Name, types.MergePatchType, patch, metav1.PatchOptions{})
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

	defer c.workqueue.Done(key)
	if strings.HasPrefix(key, workflowPodCleanupKeyPrefix) {
		c.processWorkflowPodCleanupIntent(ctx, key)
		return true
	}

	namespace, podName, action, uid := parsePodCleanupKey(key)
	ctx, log := c.log.WithFields(logging.Fields{"key": key, "action": action, "namespace": namespace, "podName": podName}).InContext(ctx)
	log.Debug(ctx, "cleaning up pod")
	if uid == "" {
		log.Warn(ctx, "ignoring cleanup request without identity")
		c.workqueue.Forget(key)
		return true
	}
	err := func() error {
		if action == reconcileWorkflowCleanup {
			return c.reconcileWorkflowCleanup(ctx, namespace, podName, uid)
		}
		pods := c.kubeclientset.CoreV1().Pods(namespace)
		pod, err := pods.Get(ctx, podName, metav1.GetOptions{})
		if apierr.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if string(pod.UID) != uid || pod.Namespace != namespace || !c.matchesInstance(pod) {
			log.WithField("currentUID", pod.UID).Debug(ctx, "ignoring cleanup for a replaced pod")
			return nil
		}
		if action == reconcilePodCleanup {
			return c.reconcilePodCleanup(ctx, pod)
		}
		switch action {
		case labelPodCompleted, deletePod, deletePodByUID, removeFinalizer:
			allowed, err := c.allowPodCleanup(ctx, pod)
			if err != nil || !allowed {
				return err
			}
		}
		switch action {
		case terminateContainers:
			if pod.Status.Phase == apiv1.PodPending {
				c.queuePodForCleanup(ctx, namespace, podName, deletePod, uid)
			} else {
				terminationGracePeriod, err := c.signalContainers(ctx, pod, syscall.SIGTERM)
				// Preserve escalation even if only some containers received
				// SIGTERM. Both failed actions retain their exact Pod UID and
				// retry from a fresh Pod read without needing another event.
				if terminationGracePeriod > 0 {
					c.queuePodForCleanupAfter(ctx, namespace, podName, killContainers, terminationGracePeriod, uid)
				}
				if err != nil {
					return err
				}
			}
		case killContainers:
			if _, err := c.signalContainers(ctx, pod, syscall.SIGKILL); err != nil {
				return err
			}
		case labelPodCompleted:
			if err := c.patchPodForCleanup(ctx, pods, pod, true); err != nil {
				return err
			}
		case deletePod, deletePodByUID:
			if err := c.patchPodForCleanup(ctx, pods, pod, false); err != nil {
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
			if err := c.patchPodForCleanup(ctx, pods, pod, false); err != nil {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		c.reportCleanupHold(ctx, key, log, err)
		if errorsutil.IsTransientErrQuiet(ctx, err) || apierr.IsConflict(err) {
			c.workqueue.AddRateLimited(key)
			return true
		}
		// Permission and other read/write failures can recover without a Pod
		// event. Keep the cleanup intent, with a bounded retry frequency.
		c.workqueue.AddAfter(key, podCleanupRetryDelay)
		return true
	}
	c.cleanupDiagnostics.forget(key)
	c.workqueue.Forget(key)
	return true
}

func (c *Controller) queuePodForCleanup(ctx context.Context, namespace string, podName string, action podCleanupAction, uid string) {
	if uid == "" {
		c.log.WithField("podName", podName).Warn(ctx, "cannot queue pod cleanup without UID")
		return
	}
	c.log.WithFields(logging.Fields{"namespace": namespace, "podName": podName, "action": action}).Debug(ctx, "queueing pod for cleanup")
	c.workqueue.AddRateLimited(newPodCleanupKeyWithUID(namespace, podName, action, uid))
}

func (c *Controller) queuePodForCleanupAfter(ctx context.Context, namespace string, podName string, action podCleanupAction, duration time.Duration, uid string) {
	if uid == "" {
		c.log.WithField("podName", podName).Warn(ctx, "cannot queue pod cleanup without UID")
		return
	}
	logCtx := c.log.WithFields(logging.Fields{"namespace": namespace, "podName": podName, "action": action, "after": duration})
	if duration > 0 {
		logCtx.Debug(ctx, "queueing pod for cleanup after")
		c.workqueue.AddAfter(newPodCleanupKeyWithUID(namespace, podName, action, uid), duration)
	} else {
		logCtx.Warn(ctx, "queueing pod for cleanup now, rather than delayed")
		c.workqueue.AddRateLimited(newPodCleanupKeyWithUID(namespace, podName, action, uid))
	}
}

func (c *Controller) queuePodForCleanupByUID(ctx context.Context, namespace string, podName string, uid string) {
	c.log.WithFields(logging.Fields{"namespace": namespace, "podName": podName, "uid": uid, "action": deletePodByUID}).Info(ctx, "queueing pod for cleanup by UID")
	c.queuePodForCleanup(ctx, namespace, podName, deletePodByUID, uid)
}
