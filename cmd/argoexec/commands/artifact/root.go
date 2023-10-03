package artifact

import (
	"github.com/spf13/cobra"
)

func NewArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifact",
		Short: "Implement a new artifact",
		Example: `# Initialize a workflow for new artifact:

	argo artifact my-wf
			  
# Artifact multiple workflows:
			  
	argo artifact my-wf my-other-wf my-third-wf
			  
# Artifact multiple workflows by label selector:
			  
	argo artifact -l workflows.argoproj.io/test=true
			  
# Artifact multiple workflows by field selector:
			  
	argo artifact --field-selector metadata.namespace=argo
		  
# Artifact and wait for completion:
			  
	argo artifact --wait my-wf.yaml
			  
# Artifact and watch until completion:
			  
	argo artifact --watch my-wf.yaml
			  
# Artifact and tail logs until completion:
			  
	argo artifact --log my-wf.yaml
		  
# Artifact the latest workflow:
			  
	argo artifact @latest
`,
	}
	cmd.AddCommand(NewArtifactDeleteCommand())
	return cmd
}
