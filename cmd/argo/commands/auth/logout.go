package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
)

func NewLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Delete the token saved by argo auth login",
		Long: `Delete the token that "argo auth login" saved for the server given by --argo-server, --argo-base-href and --secure.

The token stays valid on the Argo Server until it expires.`,
		Example: `# Delete the saved token
  argo auth logout --argo-server argo.example.com:443`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if client.ArgoServerOpts.URL == "" {
				return fmt.Errorf("argo auth logout needs the Argo Server address: use --argo-server or set ARGO_SERVER")
			}
			server := client.ServerKey()
			deleted, err := client.DeleteToken(server)
			if err != nil {
				return err
			}
			if !deleted {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "No saved token for %s\n", server)
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Deleted the saved token for %s\n", server)
			return err
		},
	}
}
