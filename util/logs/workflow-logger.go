package logs

import (
	"bufio"
	"context"
	"reflect"
	"sort"
	"sync"

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
	wg                   sync.WaitGroup
	initialPods          []corev1.Pod
	ensureWeAreStreaming func(pod *corev1.Pod)
	podWatch             watch.Interface
	wfWatch              watch.Interface
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

	// keep a track of those we are logging, we also have a mutex to guard reads
	streamedPods := make(map[types.UID]bool)
	var streamedPodsGuard sync.Mutex
	var wg sync.WaitGroup

	// this func start a stream if one is not already running
	ensureWeAreStreaming := func(pod *corev1.Pod) {
		wg.Add(1)
		defer wg.Done()
		streamedPodsGuard.Lock()
		defer streamedPodsGuard.Unlock()
		logCtx := logCtx.WithField("podName", pod.GetName())
		defer logCtx.Debug("Pod logs stream done")
		logCtx.WithFields(log.Fields{"podPhase": pod.Status.Phase, "alreadyStreaming": streamedPods[pod.UID]}).Debug("Ensuring pod logs stream")
		if pod.Status.Phase != corev1.PodPending && !streamedPods[pod.UID] {
			logCtx.WithField("logOptions", req.GetLogOptions()).Debug("Streaming pod logs")
			streamedPods[pod.UID] = true
			wg.Add(1)
			go func(podName string) {
				defer wg.Done()
				defer func() {
					streamedPodsGuard.Lock()
					defer streamedPodsGuard.Unlock()
					logCtx.Debug("Pod logs stream done")
				}()
				stream, err := podInterface.GetLogs(podName, req.GetLogOptions()).Stream()
				if err != nil {
					logCtx.WithField("err", err).Error("Unable to get pod logs")
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
				logCtx.Debug("No more log lines to stream")
				// out of data, we do not want to start watching again
			}(pod.GetName())
		}
	}

	logCtx.Debug("Listing workflow pods")
	list, err := podInterface.List(options)
	if err != nil {
		return nil, err
	}

	podWatch, err := podInterface.Watch(options)
	if err != nil {
		return nil, err
	}

	return &workflowLogger{
		logCtx:               logCtx,
		initialPods:          list.Items,
		completed:            completed,
		follow:               req.GetLogOptions().Follow,
		wg:                   wg,
		ensureWeAreStreaming: ensureWeAreStreaming,
		wfWatch:              wfWatch,
		podWatch:             podWatch,
	}, nil
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
		// the purpose of this watch is to make sure we do not exit until we are complete
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			defer l.logCtx.Debug("Workflow watch done")
			l.logCtx.Debug("Watching for workflow events")
			for {
				select {
				case <-ctx.Done():
					l.logCtx.Debug("Done")
					return
				case event := <-l.wfWatch.ResultChan():
					wf, ok := event.Object.(*wfv1.Workflow)
					if !ok {
						l.logCtx.Errorf("watch object was not a workflow %v", reflect.TypeOf(event.Object))
						return
					}
					l.logCtx.WithFields(log.Fields{"eventType": event.Type, "podName": wf.GetName(), "completed": wf.Status.Completed()}).Debug("Workflow event")
					// whenever a new pod appears, we start a goroutine to watch it
					if event.Type == watch.Deleted || wf.Status.Completed() {
						return // this will cause wg.Done and result in exit if all are done
					}
				}
			}
		}()

		// the purpose of this watch is to start streaming any new pods
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			defer l.logCtx.Debug("Pod watch done")
			l.logCtx.Debug("Watching for pod events")
			for {
				select {
				case <-ctx.Done():
					l.logCtx.Debug("Done")
					return
				case event := <-l.podWatch.ResultChan():
					pod, ok := event.Object.(*corev1.Pod)
					if !ok {
						l.logCtx.Errorf("watch object was not a pod %v", reflect.TypeOf(event.Object))
						return
					}
					l.logCtx.WithFields(log.Fields{"eventType": event.Type, "podName": pod.GetName(), "phase": pod.Status.Phase}).Debug("Pod event")
					switch pod.Status.Phase {
					case corev1.PodRunning:
						l.ensureWeAreStreaming(pod)
					case corev1.PodSucceeded, corev1.PodFailed:
						return
					}
				}
			}
		}()
	} else {
		l.logCtx.Debug("Not starting watches")
	}

	l.logCtx.Debug("Waiting for work-group")
	l.wg.Wait()
	l.logCtx.Debug("Wait for work-group done")
}
