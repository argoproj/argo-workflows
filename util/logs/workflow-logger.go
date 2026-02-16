package logs

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// The goal of this class is to stream the logs of the workflow you want.
// * If you request "follow" and the workflow is not completed: logs will be tailed until the workflow is completed or context done.
// * Otherwise, it will print recent logs and exit.

type request interface {
	GetNamespace() string
	GetName() string
	GetPodName() string
	GetLogOptions() *corev1.PodLogOptions
	GetGrep() string
	GetSelector() string
}

type sender interface {
	Send(entry *workflowpkg.LogEntry) error
}

const maxTokenLength = 1024 * 1024
const startBufSize = 16 * 1024

func scanLinesOrGiveLong(data []byte, atEOF bool) (advance int, token []byte, err error) {
	advance, token, err = bufio.ScanLines(data, atEOF)
	if advance > 0 || token != nil || err != nil {
		// bufio.ScanLines found something, use it
		return
	}

	// bufio.ScanLines found nothing
	// if our buffer is still a reasonable size, continue scanning for regular lines
	if len(data) < maxTokenLength {
		return
	}

	// our buffer is getting massive, stop waiting for line breaks and return data now
	// this avoids bufio.ErrTooLong
	return maxTokenLength, data[0:maxTokenLength], nil
}

