package commands

import (
	"log"
	"os"
	"time"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/common"
)

func NewLogsCommand() *cobra.Command {
	var (
		since     time.Duration
		sinceTime string
		tailLines int64
		grep      string
		selector  string
	)
	logOptions := &corev1.PodLogOptions{}
	command := &cobra.Command{
		Use:   "logs WORKFLOW [POD]",
		Short: "view logs of a pod or workflow",
		Example: `# Print the logs of a workflow:

  argo logs my-wf

# Follow the logs of a workflows:

  argo logs my-wf --follow

# Print the logs of a workflows with a selector:

  argo logs my-wf -l app=sth

# Print the logs of single container in a pod

  argo logs my-wf my-pod -c my-container

# Print the logs of a workflow's pods:

  argo logs my-wf my-pod

# Print the logs of a pods:

  argo logs --since=1h my-pod

# Print the logs of the latest workflow:
  argo logs @latest
`,
		Run: func(cmd *cobra.Command, args []string) {
			// parse all the args
			workflow := ""
			podName := ""

			switch len(args) {
			case 1:
				workflow = args[0]
			case 2:
				workflow = args[0]
				podName = args[1]
			default:
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			if since > 0 && sinceTime != "" {
				log.Fatal("--since-time and --since cannot be used together")
			}

			if since > 0 {
				logOptions.SinceSeconds = pointer.Int64Ptr(int64(since.Seconds()))
			}

			if sinceTime != "" {
				parsedTime, err := time.Parse(time.RFC3339, sinceTime)
				errors.CheckError(err)
				sinceTime := metav1.NewTime(parsedTime)
				logOptions.SinceTime = &sinceTime
			}

			if tailLines >= 0 {
				logOptions.TailLines = pointer.Int64Ptr(tailLines)
			}

			// set-up
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()

			common.LogWorkflow(ctx, serviceClient, namespace, workflow, podName, grep, selector, logOptions)
		},
	}
	command.Flags().StringVarP(&logOptions.Container, "container", "c", "main", "Print the logs of this container")
	command.Flags().BoolVarP(&logOptions.Follow, "follow", "f", false, "Specify if the logs should be streamed.")
	command.Flags().BoolVarP(&logOptions.Previous, "previous", "p", false, "Specify if the previously terminated container logs should be returned.")
	command.Flags().DurationVar(&since, "since", 0, "Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used.")
	command.Flags().StringVar(&sinceTime, "since-time", "", "Only return logs after a specific date (RFC3339). Defaults to all logs. Only one of since-time / since may be used.")
	command.Flags().Int64Var(&tailLines, "tail", -1, "If set, the number of lines from the end of the logs to show. If not specified, logs are shown from the creation of the container or sinceSeconds or sinceTime")
	command.Flags().StringVar(&grep, "grep", "", "grep for lines")
	command.Flags().StringVarP(&selector, "selector", "l", "", "log selector for some pod")
	command.Flags().BoolVar(&logOptions.Timestamps, "timestamps", false, "Include timestamps on each line in the log output")
	command.Flags().BoolVar(&common.NoColor, "no-color", false, "Disable colorized output")
	return command
}
