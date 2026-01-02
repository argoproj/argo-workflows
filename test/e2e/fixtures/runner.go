package fixtures

import (
	"encoding/json"
	"testing"
	"time"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/server/utils"
	"github.com/argoproj/argo-workflows/v3/util/kubeconfig"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/secrets"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
)

type Runner struct {
	given       *Given
	persistence *Persistence
	restConfig  *rest.Config
}

func NewRunner(t *testing.T) *Runner {
	restConfig, err := kubeconfig.DefaultRestConfig()
	if err != nil {
		t.Fatal(err)
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatal(err)
	}
	configController := config.NewController(Namespace, common.ConfigMapName, kubeClient)
	ctx := logging.TestContext(t.Context())
	config, err := configController.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	persistence := NewPersistence(ctx, kubeClient, config)
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatal(err)
	}

	sec, err := clientset.CoreV1().Secrets(Namespace).Get(ctx, secrets.TokenName("argo-server"), metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	runner := &Runner{
		persistence: persistence,
		restConfig:  restConfig,
	}
	runner.given = NewGiven(
		t,
		versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().Workflows(Namespace),
		versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().WorkflowEventBindings(Namespace),
		versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().WorkflowTemplates(Namespace),
		versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().WorkflowTaskSets(Namespace),
		versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().ClusterWorkflowTemplates(),
		versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().CronWorkflows(Namespace),
		hydrator.New(persistence.OffloadNodeStatusRepo),
		kubeClient,
		string(sec.Data["token"]),
		restConfig,
		config)
	return runner
}

func (r *Runner) Given() *Given {
	return r.given
}

func (r *Runner) DeleteResources(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	l := func(r schema.GroupVersionResource) string {
		if r.Resource == "pods" {
			return common.LabelKeyWorkflow
		}
		return Label
	}

	pods := schema.GroupVersionResource{Version: "v1", Resource: "pods"}
	resources := []schema.GroupVersionResource{
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.CronWorkflowPlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.WorkflowPlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.WorkflowTemplatePlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.ClusterWorkflowTemplatePlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: workflow.WorkflowEventBindingPlural},
		{Group: workflow.Group, Version: workflow.Version, Resource: "sensors"},
		{Group: workflow.Group, Version: workflow.Version, Resource: "eventsources"},
		pods,
		{Version: "v1", Resource: "resourcequotas"},
		{Version: "v1", Resource: "configmaps"},
	}
	for _, resource := range resources {
		for {
			// remove finalizer from all the resources of the given GroupVersionResource
			resourceInf := DynamicFor(r.restConfig, pods)
			resourceList, err := resourceInf.List(ctx, metav1.ListOptions{LabelSelector: common.LabelKeyCompleted + "=false"})
			CheckError(t, err)
			for _, item := range resourceList.Items {
				patch, err := json.Marshal(map[string]interface{}{
					"metadata": map[string]interface{}{
						"finalizers": []string{},
					},
				})
				CheckError(t, err)
				_, err = resourceInf.Patch(ctx, item.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
				if err != nil && !apierr.IsNotFound(err) {
					CheckError(t, err)
				}
			}
			CheckError(t, DynamicFor(r.restConfig, resource).DeleteCollection(ctx, metav1.DeleteOptions{GracePeriodSeconds: ptr.To(int64(2))}, metav1.ListOptions{LabelSelector: l(resource)}))
			ls, err := DynamicFor(r.restConfig, resource).List(ctx, metav1.ListOptions{LabelSelector: l(resource)})
			CheckError(t, err)
			if len(ls.Items) == 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	// delete archived workflows from the archive
	if r.persistence.IsEnabled() {
		archive := r.persistence.WorkflowArchive
		parse, err := labels.ParseToRequirements(Label)
		CheckError(t, err)
		workflows, err := archive.ListWorkflows(ctx, utils.ListOptions{
			Namespace:         Namespace,
			LabelRequirements: parse,
		})
		CheckError(t, err)
		for _, w := range workflows {
			err := archive.DeleteWorkflow(ctx, string(w.UID))
			CheckError(t, err)
		}
		parse, err = labels.ParseToRequirements(Backfill)
		CheckError(t, err)
		backfillWorkflows, err := archive.ListWorkflows(ctx, utils.ListOptions{
			Namespace:         Namespace,
			LabelRequirements: parse,
		})
		CheckError(t, err)
		for _, w := range backfillWorkflows {
			err := archive.DeleteWorkflow(ctx, string(w.UID))
			CheckError(t, err)
		}
	}
}
