package commands

import (
	"os"
	"os/exec"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs CONTAINER",
	Short: "print the logs for a container in a workflow",
	Run:   getLogs,
}

type logsFlags struct {
	container  string // --container, -c
	follow     bool   // --follow, -f
	since      string // --since
	sinceTime  string // --since-time
	tail       int    // --tail
	timestamps bool   // --timestamps
}

var logsArgs logsFlags

func init() {
	logsCmd.Flags().StringVarP(&logsArgs.container, "container", "c", "main", "Print the logs of this container")
	logsCmd.Flags().BoolVarP(&logsArgs.follow, "follow", "f", false, "Specify if the logs should be streamed.")
	logsCmd.Flags().StringVar(&logsArgs.since, "since", "", "Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used.")
	logsCmd.Flags().StringVar(&logsArgs.sinceTime, "since-time", "", "Only return logs after a specific date (RFC3339). Defaults to all logs. Only one of since-time / since may be used.")
	logsCmd.Flags().IntVar(&logsArgs.tail, "tail", -1, "Lines of recent log file to display. Defaults to -1 with no selector, showing all log lines otherwise 10, if a selector is provided.")
	logsCmd.Flags().BoolVar(&logsArgs.timestamps, "timestamps", false, "Include timestamps on each line in the log output")
}

func getLogs(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
	argList := []string{"logs", args[0]}
	argList = append(argList, "-c", logsArgs.container)
	if logsArgs.follow {
		argList = append(argList, "-f")
	}
	if logsArgs.since != "" {
		argList = append(argList, "--since", logsArgs.since)
	}
	if logsArgs.sinceTime != "" {
		argList = append(argList, "--since-time", logsArgs.sinceTime)
	}
	if logsArgs.tail != -1 {
		argList = append(argList, "--tail", strconv.Itoa(logsArgs.tail))
	}
	if logsArgs.timestamps {
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
}