func WorkflowLogs(ctx context.Context, wfClient versioned.Interface, kubeClient kubernetes.Interface, req request, sender sender) error {
	wfInterface := wfClient.ArgoprojV1alpha1().Workflows(req.GetNamespace())
	_, err := wfInterface.Get(ctx, req.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}

	rx, err := regexp.Compile(req.GetGrep())
	if err != nil {
		return fmt.Errorf("failed to compile %q: %w", req.GetGrep(), err)
	}

	podInterface := kubeClient.CoreV1().Pods(req.GetNamespace())

	ctx, logger := logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"workflow": req.GetName(), "namespace": req.GetNamespace()}).InContext(ctx)

	var podListOptions metav1.ListOptions

	// we add selector if cli specify the pod selector when using logs
	if req.GetSelector() != "" {
		podListOptions = metav1.ListOptions{LabelSelector: common.LabelKeyWorkflow + "=" + req.GetName() + "," + req.GetSelector()}
	} else {
		// we create a watch on the pods labelled with the workflow name,
		// but we also filter by pod name if that was requested
		podListOptions = metav1.ListOptions{LabelSelector: common.LabelKeyWorkflow + "=" + req.GetName()}
	}

	if req.GetPodName() != "" {
		podListOptions.FieldSelector = "metadata.name=" + req.GetPodName()
	}

	logger.WithField("options", podListOptions).Debug(ctx, "List options")

	// Keep a track of those we are logging, we also have a mutex to guard reads. Even if we stop streaming, we
	// keep a marker here so we don't start again.
	streamedPods := make(map[types.UID]bool)
	var streamedPodsGuard sync.Mutex
	var wg sync.WaitGroup
	// A non-blocking channel for log entries to go down.
	unsortedEntries := make(chan logEntry, 128)
	logOptions := req.GetLogOptions()
	if logOptions == nil {
		logOptions = &corev1.PodLogOptions{}
	}
	logger.WithField("options", logOptions).Debug(ctx, "Log options")

	// make a copy of requested log options and set timestamps to true, so they can be parsed out later
	podLogStreamOptions := *logOptions
	podLogStreamOptions.Timestamps = true

	// this func start a stream if one is not already running
	ensureWeAreStreaming := func(pod *corev1.Pod) {
		streamedPodsGuard.Lock()
		defer streamedPodsGuard.Unlock()
		ctx, logger := logger.WithField("podName", pod.GetName()).InContext(ctx)
		logger.WithFields(logging.Fields{"podPhase": pod.Status.Phase, "alreadyStreaming": streamedPods[pod.UID]}).Debug(ctx, "Ensuring pod logs stream")
		if pod.Status.Phase != corev1.PodPending && !streamedPods[pod.UID] {
			streamedPods[pod.UID] = true
			podName := pod.GetName()
			wg.Go(func() {
				logger.Debug(ctx, "Streaming pod logs")
				defer logger.Debug(ctx, "Pod logs stream done")
				stream, err := podInterface.GetLogs(podName, &podLogStreamOptions).Stream(ctx)
				if err != nil {
					logger.WithError(err).Error(ctx, "Failed to get pod logs")
					return
				}
				defer func() {
					if err := stream.Close(); err != nil {
						logger.WithError(err).Warn(ctx, "Failed to close stream")
					}
				}()
				scanner := bufio.NewScanner(stream)
				// give it more space for long line
				scanner.Buffer(make([]byte, startBufSize), maxTokenLength)
				// avoid bufio.ErrTooLong error when encounters a very very long line
				scanner.Split(scanLinesOrGiveLong)
				for scanner.Scan() {
					select {
					case <-ctx.Done():
						return
					default:
						line := scanner.Text()
						parts := strings.SplitN(line, " ", 2)
						// on old version k8s, the line may contains no space, hence len(parts) would equal to 1
						content := ""
						if len(parts) > 1 {
							content = parts[1]
						}
						timestamp, err := time.Parse(time.RFC3339, parts[0])
						if err != nil {
							logger.WithError(err).Error(ctx, "Failed to decode or infer timestamp from log line")
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
						if rx.MatchString(content) { // this means we filter the lines in the server, but will still incur the cost of retrieving them from Kubernetes
							logger.WithFields(logging.Fields{"timestamp": timestamp, "content": content}).Debug(ctx, "Log line")
							unsortedEntries <- logEntry{podName: podName, content: content, timestamp: timestamp}
						}
					}
				}
				logger.Debug(ctx, "No more log lines to stream")
				// out of data, we do not want to start watching again
			})
		}
	}

	podWatch, err := podInterface.Watch(ctx, podListOptions)
	if err != nil {
		return err
	}
	defer podWatch.Stop()

	// only list after we start the watch
	logger.Debug(ctx, "Listing workflow pods")
	list, err := podInterface.List(ctx, podListOptions)
	if err != nil {
		return err
	}

	// start watches by start-time
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[i].Status.StartTime.Before(list.Items[j].Status.StartTime)
	})

	for _, pod := range list.Items {
		ensureWeAreStreaming(&pod)
	}

	if logOptions.Follow {
		wfListOptions := metav1.ListOptions{FieldSelector: "metadata.name=" + req.GetName(), ResourceVersion: "0"}
		wfWatch, err := wfInterface.Watch(ctx, wfListOptions)
		if err != nil {
			return err
		}
		defer wfWatch.Stop()
		// We never send anything on this channel apart from closing it to indicate we should stop waiting for new pods.
		stopWatchingPods := make(chan struct{})
		// The purpose of this watch is to make sure we do not exit until the workflow is completed or deleted.
		// When that happens, it signals we are done by closing the stop channel.
		wg.Go(func() {
			defer close(stopWatchingPods)
			defer logger.Debug(ctx, "Done watching workflow events")
			logger.Debug(ctx, "Watching for workflow events")
			for {
				select {
				case <-ctx.Done():
					return
				case event, open := <-wfWatch.ResultChan():
					if !open {
						logger.Debug(ctx, "Re-establishing workflow watch")
						wfWatch.Stop()
						wfWatch, err = wfInterface.Watch(ctx, wfListOptions)
						if err != nil {
							logger.WithError(err).Error(ctx, "Failed to re-establish workflow watch")
							return
						}
						continue
					}
					wf, ok := event.Object.(*wfv1.Workflow)
					if !ok {
						// object is probably probably metav1.Status
						logger.WithError(apierr.FromObject(event.Object)).Warn(ctx, "watch object was not a workflow")
						return
					}
					logger.WithFields(logging.Fields{"eventType": event.Type, "completed": wf.Status.Fulfilled()}).Debug(ctx, "Workflow event")
					if event.Type == watch.Deleted || wf.Status.Fulfilled() {
						return
					}
				}
			}
		})

		// The purpose of this watch is to start streaming any new pods that appear when we are running.
		wg.Go(func() {
			defer logger.Debug(ctx, "Done watching pod events")
			logger.Debug(ctx, "Watching for pod events")
			for {
				select {
				case <-stopWatchingPods:
					return
				case event, open := <-podWatch.ResultChan():
					if !open {
						logger.Info(ctx, "Re-establishing pod watch")
						podWatch.Stop()
						podWatch, err = podInterface.Watch(ctx, podListOptions)
						if err != nil {
							logger.WithError(err).Error(ctx, "Failed to re-establish pod watch")
							return
						}
						continue
					}
					pod, ok := event.Object.(*corev1.Pod)
					if !ok {
						// object is probably probably metav1.Status
						logger.WithError(apierr.FromObject(event.Object)).Warn(ctx, "watch object was not a pod")
						return
					}
					logger.WithFields(logging.Fields{"eventType": event.Type, "podName": pod.GetName()}).Debug(ctx, "Pod event")
					if pod.Status.Phase != corev1.PodPending {
						ensureWeAreStreaming(pod)
					}
					podListOptions.ResourceVersion = pod.ResourceVersion
				}
			}
		})
	} else {
		logger.Debug(ctx, "Not starting watches")
	}

	doneSorting := make(chan struct{})
	go func() {
		defer close(doneSorting)
		defer logger.Debug(ctx, "Done sorting entries")
		logger.Debug(ctx, "Sorting entries")
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
				logger.WithFields(logging.Fields{"timestamp": e.timestamp, "content": e.content}).Debug(ctx, "Sending entry")
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
				logger.WithError(err).Error(ctx, "Failed to send final logs")
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
					logger.WithError(err).Error(ctx, "Failed to send logs")
					return
				}
			}
		}
	}()

	logger.Debug(ctx, "Waiting for work-group")
	wg.Wait()
	logger.Debug(ctx, "Work-group done")
	close(unsortedEntries)
	<-doneSorting
	logger.Debug(ctx, "Done-done")
	return nil
}
