package commands

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type copyArtifactOpts struct {
	namespace    string // --namespace
	nodeId       string // --node-id
	templateName string // --template-name
	artifactName string // --artifact-name
}

func NewCpCommand() *cobra.Command {
	var copyArtifactArgs copyArtifactOpts

	command := &cobra.Command{
		Use:   "cp my-wf output-directory ...",
		Short: "copy artifacts from workflow",
		Example: `# Copy a workflow's artifacts to a local output directory:

  argo cp my-wf output-directory

# Copy artifacts from a specific node in a workflow to a local output directory:

  argo cp my-wf output-directory --node-id=my-wf-node-id-123
`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				cmd.HelpFunc()(cmd, args)
				return fmt.Errorf("incorrect number of arguments")
			}
			workflowName := args[0]
			outputDir := args[1]

			argoServerUrl, err := cmd.Parent().Flags().GetString("argo-server")
			argoBasePath, err := cmd.Parent().Flags().GetString("argo-base-href")
			insecureSkipVerify, err := cmd.Parent().Flags().GetBool("insecure-skip-verify")
			if err != nil {
				return fmt.Errorf("not able to read flags correctly %w", err)
			}
			argoServerOpts := apiclient.ArgoServerOpts{
				URL:                argoServerUrl,
				Path:               argoBasePath,
				InsecureSkipVerify: insecureSkipVerify,
			}

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			if len(copyArtifactArgs.namespace) > 0 {
				namespace = copyArtifactArgs.namespace
			}
			workflow, err := serviceClient.GetWorkflow(ctx, &workflowpkg.WorkflowGetRequest{
				Name:      workflowName,
				Namespace: namespace,
			})
			if err != nil {
				return fmt.Errorf("failed to get workflow: %w", err)
			}

			workflowName = workflow.Name
			artifactSearchQuery := v1alpha1.ArtifactSearchQuery{
				ArtifactName: copyArtifactArgs.artifactName,
				TemplateName: copyArtifactArgs.templateName,
				NodeId:       copyArtifactArgs.nodeId,
			}
			artifactSearchResults := workflow.SearchArtifacts(&artifactSearchQuery)

			c := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: argoServerOpts.InsecureSkipVerify,
					},
				},
			}

			basePath := filepath.Join(outputDir, namespace, workflowName)
			for _, artifact := range artifactSearchResults {
				nodeIdFolder := filepath.Join(basePath, artifact.NodeID, "outputs")
				err = os.MkdirAll(nodeIdFolder, os.ModePerm)
				if err != nil {
					return fmt.Errorf("failed to create nodeId folder: %w", err)
				}
				key, err := artifact.GetKey()
				if err != nil {
					return fmt.Errorf("error getting key for artifact: %w", err)
				}
				err = getAndStoreArtifactData(namespace, workflowName, artifact.NodeID, artifact.Name, path.Base(key), nodeIdFolder, c, argoServerOpts)
				if err != nil {
					return fmt.Errorf("failed to get and store artifact data: %w", err)
				}
			}
			return nil
		},
	}
	command.Flags().StringVarP(&copyArtifactArgs.namespace, "namespace", "n", "", "namespace of workflow")
	command.Flags().StringVar(&copyArtifactArgs.nodeId, "node-id", "", "id of node in workflow")
	command.Flags().StringVar(&copyArtifactArgs.templateName, "template-name", "", "name of template in workflow")
	command.Flags().StringVar(&copyArtifactArgs.artifactName, "artifact-name", "", "name of output artifact in workflow")
	return command
}

func getAndStoreArtifactData(namespace string, workflowName string, nodeId string, artifactName string, fileName string, nodeIdFolder string, c *http.Client, argoServerArgs apiclient.ArgoServerOpts) error {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/artifacts/%s/%s/%s/%s", argoServerArgs.GetURL(), namespace, workflowName, nodeId, artifactName), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Authorization", client.GetAuthString())
	resp, err := c.Do(request)
	if err != nil {
		return fmt.Errorf("request failed with: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("request failed %s", resp.Status)
	}
	artifactFilePath := filepath.Join(nodeIdFolder, fileName)
	fileWriter, err := os.Create(artifactFilePath)
	if err != nil {
		return fmt.Errorf("creating file failed: %w", err)
	}
	defer fileWriter.Close()
	_, err = io.Copy(fileWriter, resp.Body)
	if err != nil {
		return fmt.Errorf("copying file contents failed: %w", err)
	}
	log.Printf("Stored artifact %s", fileName)
	return nil
}
