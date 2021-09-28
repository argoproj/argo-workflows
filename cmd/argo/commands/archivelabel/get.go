package archivelabel

import (
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
)

func NewGetCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "get labelkey",
		Short: "get workflow label key=value in the archive",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			key := args[0]
			listOpts := &metav1.ListOptions{
				FieldSelector: "key=" + key,
			}

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient, err := apiClient.NewArchivedWorkflowServiceClient()
			errors.CheckError(err)
			labels, err := serviceClient.GetArchivedWorkflowLabel(ctx, &workflowarchivepkg.GetArchivedWorkflowLabelRequest{ListOptions: listOpts})
			errors.CheckError(err)

			for _, str := range labels.Items {
				fmt.Printf("%s\n", str)
			}
		},
	}
	return command
}
