package logs

import (
	"bufio"
	"context"
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

func WorkflowLogs(ctx context.Context, wfClient versioned.Interface, kubeClient map[wfv1.ClusterName]kubernetes.Interface, hydrator hydrator.Interface, req request, sender sender) error {
	wfInterface := wfClient.ArgoprojV1alpha1().Workflows(req.GetNamespace())
	_, err := wfInterface.Get(req.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	logCtx := log.WithFields(log.Fields{"workflow": req.GetName(), "namespace": req.GetNamespace()})

	// Keep a track of those we are logging, we also have a mutex to guard reads. Even if we stop streaming, we
	// keep a marker here so we don't start again.
	clusterNamespaces := make(map[wfv1.ClusterNamespaceKey]bool)
	pods := make(map[wfv1.PodKey]bool)
	var mu sync.Mutex
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

	// this func start a stream if one is not already running
	logPod := func(pod *corev1.Pod) {
		wg.Add(1)
		defer wg.Done()
		clusterName := wfv1.ClusterNameOrDefault(pod.Labels[common.LabelKeyClusterName])
		podKey := wfv1.NewPodKey(clusterName, pod.Namespace, pod.Name)
		logCtx := log.WithField("podKey", podKey)
		logCtx.Debug()
		if pod.Status.Phase == corev1.PodPending {
			logCtx.Debug("pod pending")
			return
		}
		// return if already streaming
		if !func() bool {
			mu.Lock()
			defer mu.Unlock()
			if pods[podKey] {
				return false
			}
			pods[podKey] = true
			return true
		}() {
			logCtx.Debug("pod already streaming")
			return
		}
		err := func() error {
			stream, err := kubeClient[clusterName].CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogStreamOptions).Stream()
			if err != nil {
				return err
			}
			scanner := bufio.NewScanner(stream)
			for scanner.Scan() {
				select {
				case <-ctx.Done():
					return nil
				default:
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
					unsortedEntries <- logEntry{podName: pod.Name, content: content, timestamp: timestamp}
				}
			}
			return nil
		}()
		if err != nil {
			logCtx.WithError(err).Error("failed to stream pod")
		}
	}

	// We never send anything on this channel apart from closing it to indicate we should stop waiting for new pods.
	logClusterNamespace := func(clusterName wfv1.ClusterName, instanceID, namespace string) {
		wg.Add(1)
		defer wg.Done()
		clusterNamespaceKey := wfv1.NewClusterNamespaceKey(clusterName, namespace)
		logCtx := log.WithField("clusterNamespaceKey", clusterNamespaceKey)
		logCtx.Debug()
		if !func() bool {
			mu.Lock()
			defer mu.Unlock()
			if clusterNamespaces[clusterNamespaceKey] {
				return false
			}
			clusterNamespaces[clusterNamespaceKey] = true
			return true
		}() {
			logCtx.Debug("cluster-namespace already streaming")
			return
		}
		err := func() error {
			listOptions := metav1.ListOptions{
				LabelSelector: labels.NewSelector().
					Add(util.ClusterNameRequirement(clusterName)).
					Add(util.InstanceIDRequirement(instanceID)).
					Add(util.WorkflowNameRequirement(req.GetName())).
					String(),
			}
			if req.GetPodName() != "" {
				listOptions.FieldSelector = "metadata.name=" + req.GetPodName()
			}
			list, err := kubeClient[clusterName].CoreV1().Pods(namespace).List(listOptions)
			if err != nil {
				return err
			}
			// start watches by start-time
			sort.Slice(list.Items, func(i, j int) bool {
				return list.Items[i].Status.StartTime.Before(list.Items[j].Status.StartTime)
			})
			for _, pod := range list.Items {
				go logPod(&pod)
			}
			retryWatcher, err := retrywatch.NewRetryWatcher(list.ResourceVersion, &cache.ListWatch{
				WatchFunc: func(x metav1.ListOptions) (watch.Interface, error) {
					x.LabelSelector = listOptions.LabelSelector
					return kubeClient[clusterName].CoreV1().Pods(namespace).Watch(x)
				},
			})
			if err != nil {
				return err
			}
			for event := range retryWatcher.ResultChan() {
				select {
				case <-ctx.Done():
					return nil
				default:
					pod, ok := event.Object.(*corev1.Pod)
					if !ok {
						return apierr.FromObject(event.Object)
					}
					logCtx.WithFields(log.Fields{"eventType": event.Type, "podName": pod.GetName(), "phase": pod.Status.Phase}).Debug("Pod event")
					if pod.Status.Phase == corev1.PodRunning {
						logPod(pod)
					}
				}
			}
			return nil
		}()
		if err != nil {
			logCtx.WithError(err).Error("failed to log cluster-namespace")
		}
	}

	logWorkflow := func(wf *wfv1.Workflow) error {
		err := hydrator.Hydrate(wf)
		if err != nil {
			return err
		}
		for clusterName, namespaces := range wf.Status.Nodes.GetClusterNamespaces() {
			for namespace := range namespaces {
				go logClusterNamespace(wfv1.ClusterNameOrDefault(clusterName), wf.Labels[common.LabelKeyControllerInstanceID], namespaceOr(namespace, wf.Namespace))
			}
		}
		return nil
	}

	// The purpose of this watch is to make sure we do not exit until the workflow is completed or deleted.
	// When that happens, it signals we are done by closing the stop channel.
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := func() error {
			wf, err := wfInterface.Get(req.GetName(), metav1.GetOptions{})
			if err != nil {
				return err
			}
			err = logWorkflow(wf)
			if err != nil {
				return err
			}
			if !req.GetLogOptions().Follow {
				return nil
			}
			retryWatcher, err := retrywatch.NewRetryWatcher(wf.ResourceVersion, &cache.ListWatch{
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					options.FieldSelector = "metadata.name=" + req.GetName()
					return kubeClient[wfv1.DefaultClusterName].CoreV1().Pods(req.GetNamespace()).Watch(options)
				},
			})
			if err != nil {
				return err
			}
			defer retryWatcher.Stop()
			for event := range retryWatcher.ResultChan() {
				select {
				case <-ctx.Done():
					return nil
				default:
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
			return nil
		}()
		if err != nil {
			logCtx.WithError(err).Error("failed to watch workflow")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
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
	return nil
}

func namespaceOr(namespace string, otherNamespace string) string {
	if namespace != "" {
		return namespace
	}
	return otherNamespace
}
