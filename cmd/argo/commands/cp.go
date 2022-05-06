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
	"strings"

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
	path         string // --path
}

func NewCpCommand() *cobra.Command {
	var cpArtifactOpts copyArtifactOpts

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

			ctx, apiClient := client.NewAPIClient(cmd.Context())
			serviceClient := apiClient.NewWorkflowServiceClient()
			namespace := client.Namespace()
			if len(cpArtifactOpts.namespace) > 0 {
				namespace = cpArtifactOpts.namespace
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
				ArtifactName: cpArtifactOpts.artifactName,
				TemplateName: cpArtifactOpts.templateName,
				NodeId:       cpArtifactOpts.nodeId,
			}
			artifactSearchResults := workflow.SearchArtifacts(&artifactSearchQuery)

			c := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: client.ArgoServerOpts.InsecureSkipVerify,
					},
				},
			}

			for _, artifact := range artifactSearchResults {
				customPath := filepath.Join(outputDir, cpArtifactOpts.path)
				nodeInfo := workflow.Status.Nodes.Find(func(n v1alpha1.NodeStatus) bool { return n.ID == artifact.NodeID })
				if nodeInfo != nil {
					customPath = strings.Replace(customPath, "{templateName}", nodeInfo.TemplateName, 1)
				}
				customPath = strings.Replace(customPath, "{namespace}", namespace, 1)
				customPath = strings.Replace(customPath, "{workflowName}", workflowName, 1)
				customPath = strings.Replace(customPath, "{nodeId}", artifact.NodeID, 1)
				customPath = strings.Replace(customPath, "{artifactName}", artifact.Name, 1)
				err = os.MkdirAll(customPath, os.ModePerm)
				if err != nil {
					return fmt.Errorf("failed to create folder path: %w", err)
				}
				key, err := artifact.GetKey()
				if err != nil {
					return fmt.Errorf("error getting key for artifact: %w", err)
				}
				err = getAndStoreArtifactData(namespace, workflowName, artifact.NodeID, artifact.Name, path.Base(key), customPath, c, client.ArgoServerOpts)
				if err != nil {
					return fmt.Errorf("failed to get and store artifact data: %w", err)
				}
			}
			return nil
		},
	}
	command.Flags().StringVarP(&cpArtifactOpts.namespace, "namespace", "n", "", "namespace of workflow")
	command.Flags().StringVar(&cpArtifactOpts.nodeId, "node-id", "", "id of node in workflow")
	command.Flags().StringVar(&cpArtifactOpts.templateName, "template-name", "", "name of template in workflow")
	command.Flags().StringVar(&cpArtifactOpts.artifactName, "artifact-name", "", "name of output artifact in workflow")
	command.Flags().StringVar(&cpArtifactOpts.path, "path", "{namespace}/{workflowName}/{nodeId}/outputs/{artifactName}", "use variables {workflowName}, {nodeId}, {templateName}, {artifactName}, and {namespace} to create a customized path to store the artifacts; example: {workflowName}/{templateName}/{artifactName}")
	return command
}

func getAndStoreArtifactData(namespace string, workflowName string, nodeId string, artifactName string, fileName string, customPath string, c *http.Client, argoServerOpts apiclient.ArgoServerOpts) error {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/artifacts/%s/%s/%s/%s", argoServerOpts.GetURL(), namespace, workflowName, nodeId, artifactName), nil)
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
	artifactFilePath := filepath.Join(customPath, fileName)
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
