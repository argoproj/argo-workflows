package logs

import (
	"bufio"
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/workflow/common"
)

type Request interface {
	GetNamespace() string
	GetName() string
	GetPodName() string
	GetLogOptions() *corev1.PodLogOptions
}

type Sender interface {
	Send(entry *workflowpkg.LogEntry) error
}

// This function ensure you're logging the output from pods for the workflow.
// This includes existing pods, but it also watches for new pods.
// Notes:
// * It does not check to see if your workflow exists of if you have read permission on it (it does not need to).
// * It must assume you don't know if a pod might appear or not, so it intentionally  does not check
//   to see if the pod exists. This means it may wait forever if you have made a spelling mistake.
func PodLogs(ctx context.Context, kubeClient kubernetes.Interface, req Request, sender Sender) error {
	podInterface := kubeClient.CoreV1().Pods(req.GetNamespace())

	logCtx := log.WithFields(log.Fields{"workflow": req.GetName(), "namespace": req.GetNamespace()})

	// we create a watch on the pods labelled with the workflow name,
	// but we also filter by pod name if that was requested
	options := metav1.ListOptions{LabelSelector: common.LabelKeyWorkflow + "=" + req.GetName()}
	if req.GetPodName() != "" {
		options.FieldSelector = "metadata.name=" + req.GetPodName()
	}

	// keep a track of those we are logging
	streamedPods := make(map[types.UID]bool)
	var streamedPodsGuard sync.Mutex

	ensureWeAreStreaming := func(pod *corev1.Pod) {
		streamedPodsGuard.Lock()
		defer streamedPodsGuard.Unlock()
		logCtx.WithFields(log.Fields{"podPhase": pod.Status.Phase, "alreadyStreaming": streamedPods[pod.UID]}).Debug("Ensuring watch")
		if pod.Status.Phase != corev1.PodPending && !streamedPods[pod.UID] {
			streamedPods[pod.UID] = true
			go func(podName string) {
				defer func() {
					streamedPodsGuard.Lock()
					defer streamedPodsGuard.Unlock()
					streamedPods[pod.UID] = false
					logCtx.Debug("Stopping streaming")
				}()
				stream, err := podInterface.GetLogs(podName, req.GetLogOptions()).Stream()
				if err != nil {
					logCtx.WithField("err", err).Error("Unable to get logs")
					return
				}
				scanner := bufio.NewScanner(stream)
				for scanner.Scan() {
					select {
					case <-ctx.Done():
						logCtx.Debug("Done")
						return
					default:
						content := scanner.Text()
						logCtx.WithField("content", content).Debug("Log line")
						// we actually don't know the container name AFAIK
						err = sender.Send(&workflowpkg.LogEntry{PodName: podName, Content: content})
						if err != nil {
							logCtx.WithField("err", err).Error("Unable to send log entry")
							return
						}
					}
				}
				logCtx.Debug("No more data")
				// out of data, we do not want to start watching again
			}(pod.GetName())
		}
	}

	logCtx.Debug("Watching for known pods")
	list, err := podInterface.List(options)
	if err != nil {
		return err
	}

	w, err := podInterface.Watch(options)
	if err != nil {
		return err
	}
	defer w.Stop()

	// no errors are returned from this point forward as we will block using a select

	for _, pod := range list.Items {
		ensureWeAreStreaming(&pod)
	}

	logCtx.Debug("Watching for pod events")

	for {
		select {
		case <-ctx.Done():
			logCtx.Debug("Done")
			return ctx.Err()
		case event := <-w.ResultChan():
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				return fmt.Errorf("watch object was not a workflow %v", pod.GroupVersionKind())
			}
			logCtx.WithFields(log.Fields{"eventType": event.Type, "podName": pod.GetName()}).Debug("Event")
			// whenever a new pod appears, we start a goroutine to watch it
			if event.Type != watch.Deleted {
				ensureWeAreStreaming(pod)
			}
		}
	}
}
