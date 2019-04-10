package commands

import (
	"bufio"
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	pkgwatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/watch"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	workflowv1 "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/pkg/errors"
)

type logEntry struct {
	displayName string
	pod         string
	time        time.Time
	line        string
}

func NewLogsCommand() *cobra.Command {
	var (
		printer   logPrinter
		workflow  bool
		since     string
		sinceTime string
		tail      int64
	)
	var command = &cobra.Command{
		Use:   "logs POD/WORKFLOW",
		Short: "view logs of a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			conf, err := clientConfig.ClientConfig()
			errors.CheckError(err)
			printer.kubeClient = kubernetes.NewForConfigOrDie(conf)
			if tail > 0 {
				printer.tail = &tail
			}
			if sinceTime != "" {
				parsedTime, err := time.Parse(time.RFC3339, sinceTime)
				errors.CheckError(err)
				meta1Time := metav1.NewTime(parsedTime)
				printer.sinceTime = &meta1Time
			} else if since != "" {
				parsedSince, err := strconv.ParseInt(since, 10, 64)
				errors.CheckError(err)
				printer.sinceSeconds = &parsedSince
			}

			if workflow {
				err = printer.PrintWorkflowLogs(args[0])
				errors.CheckError(err)
			} else {
				err = printer.PrintPodLogs(args[0])
				errors.CheckError(err)
			}

		},
	}
	command.Flags().StringVarP(&printer.container, "container", "c", "main", "Print the logs of this container")
	command.Flags().BoolVarP(&workflow, "workflow", "w", false, "Specify that whole workflow logs should be printed")
	command.Flags().BoolVarP(&printer.follow, "follow", "f", false, "Specify if the logs should be streamed.")
	command.Flags().StringVar(&since, "since", "", "Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used.")
	command.Flags().StringVar(&sinceTime, "since-time", "", "Only return logs after a specific date (RFC3339). Defaults to all logs. Only one of since-time / since may be used.")
	command.Flags().Int64Var(&tail, "tail", -1, "Lines of recent log file to display. Defaults to -1 with no selector, showing all log lines otherwise 10, if a selector is provided.")
	command.Flags().BoolVar(&printer.timestamps, "timestamps", false, "Include timestamps on each line in the log output")
	return command
}

type logPrinter struct {
	container    string
	follow       bool
	sinceSeconds *int64
	sinceTime    *metav1.Time
	tail         *int64
	timestamps   bool
	kubeClient   kubernetes.Interface
}

// PrintWorkflowLogs prints logs for all workflow pods
func (p *logPrinter) PrintWorkflowLogs(workflow string) error {
	wfClient := InitWorkflowClient()
	wf, err := wfClient.Get(workflow, metav1.GetOptions{})
	if err != nil {
		return err
	}
	timeByPod := p.printRecentWorkflowLogs(wf)
	if p.follow {
		p.printLiveWorkflowLogs(wf.Name, wfClient, timeByPod)
	}
	return nil
}

// PrintPodLogs prints logs for a single pod
func (p *logPrinter) PrintPodLogs(podName string) error {
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}
	var logs []logEntry
	err = p.getPodLogs(context.Background(), "", podName, namespace, p.follow, p.tail, p.sinceSeconds, p.sinceTime, func(entry logEntry) {
		logs = append(logs, entry)
	})
	if err != nil {
		return err
	}
	for _, entry := range logs {
		p.printLogEntry(entry)
	}
	return nil
}

// Prints logs for workflow pod steps and return most recent log timestamp per pod name
func (p *logPrinter) printRecentWorkflowLogs(wf *v1alpha1.Workflow) map[string]*time.Time {
	var podNodes []v1alpha1.NodeStatus
	err := util.DecompressWorkflow(wf)
	if err != nil {
		log.Warn(err)
		return nil
	}
	for _, node := range wf.Status.Nodes {
		if node.Type == v1alpha1.NodeTypePod && node.Phase != v1alpha1.NodeError {
			podNodes = append(podNodes, node)
		}
	}
	var logs [][]logEntry
	var wg sync.WaitGroup
	wg.Add(len(podNodes))
	var mux sync.Mutex

	for i := range podNodes {
		node := podNodes[i]
		go func() {
			defer wg.Done()
			var podLogs []logEntry
			err := p.getPodLogs(context.Background(), getDisplayName(node), node.ID, wf.Namespace, false, p.tail, p.sinceSeconds, p.sinceTime, func(entry logEntry) {
				podLogs = append(podLogs, entry)
			})

			if err != nil {
				log.Warn(err)
				return
			}

			mux.Lock()
			logs = append(logs, podLogs)
			mux.Unlock()
		}()

	}
	wg.Wait()

	flattenLogs := mergeSorted(logs)

	if p.tail != nil {
		tail := *p.tail
		if int64(len(flattenLogs)) < tail {
			tail = int64(len(flattenLogs))
		}
		flattenLogs = flattenLogs[0:tail]
	}
	timeByPod := make(map[string]*time.Time)
	for _, entry := range flattenLogs {
		p.printLogEntry(entry)
		timeByPod[entry.pod] = &entry.time
	}
	return timeByPod
}

