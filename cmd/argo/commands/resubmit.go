package commands

import (
	"log"
	"os"

	"github.com/argoproj/argo/workflow/common"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	RootCmd.AddCommand(resubmitCmd)
	resubmitCmd.Flags().BoolVar(&resubmitArgs.memoized, "memoized", false, "re-use successful steps & outputs from the previous run (experimental)")
}

var resubmitCmd = &cobra.Command{
	Use:   "resubmit WORKFLOW",
	Short: "resubmit a workflow",
	Run:   ResubmitWorkflow,
}

type resubmitFlags struct {
	memoized bool // --memoized
}

var resubmitArgs resubmitFlags

// ResubmitWorkflow resubmits a previous workflow
func ResubmitWorkflow(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}

	wfClient := InitWorkflowClient()
	wf, err := wfClient.Get(args[0], metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	newWF, err := common.FormulateResubmitWorkflow(wf, resubmitArgs.memoized)
	if err != nil {
		log.Fatal(err)
	}
	_, err = submitWorkflow(newWF)
	if err != nil {
		log.Fatal(err)
	}
}
