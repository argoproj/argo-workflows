package logs

import (
	"bufio"
	"context"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
)

// The goal of this class is to stream the logs of the workflow you want.
// * If you request "follow" and the workflow is not completed: logs will be tailed until the workflow is completed or context done.
// * Otherwise, it will print recent logs and exit
type WorkflowLogger interface {
	Run(ctx context.Context)
}

type request interface {
	GetNamespace() string
	GetName() string
	GetPodName() string
	GetLogOptions() *corev1.PodLogOptions
}

type sender interface {
	Send(entry *workflowpkg.LogEntry) error
}

type workflowLogger struct {
	logCtx               *log.Entry
	completed            bool
	follow               bool
	wg                   *sync.WaitGroup
	initialPods          []corev1.Pod
	ensureWeAreStreaming func(pod *corev1.Pod)
	podWatch             watch.Interface
	wfWatch              watch.Interface
	// We cannot be sure what order entries will come in, because we do not know how Goroutine scheduling will occur.
	// Rather than immediately send entries to the client, we send them them to this channel, and another Goroutine
	// uses a ticker to periodically (every 1s) sort the message and send them in order.
	// This channel is closed whenever there is no more work to be done.
	unsortedEntries chan logEntry
	sender          sender
}

func NewWorkflowLogger(ctx context.Context, wfClient versioned.Interface, kubeClient kubernetes.Interface, req request, sender sender) (WorkflowLogger, error) {

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.GetNamespace()).Get(req.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	completed := wf.Status.Completed()

	wfWatch, err := wfClient.ArgoprojV1alpha1().Workflows(req.GetNamespace()).Watch(metav1.ListOptions{FieldSelector: "metadata.name=" + req.GetName()})
	if err != nil {
		return nil, err
	}

	podInterface := kubeClient.CoreV1().Pods(req.GetNamespace())

	logCtx := log.WithFields(log.Fields{"workflow": req.GetName(), "namespace": req.GetNamespace()})

	// we create a watch on the pods labelled with the workflow name,
	// but we also filter by pod name if that was requested
	options := metav1.ListOptions{LabelSelector: common.LabelKeyWorkflow + "=" + req.GetName()}
	if req.GetPodName() != "" {
		options.FieldSelector = "metadata.name=" + req.GetPodName()
	}

	logCtx.WithField("options", options).Debug("List options")

	// Keep a track of those we are logging, we also have a mutex to guard reads. Even if we stop streaming, we
	// keep a marker here so we don't start again.
	streamedPods := make(map[types.UID]bool)
	var streamedPodsGuard sync.Mutex
	var wg sync.WaitGroup
	unsortedEntries := make(chan logEntry)
	logOptions := req.GetLogOptions().DeepCopy()
	logOptions.Timestamps = true

	// this func start a stream if one is not already running
	ensureWeAreStreaming := func(pod *corev1.Pod) {
		streamedPodsGuard.Lock()
		defer streamedPodsGuard.Unlock()
		logCtx := logCtx.WithField("podName", pod.GetName())
		logCtx.WithFields(log.Fields{"podPhase": pod.Status.Phase, "alreadyStreaming": streamedPods[pod.UID]}).Debug("Ensuring pod logs stream")
		if pod.Status.Phase != corev1.PodPending && !streamedPods[pod.UID] {
			streamedPods[pod.UID] = true
			wg.Add(1)
			go func(podName string) {
				defer wg.Done()
				logCtx.Debug("Streaming pod logs")
				defer logCtx.Debug("Pod logs stream done")
				stream, err := podInterface.GetLogs(podName, logOptions).Stream()
				if err != nil {
					logCtx.WithField("err", err).Error("Unable to get pod logs")
					return
				}
				scanner := bufio.NewScanner(stream)
				for scanner.Scan() {
					select {
					case <-ctx.Done():
						return
					default:
						line := scanner.Text()
						parts := strings.SplitN(line, " ", 2)
						timestamp, err := time.Parse(time.RFC3339, parts[0])
						if err != nil {
							logCtx.WithField("err", err).Error("Unable to get pod logs")
							return
						}
						content := parts[1]
						// You might ask - why don't we let the client do this? Well, it is because
						// this is the same as how this works for `kubectl logs`
						if req.GetLogOptions().Timestamps {
							content = line
						}
						logCtx.WithFields(log.Fields{"time": timestamp, "content": content}).Debug("Log line")
						unsortedEntries <- logEntry{PodName: podName, Content: content, timestamp: timestamp}
					}
				}
				logCtx.Debug("No more log lines to stream")
				// out of data, we do not want to start watching again
			}(pod.GetName())
		}
	}

	podWatch, err := podInterface.Watch(options)
	if err != nil {
		return nil, err
	}

	// only list after we start the watch
	logCtx.Debug("Listing workflow pods")
	list, err := podInterface.List(options)
	if err != nil {
		return nil, err
	}

	return &workflowLogger{
		logCtx:               logCtx,
		initialPods:          list.Items,
		completed:            completed,
		follow:               req.GetLogOptions().Follow,
		wg:                   &wg,
		ensureWeAreStreaming: ensureWeAreStreaming,
		wfWatch:              wfWatch,
		podWatch:             podWatch,
		unsortedEntries:      unsortedEntries,
		sender:               sender,
	}, nil
}

