package commands

import (
	"os"
	"os/exec"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewLogsCommand() *cobra.Command {
	var (
		container  string
		follow     bool
		since      string
		sinceTime  string
		tail       int
		timestamps bool
	)
	var command = &cobra.Command{
		Use:   "logs CONTAINER",
		Short: "print the logs for a container in a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			argList := []string{"logs", args[0]}
			argList = append(argList, "-c", container)
			if follow {
				argList = append(argList, "-f")
			}
			if since != "" {
				argList = append(argList, "--since", since)
			}
			if sinceTime != "" {
				argList = append(argList, "--since-time", sinceTime)
			}
			if tail != -1 {
				argList = append(argList, "--tail", strconv.Itoa(tail))
			}
			if timestamps {
				argList = append(argList, "--timestamps=true")
			}
			initKubeClient()
			namespace, _, err := clientConfig.Namespace()
			if err != nil {
				log.Fatal(err)
			}
			if namespace != "" {
				argList = append(argList, "--namespace", namespace)
			}

			execCmd := exec.Command("kubectl", argList...)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			err = execCmd.Run()
			if err != nil {
				log.Fatal(err)
			}
		},
	}
	command.Flags().StringVarP(&container, "container", "c", "main", "Print the logs of this container")
	command.Flags().BoolVarP(&follow, "follow", "f", false, "Specify if the logs should be streamed.")
	command.Flags().StringVar(&since, "since", "", "Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used.")
	command.Flags().StringVar(&sinceTime, "since-time", "", "Only return logs after a specific date (RFC3339). Defaults to all logs. Only one of since-time / since may be used.")
	command.Flags().IntVar(&tail, "tail", -1, "Lines of recent log file to display. Defaults to -1 with no selector, showing all log lines otherwise 10, if a selector is provided.")
	command.Flags().BoolVar(&timestamps, "timestamps", false, "Include timestamps on each line in the log output")
	return command
}
