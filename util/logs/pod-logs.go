package logs

import (
	"bufio"
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/workflow/common"
)

func PodLogs(ctx context.Context, kubeClient kubernetes.Interface, req *workflowpkg.WorkflowLogRequest, ws workflowpkg.WorkflowService_PodLogsServer) error {
	podInterface := kubeClient.CoreV1().Pods(req.GetNamespace())

	logCtx := log.WithFields(log.Fields{"workflow": req.Name, "namespace": req.Namespace})

	// we create a watch on the pods labelled with the workflow name,
	// but we also filter by pod name if that was requested
	options := metav1.ListOptions{LabelSelector: common.LabelKeyWorkflow + "=" + req.Name}
	if req.PodName != "" {
		options.FieldSelector = "metadata.name=" + req.PodName
	}

	w, err := podInterface.Watch(options)
	if err != nil {
		return err
	}
	defer w.Stop()

	logCtx.Debug("Watching for pods")

	podsWeAreLogging := make(map[string]bool)

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
			logCtx := logCtx.WithField("podName", pod.GetName())
			logCtx.WithFields(log.Fields{"eventType": event.Type, "podPhase": pod.Status.Phase, "alreadyLogging": podsWeAreLogging[pod.Name]}).Debug("Event")
			// whenever a new pod appears, we start a goroutine to watch it
			if event.Type != watch.Deleted && pod.Status.Phase != corev1.PodPending && !podsWeAreLogging[pod.Name] {
				podsWeAreLogging[pod.Name] = true
				go func(podName string) {
					stream, err := podInterface.GetLogs(podName, req.LogOptions).Stream()
					if err != nil {
						podsWeAreLogging[pod.Name] = false
						logCtx.WithField("err", err).Error("Unable to get logs")
						return
					}
					scanner := bufio.NewScanner(stream)
					for scanner.Scan() {
						select {
						case <-ctx.Done():
							podsWeAreLogging[pod.Name] = false
							logCtx.Debug("Done")
							return
						default:
							content := scanner.Text()
							logCtx.WithField("content", content).Debug("Log line")
							// we actually don't know the container name AFAIK
							err = ws.Send(&workflowpkg.LogEntry{PodName: podName, Content: content})
							if err != nil {
								podsWeAreLogging[pod.Name] = false
								logCtx.WithField("err", err).Error("Unable to send log entry")
								return
							}
						}
					}
				}(pod.GetName())
			}
		}
	}
}
