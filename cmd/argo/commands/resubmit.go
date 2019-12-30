package commands

import (
	"log"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	apiUtil "github.com/argoproj/argo/util/api"
	"github.com/argoproj/argo/workflow/util"
)

func NewResubmitCommand() *cobra.Command {
	var (
		memoized      bool
		cliSubmitOpts cliSubmitOpts
	)
	var command = &cobra.Command{
		Use:   "resubmit WORKFLOW",
		Short: "resubmit a workflow",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}

			namespace, _, err := client.Config.Namespace()
			if err != nil {
				log.Fatal(err)
			}
			var created *v1alpha1.Workflow

			if client.ArgoServer != "" {
				conn := client.GetClientConn()
				defer conn.Close()
				apiGRPCClient, ctx := GetApiServerGRPCClient(conn)
				errors.CheckError(err)
				wfReq := workflow.WorkflowGetRequest{
					Namespace:    namespace,
					WorkflowName: args[0],
				}
				wf, err := apiGRPCClient.GetWorkflow(ctx, &wfReq)
				errors.CheckError(err)
				newWF, err := util.FormulateResubmitWorkflow(wf, memoized)
				errors.CheckError(err)
				newWF.Namespace = namespace
				created, err = apiUtil.SubmitWorkflowToAPIServer(apiGRPCClient, ctx, newWF, false)
				errors.CheckError(err)
			} else {
				wfClient := InitWorkflowClient()
				wf, err := wfClient.Get(args[0], metav1.GetOptions{})
				errors.CheckError(err)
				newWF, err := util.FormulateResubmitWorkflow(wf, memoized)
				errors.CheckError(err)
				created, err = util.SubmitWorkflow(wfClient, wfClientset, namespace, newWF, &util.SubmitOpts{})
				errors.CheckError(err)
			}
			printWorkflow(created, cliSubmitOpts.output, DefaultStatus)
			waitOrWatch([]string{created.Name}, cliSubmitOpts)
		},
	}

	command.Flags().StringVarP(&cliSubmitOpts.output, "output", "o", "", "Output format. One of: name|json|yaml|wide")
	command.Flags().BoolVarP(&cliSubmitOpts.wait, "wait", "w", false, "wait for the workflow to complete")
	command.Flags().BoolVar(&cliSubmitOpts.watch, "watch", false, "watch the workflow until it completes")
	command.Flags().BoolVar(&memoized, "memoized", false, "re-use successful steps & outputs from the previous run (experimental)")
	return command
}
