package commands

import (
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"strings"
	"time"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

func NewLogsCommand() *cobra.Command {
	var (
		since      time.Duration
		sinceTime  string
		tailLines  int64
		logOptions v1.PodLogOptions
	)
	var command = &cobra.Command{
		Use:   "logs POD|WORKFLOW",
		Short: "view logs of a pod or workflow",
		Example: `# Follow the logs of a workflow:

  argo logs my-wf

# Follow the logs of single container in a pod

  argo logs my-wf my-pod -c my-container

# Follow the logs of a workflow's pods:

  argo logs my-wf my-pod

# Follow the logs of a pods:

  argo logs --since=1h my-pod

`,
		Run: func(cmd *cobra.Command, args []string) {

			workflow := args[0]
			podName := ""

			switch len(args) {
			case 1:
				break
			case 2:
				podName = args[1]
				break
			default:
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			// parse all the args
			if since > 0 {
				seconds := int64(since.Seconds())
				logOptions.SinceSeconds = &seconds
			}

			if sinceTime != "" {
				parsedTime, err := time.Parse(time.RFC3339, sinceTime)
				errors.CheckError(err)
				sinceTime := metav1.NewTime(parsedTime)
				logOptions.SinceTime = &sinceTime
			}

			ctx, apiClient := client.NewAPIClient()
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			// color output on pod name
			colors := []int{FgRed, FgGreen, FgYellow, FgBlue, FgMagenta, FgCyan, FgWhite, FgDefault}
			h := fnv.New32a()
			_, err := h.Write([]byte(podName))
			errors.CheckError(err)
			colorIndex := int(math.Mod(float64(h.Sum32()), float64(len(colors))))

			// this outer loop allows us to retry when we can't find pods
			for {
				var logStream workflowpkg.WorkflowService_PodLogsClient
				// keep trying to get the pod logs
				for {
					logStream, err = serviceClient.PodLogs(ctx, &workflowpkg.WorkflowLogRequest{
						Name:       workflow,
						Namespace:  namespace,
						PodName:    podName,
						LogOptions: &logOptions,
					})
					if err != nil {
						_, _ = fmt.Fprintln(os.Stderr, err.Error())
						if strings.Contains(err.Error(), "ContainerCreating") {
							time.Sleep(3 * time.Second)
							continue // try again in 3s
						}
						return // give up
					}
					break // all good - lets start tailing
				}
				// loop on log lines
				for {
					event, err := logStream.Recv()
					if err != nil {
						_, _ = fmt.Fprintln(os.Stderr, err.Error())
						if strings.Contains(err.Error(), "waiting to start") {
							time.Sleep(3 * time.Second)
							break // break out of logging loop so we try to connect again in 3s
						}
						return // give up
					}
					fmt.Println(ansiFormat(fmt.Sprintf("%s %s", event.PodName, event.Content), colors[colorIndex]))
				}
			}
		},
	}
	command.Flags().StringVarP(&logOptions.Container, "container", "c", "main", "Print the logs of this container")
	command.Flags().BoolVarP(&logOptions.Follow, "follow", "f", false, "Specify if the logs should be streamed.")
	command.Flags().DurationVar(&since, "since", 0, "Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used.")
	command.Flags().StringVar(&sinceTime, "since-time", "", "Only return logs after a specific date (RFC3339). Defaults to all logs. Only one of since-time / since may be used.")
	command.Flags().Int64Var(&tailLines, "tail", -1, "Lines of recent log file to display. Defaults to -1 with no selector, showing all log lines otherwise 10, if a selector is provided.")
	command.Flags().BoolVar(&logOptions.Timestamps, "timestamps", false, "Include timestamps on each line in the log output")
	command.Flags().BoolVar(&noColor, "no-color", false, "Disable colorized output")
	return command
}
