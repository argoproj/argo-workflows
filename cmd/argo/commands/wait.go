package commands

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	goversion "github.com/hashicorp/go-version"
	"github.com/jpillora/backoff"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewWaitCommand() *cobra.Command {
	var (
		ignoreNotFound bool
	)
	var command = &cobra.Command{
		Use:   "wait WORKFLOW1 WORKFLOW2..,",
		Short: "waits for all workflows specified on command line to complete",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			wfc := InitWorkflowClient()
			wsp := NewWorkflowStatusPoller(wfc, ignoreNotFound, false)
			wsp.WaitWorkflows(args)
		},
	}
	command.Flags().BoolVar(&ignoreNotFound, "ignore-not-found", false, "Ignore the wait if the workflow is not found")
	return command
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
type WorkflowStatusPoller struct {
	wfc            v1alpha1.WorkflowInterface
	ignoreNotFound bool
	noOutput       bool
}

// NewWorkflowStatusPoller creates a new WorkflowStatusPoller object.
func NewWorkflowStatusPoller(wfc v1alpha1.WorkflowInterface, ignoreNotFound bool, noOutput bool) *WorkflowStatusPoller {
	return &WorkflowStatusPoller{wfc, ignoreNotFound, noOutput}
}

func (wsp *WorkflowStatusPoller) waitOnOne(workflowName string) {
	b := &backoff.Backoff{
		Min:    1 * time.Second,
		Max:    1 * time.Minute,
		Factor: 2,
	}
	for {
		wf, err := wsp.wfc.Get(workflowName, metav1.GetOptions{})
		if err != nil {
			if wsp.ignoreNotFound && apierr.IsNotFound(err) {
				if !wsp.noOutput {
					fmt.Printf("%s not found. Ignoring...\n", workflowName)
				}
				return
			}
			panic(err)
		}

		if !wf.Status.FinishedAt.IsZero() {
			if !wsp.noOutput {
				fmt.Printf("%s completed at %v\n", workflowName, wf.Status.FinishedAt)
			}
			return
		}

		time.Sleep(b.Duration())
		continue
	}
}

func (wsp *WorkflowStatusPoller) waitUpdateWaitGroup(workflowName string, wg *sync.WaitGroup) {
	defer wg.Done()
	wsp.waitOnOne(workflowName)
}

// WaitWorkflows waits for the given workflowNames.
func (wsp *WorkflowStatusPoller) WaitWorkflows(workflowNames []string) {
	// TODO(shri): When Kubernetes 1.9 support is added, this block should be executed
	// only for versions 1.8 and for 1.9, a new "watch" based implmentation should be
	// used.
	var vc VersionChecker
	vc.run()

	var wg sync.WaitGroup
	for _, workflowName := range workflowNames {
		wg.Add(1)
		go wsp.waitUpdateWaitGroup(workflowName, &wg)
	}
	wg.Wait()
}
