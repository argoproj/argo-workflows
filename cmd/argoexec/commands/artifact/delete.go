package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	wfv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/util/retry"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	executor "github.com/argoproj/argo-workflows/v3/workflow/artifacts"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func NewArtifactDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "delete",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			namespace := client.Namespace()
			clientConfig := client.GetConfig()

			if podName, ok := os.LookupEnv(common.EnvVarArtifactGCPodHash); ok {

				config, err := clientConfig.ClientConfig()
				workflowInterface := workflow.NewForConfigOrDie(config)
				if err != nil {
					return err
				}

				artifactGCTaskInterface := workflowInterface.ArgoprojV1alpha1().WorkflowArtifactGCTasks(namespace)
				labelSelector := fmt.Sprintf("%s = %s", common.LabelKeyArtifactGCPodHash, podName)

				err = deleteArtifacts(labelSelector, cmd.Context(), artifactGCTaskInterface)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
}

type request struct {
	Task             *v1alpha1.WorkflowArtifactGCTask
	NodeName         string
	ArtifactNodeSpec *v1alpha1.ArtifactNodeSpec
}

type response struct {
	Task     *v1alpha1.WorkflowArtifactGCTask
	NodeName string
	Results  map[string]v1alpha1.ArtifactResult
	Err      error
}

func deleteArtifacts(labelSelector string, ctx context.Context, artifactGCTaskInterface wfv1alpha1.WorkflowArtifactGCTaskInterface) error {
	taskList, err := artifactGCTaskInterface.List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return err
	}
	taskWorkers := env.LookupEnvIntOr(common.EnvExecGCWorkers, 4)

	totalTasks := 0
	for _, task := range taskList.Items {
		totalTasks += len(task.Spec.ArtifactsByNode)
	}
	taskQueue := make(chan *request, totalTasks)
	responseQueue := make(chan response, totalTasks)
	for i := 0; i < taskWorkers; i++ {
		go deleteWorker(ctx, taskQueue, responseQueue)
	}

	taskno := 0
	nodesToGo := make(map[*v1alpha1.WorkflowArtifactGCTask]int)
	for _, task := range taskList.Items {
		nodesToGo[&task] = len(task.Spec.ArtifactsByNode)
		task.Status.ArtifactResultsByNode = make(map[string]v1alpha1.ArtifactResultNodeStatus)
		for nodeName, artifactNodeSpec := range task.Spec.ArtifactsByNode {
			taskQueue <- &request{Task: &task, NodeName: nodeName, ArtifactNodeSpec: &artifactNodeSpec}
			taskno++
		}
	}
	close(taskQueue)
	completed := 0
	for {
		response := <-responseQueue
		if response.Err != nil {
			return response.Err
		}
		if response.Task == nil {
			completed++
			if completed >= taskWorkers {
				break
			}
		} else {
			response.Task.Status.ArtifactResultsByNode[response.NodeName] = v1alpha1.ArtifactResultNodeStatus{ArtifactResults: response.Results}
			// Check for completed tasks
			nodesToGo[response.Task]--

			if nodesToGo[response.Task] == 0 {
				patch, err := json.Marshal(map[string]interface{}{"status": v1alpha1.ArtifactGCStatus{ArtifactResultsByNode: response.Task.Status.ArtifactResultsByNode}})
				if err != nil {
					return err
				}
				_, err = artifactGCTaskInterface.Patch(context.Background(), response.Task.Name, types.MergePatchType, patch, metav1.PatchOptions{}, "status")
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func deleteWorker(ctx context.Context, taskQueue chan *request, responseQueue chan response) {
	for {
		item, ok := <-taskQueue
		if !ok {
			// Done
			responseQueue <- response{Task: nil, NodeName: "", Err: nil}
			return
		}
		var archiveLocation *v1alpha1.ArtifactLocation
		results := make(map[string]v1alpha1.ArtifactResult)
		if item.ArtifactNodeSpec.ArchiveLocation != nil {
			archiveLocation = item.ArtifactNodeSpec.ArchiveLocation
		}

		var resources resources
		resources.Files = make(map[string][]byte) // same resources for every artifact
		for _, artifact := range item.ArtifactNodeSpec.Artifacts {
			if archiveLocation != nil {
				err := artifact.Relocate(archiveLocation)
				if err != nil {
					responseQueue <- response{Task: item.Task, NodeName: item.NodeName, Results: results, Err: err}
					continue
				}
			}
			drv, err := executor.NewDriver(ctx, &artifact, resources)
			if err != nil {
				responseQueue <- response{Task: item.Task, NodeName: item.NodeName, Results: results, Err: err}
				continue
			}

			err = waitutil.Backoff(retry.DefaultRetry, func() (bool, error) {
				err = drv.Delete(&artifact)
				if err != nil {
					errString := err.Error()
					results[artifact.Name] = v1alpha1.ArtifactResult{Name: artifact.Name, Success: false, Error: &errString}
					return false, err
				}
				results[artifact.Name] = v1alpha1.ArtifactResult{Name: artifact.Name, Success: true, Error: nil}
				return true, err
			})
		}
		responseQueue <- response{Task: item.Task, NodeName: item.NodeName, Results: results, Err: nil}
	}
}

type resources struct {
	Files map[string][]byte
}

func (r resources) GetSecret(ctx context.Context, name, key string) (string, error) {

	path := filepath.Join(common.SecretVolMountPath, name, key)
	if file, ok := r.Files[path]; ok {
		return string(file), nil
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return "", err
	} else {
		r.Files[path] = file
		return string(file), err
	}
}

func (r resources) GetConfigMapKey(ctx context.Context, name, key string) (string, error) {
	return "", fmt.Errorf("not supported")
}