func (l *workflowLogger) sortAndSend() {
	defer l.logCtx.Debug("Done sorting entries")
	l.logCtx.Debug("Sorting entries")
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	entries := logEntries{}
	// Ugly to have this func, but we use it in two places (normal operation and finishing up).
	send := func() error {
		sort.Sort(entries)
		for len(entries) > 0 {
			// pop
			var e logEntry
			e, entries = entries[len(entries)-1], entries[:len(entries)-1]
			err := l.sender.Send(&workflowpkg.LogEntry{Content: e.Content, PodName: e.PodName})
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
			log.Error(err)
		}
	}()
	for {
		select {
		case <-ticker.C:
			err := send()
			if err != nil {
				log.Error(err)
				return
			}
		case entry, ok := <-l.unsortedEntries:
			if !ok {
				// The fact this channel is closed indicates that we do not need to do any more work.
				return
			} else {
				entries = append(entries, entry)
			}
		}
	}
}

func (l *workflowLogger) Run(ctx context.Context) {
	defer l.wfWatch.Stop()
	defer l.podWatch.Stop()

	l.logCtx.WithFields(log.Fields{"completed": l.completed, "follow": l.follow}).Debug("Running")

	// print logs by start time-ish
	sort.Slice(l.initialPods, func(i, j int) bool {
		return l.initialPods[i].Status.StartTime.Before(l.initialPods[j].Status.StartTime)
	})

	for _, pod := range l.initialPods {
		l.ensureWeAreStreaming(&pod)
	}

	if !l.completed && l.follow {
		// We never send anything on this channel apart from closing it to indicate we should stop waiting for new pods.
		stopCh := make(chan struct{})
		// The purpose of this watch is to make sure we do not exit until the workflow is completed or deleted.
		// When that happens, it signals we are done by closing the stop channel.
		l.wg.Add(1)
		go func() {
			defer close(stopCh)
			defer l.wg.Done()
			defer l.logCtx.Debug("Done watching workflow events")
			l.logCtx.Debug("Watching for workflow events")
			for {
				select {
				case <-ctx.Done():
					return
				case event := <-l.wfWatch.ResultChan():
					wf, ok := event.Object.(*wfv1.Workflow)
					if !ok {
						l.logCtx.Errorf("watch object was not a workflow %v", reflect.TypeOf(event.Object))
						return
					}
					l.logCtx.WithFields(log.Fields{"eventType": event.Type, "completed": wf.Status.Completed()}).Debug("Workflow event")
					if event.Type == watch.Deleted || wf.Status.Completed() {
						return
					}
				}
			}
		}()

		// The purpose of this watch is to start streaming any new pods that appear when we are running.
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			defer l.logCtx.Debug("Done watching pod events")
			l.logCtx.Debug("Watching for pod events")
			for {
				select {
				case <-stopCh:
					return
				case event := <-l.podWatch.ResultChan():
					pod, ok := event.Object.(*corev1.Pod)
					if !ok {
						l.logCtx.Errorf("watch object was not a pod %v", reflect.TypeOf(event.Object))
						return
					}
					l.logCtx.WithFields(log.Fields{"eventType": event.Type, "podName": pod.GetName(), "phase": pod.Status.Phase}).Debug("Pod event")
					if pod.Status.Phase == corev1.PodRunning {
						l.ensureWeAreStreaming(pod)
					}
				}
			}
		}()
	} else {
		l.logCtx.Debug("Not starting watches")
	}
	go l.sortAndSend()
	l.logCtx.Debug("Waiting for work-group")
	l.wg.Wait()
	l.logCtx.Debug("Work-group done")
	// Tell the sorting Goroutine it can finish.
	close(l.unsortedEntries)
}
