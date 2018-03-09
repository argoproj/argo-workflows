package commands

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"hash/fnv"

	"math"

	"github.com/argoproj/argo-cd/errors"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type logEntry struct {
	source string
	pod    string
	time   time.Time
	line   string
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
		Short: "print the logs for a container in a workflow",
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

func (p *logPrinter) PrintWorkflowLogs(workflow string) error {
	wfClient := InitWorkflowClient()
	wf, err := wfClient.Get(workflow, metav1.GetOptions{})
	if err != nil {
		return err
	}
	timeByPod := p.printRecentWorkflowLogs(wf)
	if p.follow {
		p.printLiveWorkflowLogs(wf, timeByPod)
	}
	return nil
}

func (p *logPrinter) PrintPodLogs(podName string) error {
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}
	logs := p.getPodLogs("", podName, namespace, p.follow, p.tail, p.sinceSeconds, p.sinceTime)
	for entry := range logs {
		p.printLogEntry(entry)
	}
	return nil
}

func (p *logPrinter) printRecentWorkflowLogs(wf *v1alpha1.Workflow) map[string]*time.Time {
	var logs [][]logEntry
	for id, node := range wf.Status.Nodes {
		if node.Type == v1alpha1.NodeTypePod {
			log := p.getPodLogs(getSource(node), id, wf.Namespace, false, p.tail, p.sinceSeconds, p.sinceTime)
			var podLogs []logEntry
			for log := range log {
				podLogs = append(podLogs, log)
			}
			logs = append(logs, podLogs)
		}

	}
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

func (p *logPrinter) printLiveWorkflowLogs(wf *v1alpha1.Workflow, timeByPod map[string]*time.Time) {
	var logs []<-chan logEntry
	for id, node := range wf.Status.Nodes {
		if node.Phase == v1alpha1.NodeRunning && node.Type == v1alpha1.NodeTypePod {
			var sinceTimePtr *metav1.Time
			podTime := timeByPod[id]
			if podTime != nil {
				sinceTime := metav1.NewTime(podTime.Add(time.Second))
				sinceTimePtr = &sinceTime
			}
			log := p.getPodLogs(getSource(node), id, wf.Namespace, true, nil, nil, sinceTimePtr)
			logs = append(logs, log)
		}
	}
	mergedLogs := mergeChannels(logs)
	for entry := range mergedLogs {
		p.printLogEntry(entry)
	}
}

func getSource(node v1alpha1.NodeStatus) string {
	source := node.DisplayName
	if source == "" {
		source = node.Name
	}
	return source
}

func (p *logPrinter) printLogEntry(entry logEntry) {
	line := entry.line
	if p.timestamps {
		line = entry.time.Format(time.RFC3339) + "	" + line
	}
	if entry.source != "" {
		colors := []int{FgRed, FgGreen, FgYellow, FgBlue, FgMagenta, FgCyan, FgWhite, FgDefault}
		h := fnv.New32a()
		_, err := h.Write([]byte(entry.source))
		errors.CheckError(err)
		colorIndex := int(math.Mod(float64(h.Sum32()), float64(len(colors))))
		line = ansiFormat(entry.source, colors[colorIndex]) + ":	" + line
	}
	println(line)
}

func (p *logPrinter) getPodLogs(
	displayName string, podName string, podNamespace string, follow bool, tail *int64, sinceSeconds *int64, sinceTime *metav1.Time) <-chan logEntry {
	logs := make(chan logEntry)
	go func() {
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
					logs <- logEntry{
						pod:    podName,
						source: displayName,
						time:   logTime,
						line:   strings.Join(parts[1:], " "),
					}
				}
			}
			close(logs)
		}
	}()
	return logs
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

func mergeChannels(cs []<-chan logEntry) <-chan logEntry {
	out := make(chan logEntry)
	var wg sync.WaitGroup
	wg.Add(len(cs))
	for _, c := range cs {
		go func(c <-chan logEntry) {
			for v := range c {
				out <- v
			}
			wg.Done()
		}(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
