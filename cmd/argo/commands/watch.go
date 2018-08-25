package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func NewWatchCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "watch WORKFLOW",
		Short: "watch a workflow until it completes",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			InitWorkflowClient()
			watchWorkflow(args[0])
		},
	}
	return command
}

func watchWorkflow(name string) {
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))
	opts := metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	}
	wf, err := wfClient.Get(name, metav1.GetOptions{})
	errors.CheckError(err)

	watchIf, err := wfClient.Watch(opts)
	errors.CheckError(err)
	defer watchIf.Stop()
	ticker := time.NewTicker(time.Second)

	var ok bool
	for {
		select {
		case next := <-watchIf.ResultChan():
			wf, ok = next.Object.(*wfv1.Workflow)
			if !ok {
				continue
			}
		case <-ticker.C:
		}
		print("\033[H\033[2J")
		print("\033[0;0H")
		printWorkflowHelper(wf, "")
		if !wf.Status.FinishedAt.IsZero() {
			return
		}
	}
}
