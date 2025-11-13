//go:build examples

package e2e

import (
	"fmt"
	"testing"

	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	fileutil "github.com/argoproj/argo-workflows/v3/util/file"
	"github.com/argoproj/argo-workflows/v3/util/kubeconfig"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/secrets"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExampleWorkflows(t *testing.T) {
	restConfig, err := kubeconfig.DefaultRestConfig()
	if err != nil {
		t.Fatal(err)
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatal(err)
	}
	configController := config.NewController(fixtures.Namespace, common.ConfigMapName, kubeClient)
	ctx := logging.TestContext(t.Context())
	config, err := configController.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	persistence := fixtures.NewPersistence(ctx, kubeClient, config)

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatal(err)
	}
	sec, err := clientset.CoreV1().Secrets(fixtures.Namespace).Get(ctx, secrets.TokenName("argo-server"), metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	err = fileutil.WalkManifests(ctx, "../../examples", func(path string, data []byte) error {
		wfs, err := common.SplitWorkflowYAMLFile(ctx, data, true)
		if err != nil {
			t.Fatalf("Error parsing %s: %v", path, err)
		}
		for _, wf := range wfs {
			t.Run(path, func(t *testing.T) {
				t.Parallel()
				noTestKeyword, noTextLabelExists := wf.GetLabels()["workflows.argoproj.io/no-test"]
				if noTextLabelExists {
					t.Skip(fmt.Sprintf("Impossible to run this example: %s", noTestKeyword))
				}
				given := fixtures.NewGiven(
					t,
					versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().Workflows(fixtures.Namespace),
					versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().WorkflowEventBindings(fixtures.Namespace),
					versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().WorkflowTemplates(fixtures.Namespace),
					versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().WorkflowTaskSets(fixtures.Namespace),
					versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().ClusterWorkflowTemplates(),
					versioned.NewForConfigOrDie(restConfig).ArgoprojV1alpha1().CronWorkflows(fixtures.Namespace),
					hydrator.New(persistence.OffloadNodeStatusRepo),
					kubeClient,
					string(sec.Data["token"]),
					restConfig,
					config,
				)

				given.KubectlApply("../../examples/configmaps/simple-parameters-configmap.yaml", fixtures.NoError)
				given.
					ExampleWorkflow(&wf).
					When().
					SubmitWorkflow().
					WaitForWorkflow(fixtures.ToBeSucceeded)
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
