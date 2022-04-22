package artifactgc

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/strings/slices"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	clienset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/util/slice"
	"github.com/argoproj/argo-workflows/v3/util/unstructured/workflow"
	"github.com/argoproj/argo-workflows/v3/workflow/artifactrepositories"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts"
	"github.com/argoproj/argo-workflows/v3/workflow/artifacts/resource"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

type Interface interface {
	Run(ctx context.Context)
}

type impl struct {
	kubernetesInterface  kubernetes.Interface
	workflowInterface    clienset.Interface
	wfInformer           cache.SharedIndexInformer
	artifactRepositories artifactrepositories.Interface
}

func New(kubernetesInterface kubernetes.Interface, workflowInterface clienset.Interface, wfInformer cache.SharedIndexInformer, artifactRepositories artifactrepositories.Interface) Interface {
	return &impl{kubernetesInterface, workflowInterface, wfInformer, artifactRepositories}
}

func (i *impl) Run(ctx context.Context) {
	log.Info("starting artifact garbage collector")
	i.wfInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			un := obj.(*unstructured.Unstructured)
			return slices.Contains(un.GetFinalizers(), Finalizer)
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				i.garbageCollect(ctx, obj)
			},
			UpdateFunc: func(_, obj interface{}) {
				i.garbageCollect(ctx, obj)
			},
			DeleteFunc: func(obj interface{}) {
				i.garbageCollect(ctx, obj)
			},
		},
	})
}

func (i *impl) garbageCollect(ctx context.Context, obj interface{}) {
	un := obj.(*unstructured.Unstructured)
	var strategies wfv1.ArtifactGCStrategies
	if workflow.GetPhase(un).Completed() {
		strategies = append(strategies, wfv1.ArtifactGCOnAWorkflowCompletion)
	}
	if un.GetDeletionTimestamp() != nil {
		strategies = append(strategies, wfv1.ArtifactGCOnWorkflowDeletion)
	}

	if err := i.deleteArtifacts(ctx, un, strategies); err != nil {
		log.WithError(err).Error("failed to delete artifacts")
		return
	}

	if un.GetDeletionTimestamp() == nil {
		return
	}

	data, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": slice.RemoveString(un.GetFinalizers(), Finalizer),
		},
	})

	_, err := i.workflowInterface.ArgoprojV1alpha1().Workflows(un.GetNamespace()).Patch(ctx, un.GetName(), types.MergePatchType, data, metav1.PatchOptions{})
	if err != nil && !apierr.IsNotFound(err) {
		log.WithError(err).Error("failed to remove artifact GC finalizer")
	}
}

func (i *impl) deleteArtifacts(ctx context.Context, un *unstructured.Unstructured, strategies wfv1.ArtifactGCStrategies) error {
	wf, _ := util.FromUnstructured(un)
	log.WithField("strategies", strategies).
		WithField("namespace", un.GetNamespace()).
		WithField("name", un.GetName()).
		Info("artifact garbage collection")
	for _, n := range wf.Status.Nodes {
		// wfv1.NodeStatus has the ArtifactGC field, but it is never set
		t := wf.GetTemplateByName(n.TemplateName)
		for _, a := range n.Outputs.GetArtifacts() {
			strategy := t.Outputs.GetArtifactByName(a.Name).GetArtifactGC().GetStrategy()
			if strategies.Contains(strategy) {
				if err := i.deleteArtifact(ctx, wf, a); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (i *impl) deleteArtifact(ctx context.Context, wf *wfv1.Workflow, a wfv1.Artifact) error {
	ar, err := i.artifactRepositories.Get(ctx, wf.Status.ArtifactRepositoryRef)
	if err != nil {
		return err
	}
	l := ar.ToArtifactLocation()
	if err := a.Relocate(l); err != nil {
		return err
	}
	key, _ := a.GetKey()
	log.WithField("name", a.Name).
		WithField("key", key).
		Info("deleting artifact")
	drv, err := artifacts.NewDriver(ctx, &a, resource.New(i.kubernetesInterface, wf.Namespace))
	if err != nil {
		return err
	}
	return drv.Delete(a)
}

const Finalizer = "workflows.argoproj.io/artifact-gc"
