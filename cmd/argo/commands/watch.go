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
	"github.com/argoproj/argo/workflow/util"
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
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case next := <-watchIf.ResultChan():
			wf, _ = next.Object.(*wfv1.Workflow)
		case <-ticker.C:
		}
		if wf == nil {
			watchIf.Stop()
			watchIf, err = wfClient.Watch(opts)
			errors.CheckError(err)
			continue
		}
		err := util.DecompressWorkflow(wf)
		errors.CheckError(err)
		print("\033[H\033[2J")
		print("\033[0;0H")
		printWorkflowHelper(wf, "")
		if !wf.Status.FinishedAt.IsZero() {
			break
		}
	}
	watchIf.Stop()
}
