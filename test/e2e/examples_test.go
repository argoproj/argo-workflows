//go:build examples

package e2e

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/yaml"
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
	dyn := dynamic.NewForConfigOrDie(restConfig)

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

	kindToApply := map[string]bool{
		"ConfigMap":               true,
		"PersistentVolumeClaim":   true,
		"WorkflowTemplate":        true,
		"ClusterWorkflowTemplate": true,
	}

	err = fileutil.WalkManifests(ctx, "../../examples", func(path string, data []byte) error {
		docs := splitYAMLDocuments(data)
		for _, doc := range docs {
			if len(doc) == 0 {
				continue
			}
			obj := &unstructured.Unstructured{}
			err = yaml.Unmarshal(doc, &obj)
			if err != nil || obj == nil {
				continue
			}

			if _, ok := kindToApply[obj.GetKind()]; !ok {
				continue
			}

			gvr := obj.GroupVersionKind().GroupVersion().WithResource(strings.ToLower(obj.GetKind() + "s"))
			if obj.GetKind() == "ClusterWorkflowTemplate" {
				// cluster scoped resources don't need a namespace
				_, err = dyn.Resource(gvr).
					Apply(
						ctx,
						obj.GetName(),
						obj,
						metav1.ApplyOptions{
							FieldManager: "go-examples",
							Force:        true,
						})
			} else {
				_, err = dyn.Resource(gvr).
					Namespace(fixtures.Namespace).
					Apply(
						ctx,
						obj.GetName(),
						obj,
						metav1.ApplyOptions{
							FieldManager: "go-examples",
							Force:        true,
						})
			}
			if err != nil {
				if apierrors.IsConflict(err) {
					t.Logf("object %s/%s already exists or applied by another manager — skipping", obj.GetKind(), obj.GetName())
					continue
				}
				if apierrors.IsAlreadyExists(err) {
					t.Logf("object %s/%s exists — skipping", obj.GetKind(), obj.GetName())
					continue
				}
				t.Fatalf("apply error %s/%s: %v", obj.GetKind(), obj.GetName(), err)
			}
		}
		return nil
	})
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
				fixtures.NewGiven(
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
					config).
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

// helper: split multi-doc YAML
func splitYAMLDocuments(data []byte) [][]byte {
	sections := bytes.Split(data, []byte("---"))
	var docs [][]byte
	for _, s := range sections {
		trimmed := bytes.TrimSpace(s)
		if len(trimmed) > 0 {
			docs = append(docs, trimmed)
		}
	}
	return docs
}