// Prints live logs for workflow pods, starting from time specified in timeByPod name.
func (p *logPrinter) printLiveWorkflowLogs(workflowName string, wfClient workflowv1.WorkflowInterface, timeByPod map[string]*time.Time) {
	logs := make(chan logEntry)
	streamedPods := make(map[string]bool)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processPods := func(wf *v1alpha1.Workflow) {
		err := util.DecompressWorkflow(wf)
		if err != nil {
			log.Warn(err)
			return
		}
		for id := range wf.Status.Nodes {
			node := wf.Status.Nodes[id]
			if node.Type == v1alpha1.NodeTypePod && node.Phase != v1alpha1.NodeError && streamedPods[node.ID] == false {
				streamedPods[node.ID] = true
				go func() {
					var sinceTimePtr *metav1.Time
					podTime := timeByPod[node.ID]
					if podTime != nil {
						sinceTime := metav1.NewTime(podTime.Add(time.Second))
						sinceTimePtr = &sinceTime
					}
					err := p.getPodLogs(ctx, getDisplayName(node), node.ID, wf.Namespace, true, nil, nil, sinceTimePtr, func(entry logEntry) {
						logs <- entry
					})
					if err != nil {
						log.Warn(err)
					}
				}()
			}
		}
	}

	go func() {
		defer close(logs)
		fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", workflowName))
		listOpts := metav1.ListOptions{FieldSelector: fieldSelector.String()}
		lw := &cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return wfClient.List(listOpts)
			},
			WatchFunc: func(options metav1.ListOptions) (pkgwatch.Interface, error) {
				return wfClient.Watch(listOpts)
			},
		}
		_, err := watch.UntilWithSync(ctx, lw, &v1alpha1.Workflow{}, nil, func(event pkgwatch.Event) (b bool, e error) {
			if wf, ok := event.Object.(*v1alpha1.Workflow); ok {
				if !wf.Status.Completed() {
					processPods(wf)
				}
				return wf.Status.Completed(), nil
			}
			return true, nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}()

	for entry := range logs {
		p.printLogEntry(entry)
	}
}

func getDisplayName(node v1alpha1.NodeStatus) string {
	res := node.DisplayName
	if res == "" {
		res = node.Name
	}
	return res
}

func (p *logPrinter) printLogEntry(entry logEntry) {
	line := entry.line
	if p.timestamps {
		line = entry.time.Format(time.RFC3339) + "	" + line
	}
	if entry.displayName != "" {
		colors := []int{FgRed, FgGreen, FgYellow, FgBlue, FgMagenta, FgCyan, FgWhite, FgDefault}
		h := fnv.New32a()
		_, err := h.Write([]byte(entry.displayName))
		errors.CheckError(err)
		colorIndex := int(math.Mod(float64(h.Sum32()), float64(len(colors))))
		line = ansiFormat(entry.displayName, colors[colorIndex]) + ":	" + line
	}
	fmt.Println(line)
}

func (p *logPrinter) hasContainerStarted(podName string, podNamespace string, container string) (bool, error) {
	pod, err := p.kubeClient.CoreV1().Pods(podNamespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	var containerStatus *v1.ContainerStatus
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == container {
			containerStatus = &status
			break
		}
	}
	if containerStatus == nil {
		return false, nil
	}

	if containerStatus.State.Waiting != nil {
		return false, nil
	}
	return true, nil
}

func (p *logPrinter) getPodLogs(
	ctx context.Context,
	displayName string,
	podName string,
	podNamespace string,
	follow bool,
	tail *int64,
	sinceSeconds *int64,
	sinceTime *metav1.Time,
	callback func(entry logEntry)) error {

	for ctx.Err() == nil {
		hasStarted, err := p.hasContainerStarted(podName, podNamespace, p.container)

		if err != nil {
			return err
		}
		if !hasStarted {
			if follow {
				time.Sleep(1 * time.Second)
			} else {
				return nil
			}
		} else {
			break
		}
	}

	stream, err := p.kubeClient.CoreV1().Pods(podNamespace).GetLogs(podName, &v1.PodLogOptions{
		Container:    p.container,
		Follow:       follow,
		Timestamps:   true,
		SinceSeconds: sinceSeconds,
		SinceTime:    sinceTime,
		TailLines:    tail,
	}).Stream()
	if err == nil {
		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, " ")
			logTime, err := time.Parse(time.RFC3339, parts[0])
			if err == nil {
				lines := strings.Join(parts[1:], " ")
				for _, line := range strings.Split(lines, "\r") {
					if line != "" {
						callback(logEntry{
							pod:         podName,
							displayName: displayName,
							time:        logTime,
							line:        line,
						})
					}
				}
			}
		}
	}
	return err
}

func mergeSorted(logs [][]logEntry) []logEntry {
	if len(logs) == 0 {
		return make([]logEntry, 0)
	}
	for len(logs) > 1 {
		left := logs[0]
		right := logs[1]
		size, i, j := len(left)+len(right), 0, 0
		merged := make([]logEntry, size, size)

		for k := 0; k < size; k++ {
			if i > len(left)-1 && j <= len(right)-1 {
				merged[k] = right[j]
				j++
			} else if j > len(right)-1 && i <= len(left)-1 {
				merged[k] = left[i]
				i++
			} else if left[i].time.Before(right[j].time) {
				merged[k] = left[i]
				i++
			} else {
				merged[k] = right[j]
				j++
			}
		}
		logs = append(logs[2:], merged)
	}
	return logs[0]
}
