package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
)

func NewTokenCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Print the auth token",
		Example: `
# Print the auth token for the current Argo Server
  argo auth token

# Save the token to a variable and send a request directly to the Argo API
  TOKEN=$(argo auth token) && curl -H "Authorization: Bearer $TOKEN" https://<ARGO_SERVER>/api/v1/userinfo
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			authString, err := client.GetAuthString(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Println(authString)
			return nil
		},
	}
}
