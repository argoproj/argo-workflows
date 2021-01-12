package logs

import (
	"bufio"
	"context"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	retrywatch "k8s.io/client-go/tools/watch"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/hydrator"
	"github.com/argoproj/argo/workflow/util"
)

// The goal of this class is to stream the logs of the workflow you want.
// * If you request "follow" and the workflow is not completed: logs will be tailed until the workflow is completed or context done.
// * Otherwise, it will print recent logs and exit.

type request interface {
	GetNamespace() string
	GetName() string
	GetPodName() string
	GetLogOptions() *corev1.PodLogOptions
}

type sender interface {
	Send(entry *workflowpkg.LogEntry) error
}

func WorkflowLogs(ctx context.Context, thisClusterName wfv1.ClusterName, wfClient versioned.Interface, kubeClient map[wfv1.ClusterNamespaceKey]kubernetes.Interface, hydrator hydrator.Interface, req request, sender sender) error {
	wfInterface := wfClient.ArgoprojV1alpha1().Workflows(req.GetNamespace())
	wf, err := wfInterface.Get(ctx, req.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	logCtx := log.WithFields(log.Fields{"workflow": req.GetName(), "namespace": req.GetNamespace()})

	// make sure we don't start logging twice
	clusterNamespaces := sync.Map{} // map[wfv1.ClusterNamespaceKey]bool
	pods := sync.Map{}              // map[wfv1.ResourceKey]bool

	// wait for everything to finish
	var wg sync.WaitGroup

	// A non-blocking channel for log entries to go down.
	unsortedEntries := make(chan logEntry, 128)

	logOptions := req.GetLogOptions()
	if logOptions == nil {
		logOptions = &corev1.PodLogOptions{}
	}
	logCtx.WithField("options", logOptions).Debug("Log options")

	// make a copy of requested log options and set timestamps to true, so they can be parsed out later
	podLogStreamOptions := *logOptions
	podLogStreamOptions.Timestamps = true

	kube := func(clusterName wfv1.ClusterName, namespace string) kubernetes.Interface {
		if x, ok := kubeClient[wfv1.NewClusterNamespaceKey(clusterName, namespace)]; ok {
			return x
		}
		return kubeClient[wfv1.NewClusterNamespaceKey(clusterName, corev1.NamespaceAll)]
	}

	// this func start a stream if one is not already running
	logPod := func(clusterName wfv1.ClusterName, namespace, podName string) {
		podKey := wfv1.NewResourceKey(clusterName, common.PodGVR, namespace, podName)
		logCtx := log.WithField("podKey", podKey)
		_, alreadyLogging := pods.LoadOrStore(podKey, true)
		logCtx.WithField("alreadyLogging", alreadyLogging).Debug("logging pod")
		if alreadyLogging {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer logCtx.Debug("pod logging done")
			err := func() error {
				stream, err := kube(clusterName, namespace).CoreV1().Pods(namespace).GetLogs(podName, &podLogStreamOptions).Stream(ctx)
				if err != nil {
					return err
				}
				scanner := bufio.NewScanner(stream)
				for {
					select {
					case <-ctx.Done():
						return nil
					default:
						if !scanner.Scan() {
							return nil
						}
						line := scanner.Text()
						parts := strings.SplitN(line, " ", 2)
						content := parts[1]
						timestamp, err := time.Parse(time.RFC3339, parts[0])
						if err != nil {
							logCtx.Errorf("unable to decode or infer timestamp from log line: %s", err)
							// The current timestamp is the next best substitute. This won't be shown, but will be used
							// for sorting
							timestamp = time.Now()
							content = line
						}
						// You might ask - why don't we let the client do this? Well, it is because
						// this is the same as how this works for `kubectl logs`
						if req.GetLogOptions().Timestamps {
							content = line
						}
						logCtx.WithFields(log.Fields{"timestamp": timestamp, "content": content}).Debug("Log line")
						unsortedEntries <- logEntry{podName: podName, content: content, timestamp: timestamp}
					}
				}
			}()
			if err != nil {
				logCtx.WithError(err).Error("failed to stream pod")
			}
		}()
	}

	stopLoggingClusterNamespace := make(chan struct{})
	logClusterNamespace := func(clusterName wfv1.ClusterName, instanceID, namespace string) {
		clusterNamespaceKey := wfv1.NewClusterNamespaceKey(clusterName, namespace)
		logCtx := log.WithField("clusterNamespaceKey", clusterNamespaceKey)
		_, alreadyLogging := clusterNamespaces.LoadOrStore(clusterNamespaceKey, true)
		logCtx.WithField("alreadyLogging", alreadyLogging).Debug("logging cluster-namespace")
		if alreadyLogging {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer logCtx.Debug("cluster-namespace logging done")
			err := func() error {
				listOptions := metav1.ListOptions{
					LabelSelector: labels.NewSelector().
						Add(util.ClusterNameRequirement(clusterName, thisClusterName)).
						Add(util.InstanceIDRequirement(instanceID)).
						Add(util.WorkflowNameRequirement(req.GetName())).
						String(),
				}
				if req.GetPodName() != "" {
					listOptions.FieldSelector = "metadata.name=" + req.GetPodName()
				}
				list, err := kube(clusterName, namespace).CoreV1().Pods(namespace).List(ctx, listOptions)
				if err != nil {
					return err
				}
				// start watches by start-time
				sort.Slice(list.Items, func(i, j int) bool {
					return list.Items[i].Status.StartTime.Before(list.Items[j].Status.StartTime)
				})
				for _, pod := range list.Items {
					if pod.Status.Phase != corev1.PodPending {
						logPod(clusterName, pod.Namespace, pod.Name)
					}
				}
				// TODO - retry watcher? resource version too old?
				retryWatcher, err := retrywatch.NewRetryWatcher(list.ResourceVersion, &cache.ListWatch{
					WatchFunc: func(x metav1.ListOptions) (watch.Interface, error) {
						x.LabelSelector = listOptions.LabelSelector
						return kube(clusterName, namespace).CoreV1().Pods(namespace).Watch(ctx, x)
					},
				})
				if err != nil {
					return err
				}
				for {
					select {
					case <-stopLoggingClusterNamespace:
						return nil
					case <-ctx.Done():
						return nil
					case <-retryWatcher.Done():
						return nil
					case event := <-retryWatcher.ResultChan():
						pod, ok := event.Object.(*corev1.Pod)
						if !ok {
							return apierr.FromObject(event.Object)
						}
						if pod.Status.Phase != corev1.PodPending {
							logPod(thisClusterName, pod.Namespace, pod.Name)
						}
					}
				}
			}()
			if err != nil {
				logCtx.WithError(err).Error("failed to log cluster-namespace")
			}
		}()
	}

	logWorkflow := func(wf *wfv1.Workflow) error {
		err := hydrator.Hydrate(wf)
		if err != nil {
			return err
		}
		for clusterNamespace := range kubeClient {
			clusterName, namespace := clusterNamespace.Split()
			logClusterNamespace(wfv1.ClusterNameOr(clusterName, thisClusterName), wf.Labels[common.LabelKeyControllerInstanceID], wfv1.NamespaceOr(namespace, wf.Namespace))
		}
		return nil
	}

	// The purpose of this watch is to make sure we do not exit until the workflow is completed or deleted.
	// When that happens, it signals we are done by closing the stop channel.
	wg.Add(1)
	go func() {
		defer close(stopLoggingClusterNamespace)
		defer wg.Done()
		defer log.Debug("workflow watch done")
		err := func() error {
			err := logWorkflow(wf)
			if err != nil {
				return err
			}
			if !req.GetLogOptions().Follow {
				return nil
			}

			wfWatch, err := wfInterface.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + req.GetName()})
			if err != nil {
				return err
			}
			defer wfWatch.Stop()
			// The purpose of this watch is to make sure we do not exit until the workflow is completed or deleted.
			// When that happens, it signals we are done by closing the stop channel.
			if err != nil {
				return err
			}
			for {
				select {
				case <-ctx.Done():
					return nil
				case event, open := <-wfWatch.ResultChan():
					if !open {
						return io.EOF
					}
					wf, ok := event.Object.(*wfv1.Workflow)
					if !ok {
						return apierr.FromObject(event.Object)
					}
					logCtx.WithFields(log.Fields{"eventType": event.Type, "completed": wf.Status.Fulfilled()}).Debug("Workflow event")
					if event.Type == watch.Deleted || wf.Status.Fulfilled() {
						return nil
					}
					err := logWorkflow(wf)
					if err != nil {
						return err
					}
				}
			}
		}()
		if err != nil {
			logCtx.WithError(err).Error("failed to watch workflow")
		}
	}()

	doneSorting := make(chan struct{})
	go func() {
		defer close(doneSorting)
		defer logCtx.Debug("Done sorting entries")
		logCtx.Debug("Sorting entries")
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		entries := logEntries{}
		// Ugly to have this func, but we use it in two places (normal operation and finishing up).
		send := func() error {
			sort.Sort(entries)
			for len(entries) > 0 {
				// head
				var e logEntry
				e, entries = entries[0], entries[1:]
				logCtx.WithFields(log.Fields{"timestamp": e.timestamp, "content": e.content}).Debug("Sending entry")
				err := sender.Send(&workflowpkg.LogEntry{Content: e.content, PodName: e.podName})
				if err != nil {
					return err
				}
			}
			return nil
		}
		// This defer make sure we flush any remaining entries on exit.
		defer func() {
			err := send()
			if err != nil {
				logCtx.Error(err)
			}
		}()
		for {
			select {
			case entry, ok := <-unsortedEntries:
				if !ok {
					// The fact this channel is closed indicates that we need to finish-up.
					return
				} else {
					entries = append(entries, entry)
				}
			case <-ticker.C:
				err := send()
				if err != nil {
					logCtx.Error(err)
					return
				}
			}
		}
	}()

	logCtx.Debug("Waiting for work-group")
	wg.Wait()
	logCtx.Debug("Work-group done")
	close(unsortedEntries)
	<-doneSorting
	logCtx.Debug("Done-done")
	return nil
}
