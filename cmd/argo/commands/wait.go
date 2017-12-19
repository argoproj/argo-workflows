package commands

import (
	"fmt"
	"os"
	"sync"
	"time"

	wfclient "github.com/argoproj/argo/workflow/client"
	goversion "github.com/hashicorp/go-version"
	"github.com/jpillora/backoff"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(waitCmd)
	waitCmd.Flags().StringVarP(&waitArgs.namespace, "namespace", "n", "default", "Namespace in which to watch workflows")
}

type waitFlags struct {
	namespace string
}

var waitArgs waitFlags

var waitCmd = &cobra.Command{
	Use:   "wait WORKFLOW1 WORKFLOW2..,",
	Short: "waits for all workflows specified on command line to complete",
	Run:   WaitWorkflows,
}

// VersionChecker checks the Kubernetes version and currently logs a message if wait should
// be implemented using watch instead of polling.
type VersionChecker struct{}

func (vc *VersionChecker) run() {
	// Watch APIs on CRDs using fieldSelectors are only supported in Kubernetes v1.9.0 onwards.
	// https://github.com/kubernetes/kubernetes/issues/51046.
	versionInfo, err := clientset.ServerVersion()
	if err != nil {
		log.Fatalf("Failed to get Kubernetes version: %v", err)
	}

	serverVersion, err := goversion.NewVersion(versionInfo.String())
	if err != nil {
		log.Fatalf("Failed to create version: %v", err)
	}

	minVersion, err := goversion.NewVersion("1.9")
	if err != nil {
		log.Fatalf("Failed to create minimum version: %v", err)
	}

	if serverVersion.Equal(minVersion) || serverVersion.GreaterThan(minVersion) {
		fmt.Printf("This should be changed to use a \"watch\" based approach.\n")
	}
}

// WorkflowStatusPoller exports methods to wait on workflows by periodically
// querying their status.
type WorkflowStatusPoller struct{}

func (wsp *WorkflowStatusPoller) waitOnOne(wfc *wfclient.WorkflowClient, workflowName string, wg *sync.WaitGroup) bool {
	b := &backoff.Backoff{
		Min:    100 * time.Millisecond,
		Max:    5 * time.Minute,
		Factor: 2,
	}
	for {
		wf, err := wfc.GetWorkflow(workflowName)
		if err != nil {
			panic(err)
		}

		if !wf.Status.FinishedAt.IsZero() {
			fmt.Printf("%s completed at %v\n", workflowName, wf.Status.FinishedAt)
			wg.Done()
			return true
		}

		time.Sleep(b.Duration())
		continue
	}
}

// WaitWorkflows waits for the workflows passed in as args
func WaitWorkflows(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}

	wfc := InitWorkflowClient(waitArgs.namespace)

	// TODO(shri): When Kubernetes 1.9 support is added, this block should be executed
	// only for versions 1.8 and for 1.9, a new "watch" based implmentation should be
	// used.
	var vc VersionChecker
	vc.run()

	var wg sync.WaitGroup
	var wsp WorkflowStatusPoller
	for _, workflowName := range args {
		wg.Add(1)
		go wsp.waitOnOne(wfc, workflowName, &wg)
	}

	wg.Wait()

}
