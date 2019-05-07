package commands

import (
	"fmt"
	"os"
	"sync"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

func NewWaitCommand() *cobra.Command {
	var (
		ignoreNotFound bool
	)
	var command = &cobra.Command{
		Use:   "wait WORKFLOW1 WORKFLOW2..,",
		Short: "waits for a workflow to complete",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			InitWorkflowClient()
			WaitWorkflows(args, ignoreNotFound, false)
		},
	}
	command.Flags().BoolVar(&ignoreNotFound, "ignore-not-found", false, "Ignore the wait if the workflow is not found")
	return command
}

// WaitWorkflows waits for the given workflowNames.
func WaitWorkflows(workflowNames []string, ignoreNotFound, quiet bool) {
	var wg sync.WaitGroup
	for _, workflowName := range workflowNames {
		wg.Add(1)
		go func(name string) {
			waitOnOne(name, ignoreNotFound, quiet)
			wg.Done()
		}(workflowName)
	}
	wg.Wait()
}

func waitOnOne(workflowName string, ignoreNotFound, quiet bool) {
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", workflowName))
	opts := metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	}

	_, err := wfClient.Get(workflowName, metav1.GetOptions{})
	if err != nil {
		if apierr.IsNotFound(err) && ignoreNotFound {
			return
		}
		errors.CheckError(err)
	}

	watchIf, err := wfClient.Watch(opts)
	errors.CheckError(err)
	defer watchIf.Stop()
	for {
		next := <-watchIf.ResultChan()
		wf, _ := next.Object.(*wfv1.Workflow)
		if wf == nil {
			watchIf.Stop()
			watchIf, err = wfClient.Watch(opts)
			errors.CheckError(err)
			continue
		}
		if !wf.Status.FinishedAt.IsZero() {
			if !quiet {
				fmt.Printf("%s completed at %v\n", workflowName, wf.Status.FinishedAt)
			}
			return
		}
	}
}
