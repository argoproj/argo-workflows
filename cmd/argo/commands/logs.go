package commands

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	wfinformers "github.com/argoproj/argo/pkg/client/informers/externalversions"
	"github.com/argoproj/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
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
	if p.follow && wf.Status.Phase == v1alpha1.NodeRunning {
		p.printLiveWorkflowLogs(wf, timeByPod)
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
	err = p.getPodLogs("", podName, namespace, p.follow, p.tail, p.sinceSeconds, p.sinceTime, func(entry logEntry) {
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
			err := p.getPodLogs(getDisplayName(node), node.ID, wf.Namespace, false, p.tail, p.sinceSeconds, p.sinceTime, func(entry logEntry) {
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

func (p *logPrinter) setupWorkflowInformer(namespace string, name string, callback func(wf *v1alpha1.Workflow, done bool)) cache.SharedIndexInformer {
	wfcClientset := wfclientset.NewForConfigOrDie(restConfig)
	wfInformerFactory := wfinformers.NewFilteredSharedInformerFactory(wfcClientset, 20*time.Minute, namespace, nil)
	informer := wfInformerFactory.Argoproj().V1alpha1().Workflows().Informer()
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				updatedWf := new.(*v1alpha1.Workflow)
				if updatedWf.Name == name {
					callback(updatedWf, updatedWf.Status.Phase != v1alpha1.NodeRunning)
				}
			},
			DeleteFunc: func(obj interface{}) {
				deletedWf := obj.(*v1alpha1.Workflow)
				if deletedWf.Name == name {
					callback(deletedWf, true)
				}
			},
		},
	)
	return informer
}

// Prints live logs for workflow pods, starting from time specified in timeByPod name.
func (p *logPrinter) printLiveWorkflowLogs(workflow *v1alpha1.Workflow, timeByPod map[string]*time.Time) {
	logs := make(chan logEntry)
	streamedPods := make(map[string]bool)

	processPods := func(wf *v1alpha1.Workflow) {
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
					err := p.getPodLogs(getDisplayName(node), node.ID, wf.Namespace, true, nil, nil, sinceTimePtr, func(entry logEntry) {
						logs <- entry
					})
					if err != nil {
						log.Warn(err)
					}
				}()
			}
		}
	}

	processPods(workflow)
	informer := p.setupWorkflowInformer(workflow.Namespace, workflow.Name, func(wf *v1alpha1.Workflow, done bool) {
		if done {
			close(logs)
		} else {
			processPods(wf)
		}
	})

	stopChannel := make(chan struct{})
	go func() {
		informer.Run(stopChannel)
	}()
	defer close(stopChannel)

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

func (p *logPrinter) ensureContainerStarted(podName string, podNamespace string, container string, retryCnt int, retryTimeout time.Duration) error {
	for retryCnt > 0 {
		pod, err := p.kubeClient.CoreV1().Pods(podNamespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		var containerStatus *v1.ContainerStatus
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == container {
				containerStatus = &status
				break
			}
		}
		if containerStatus == nil || containerStatus.State.Waiting != nil {
			time.Sleep(retryTimeout)
			retryCnt--
		} else {
			return nil
		}
	}
	return fmt.Errorf("container '%s' of pod '%s' has not started within expected timeout", container, podName)
}

func (p *logPrinter) getPodLogs(
	displayName string, podName string, podNamespace string, follow bool, tail *int64, sinceSeconds *int64, sinceTime *metav1.Time, callback func(entry logEntry)) error {
	err := p.ensureContainerStarted(podName, podNamespace, p.container, 10, time.Second)
	if err != nil {
		return err
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
