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

type copyArtifactOps struct {
	namespace    string // --namespace
	workflowName string // --workflow-name
	nodeId       string // --node-id
	templateName string // --template-name
	artifactName string // --artifact-name
}

func NewCpCommand() *cobra.Command {
	var copyArtifactArgs copyArtifactOps
	var argoServerArgs apiclient.ArgoServerOpts

	command := &cobra.Command{
		Use:   "cp outputDir ...",
		Short: "copy artifacts from workflow",
		Example: `# Copy a workflow's artifacts to a local output directory:

  argo cp output-directory --workflow-name=my-wf

# Copy artifacts from a specific node in a workflow to a local output directory:

  argo cp output-directory --workflow-name=my-wf --node-id=my-wf-node-id-123
`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			if len(copyArtifactArgs.namespace) > 0 {
				namespace = copyArtifactArgs.namespace
			}
			workflow, err := serviceClient.GetWorkflow(ctx, &workflowpkg.WorkflowGetRequest{
				Name:      copyArtifactArgs.workflowName,
				Namespace: namespace,
			})
			if err != nil {
				return fmt.Errorf("failed to get workflow: %w", err)
			}

			copyArtifactArgs.workflowName = workflow.Name
			artifactSearchQuery := v1alpha1.ArtifactSearchQuery{
				ArtifactName: copyArtifactArgs.artifactName,
				TemplateName: copyArtifactArgs.templateName,
				NodeId:       copyArtifactArgs.nodeId,
			}
			artifactSearchResults := workflow.SearchArtifacts(&artifactSearchQuery)

			outputDir := args[0]
			basePath := filepath.Join(outputDir, namespace, copyArtifactArgs.workflowName)
			err = os.MkdirAll(basePath, os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to create folder: %w", err)
			}

			c := makeClient(argoServerArgs)

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
				err = getAndStoreArtifactData(namespace, copyArtifactArgs.workflowName, artifact.NodeID, artifact.Name, path.Base(key), nodeIdFolder, c, argoServerArgs)
				if err != nil {
					return fmt.Errorf("failed to get and store artifact data: %w", err)
				}
			}
			return nil
		},
	}
	command.Flags().StringVarP(&copyArtifactArgs.namespace, "namespace", "n", "", "namespace of workflow")
	command.Flags().StringVar(&copyArtifactArgs.workflowName, "workflow-name", "", "name of workflow")
	command.Flags().StringVar(&copyArtifactArgs.nodeId, "node-id", "", "id of node in workflow")
	command.Flags().StringVar(&copyArtifactArgs.templateName, "template-name", "", "name of template in workflow")
	command.Flags().StringVar(&copyArtifactArgs.artifactName, "artifact-name", "", "name of output artifact in workflow")

	command.PersistentFlags().StringVarP(&argoServerArgs.URL, "argo-server", "s", os.Getenv("ARGO_SERVER"), "API server `host:port`. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.")
	command.PersistentFlags().StringVar(&argoServerArgs.Path, "argo-base-href", os.Getenv("ARGO_BASE_HREF"), "An path to use with HTTP client (e.g. due to BASE_HREF). Defaults to the ARGO_BASE_HREF environment variable.")
	command.PersistentFlags().BoolVarP(&argoServerArgs.InsecureSkipVerify, "insecure-skip-verify", "k", os.Getenv("ARGO_INSECURE_SKIP_VERIFY") == "true", "If true, the Argo Server's certificate will not be checked for validity. This will make your HTTPS connections insecure. Defaults to the ARGO_INSECURE_SKIP_VERIFY environment variable.")

	return command
}

func makeClient(argoServerArgs apiclient.ArgoServerOpts) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: argoServerArgs.InsecureSkipVerify,
			},
		},
	}
}

func getAndStoreArtifactData(namespace string, workflowName string, nodeId string, artifactName string, fileName string, nodeIdFolder string, c *http.Client, argoServerArgs apiclient.ArgoServerOpts) error {
	authString := ""
	if len(argoServerArgs.URL) == 0 {
		argoServerArgs.URL = "localhost:2746"
	} else {
		authString = client.GetAuthString()
	}
	request, err := http.NewRequest("GET", argoServerArgs.GetURL()+fmt.Sprintf("/artifacts/%s/%s/%s/%s", namespace, workflowName, nodeId, artifactName), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if len(authString) != 0 {
		request.Header.Set("Authorization", client.GetAuthString())
	}
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
	_, err = io.Copy(fileWriter, resp.Body)
	if err != nil {
		return fmt.Errorf("copying file contents failed: %w", err)
	}
	log.Printf("Stored artifact %s from node %s", fileName, nodeId)
	return nil
}
