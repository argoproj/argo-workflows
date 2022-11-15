package ocm

import (
	"context"
	"fmt"
	"time"

	//v1 "open-cluster-management.io/api/work/v1"

	"k8s.io/client-go/rest"

	v1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"k8s.io/client-go/tools/cache"
	ocmworkclient "open-cluster-management.io/api/client/work/clientset/versioned"
	ocmworkinterface "open-cluster-management.io/api/client/work/clientset/versioned/typed/work/v1"
	ocmtypesv1 "open-cluster-management.io/api/work/v1"
)

type OCMProcessor struct {
	wfInformer             cache.SharedIndexInformer // this one gets passed in
	wfStatusInformer       cache.SharedIndexInformer // this one gets constructed locally
	manifestWorkerInformer cache.SharedIndexInformer // this one gets constructed locally
	// todo: which of these do we actually need?
	//kubeclient             dynamic.Interface
	restConfig *rest.Config
	//kubeclientset kubernetes.Interface
	wfclientset   wfclientset.Interface
	ocmworkclient ocmworkinterface.WorkV1Interface
}

var (
	workflowResultStatusResyncPeriod = 1 * time.Minute
)

func NewOCMProcessor(wfInformer cache.SharedIndexInformer, restConfig *rest.Config, wfclientset wfclientset.Interface) *OCMProcessor {
	ocm := &OCMProcessor{wfInformer: wfInformer,
		restConfig:  restConfig,
		wfclientset: wfclientset}

	ocmClient := ocmworkclient.NewForConfigOrDie(ocm.restConfig)
	ocm.ocmworkclient = ocmClient.WorkV1()

	// todo: construct wfStatusInformer and register processStatusUpdate() to be called when there's a Status Update

	return ocm
}

// process Workflow additions and updates
func (ocm *OCMProcessor) ProcessWorkflow(ctx context.Context, wf *wfv1.Workflow) error {

	fmt.Printf("deletethis: processing Workflow in OCM Processor: %+v\n", wf)

	// locate the label which indicates the cluster name (which is the namespace that our Manifest Work will go)
	mwNamespace, found := wf.Labels[common.LabelKeyCluster]
	if !found {
		return fmt.Errorf("In multicluster mode, the Workflow Controller requires all Workflows to contain label %s", mwNamespace)
	}

	// use the Workflow UUID to derive the ManifestWork name
	mwName := string(wf.UID)

	manifestWork := ocm.generateManifestWork(mwName, mwNamespace, wf)
	fmt.Printf("deletethis: generated Manifest Work in OCM Processor: %+v\n", manifestWork)

	// attempt to create ManifestWork with this name/namespace
	created, err := ocm.ocmworkclient.ManifestWorks(mwNamespace).Create(ctx, manifestWork, metav1.CreateOptions{}) //todo: do I need mwNamespace here?
	if err != nil {
		if apierr.IsAlreadyExists(err) {
		}
	}
	fmt.Printf("deletethis: result of generating Manifest Work in OCM Processor: %v, %v\n", created, err)

	return nil
}

func (ocm *OCMProcessor) ProcessWorkflowDeletion(ctx context.Context, wf *wfv1.Workflow) error {
	// locate the label which indicates the cluster name (namespace of ManifestWork)

	// use the Workflow UUID to derive the ManifestWork name

	// delete the ManifestWork

	return nil
}

// find Workflow associated with WorkflowStatusResult and update it
/*func (ocm *OCMProcessor) processStatusUpdate(ctx context.Context, wfStatus *wfv1.WorkflowStatusResult) error {

	return nil
}*/

// generateManifestWork creates the ManifestWork that wraps the Workflow as payload
// With the status sync feedback of Workflow's phase
func (ocm *OCMProcessor) generateManifestWork(name, namespace string, workflow *wfv1.Workflow) *ocmtypesv1.ManifestWork {
	return &ocmtypesv1.ManifestWork{ // TODO use OCM API helper to generate manifest work.
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			//Labels:    map[string]string{LabelKeyEnableOCMStatusSync: strconv.FormatBool(true)}, // todo: why is Mike using this?
			//Annotations: map[string]string{AnnotationKeyHubWorkflowNamespace: workflow.Namespace,
			//	AnnotationKeyHubWorkflowName: workflow.Name},
		},
		Spec: ocmtypesv1.ManifestWorkSpec{
			Workload: ocmtypesv1.ManifestsTemplate{
				Manifests: []ocmtypesv1.Manifest{{RawExtension: runtime.RawExtension{Object: workflow}}},
			},
			ManifestConfigs: []ocmtypesv1.ManifestConfigOption{
				{
					ResourceIdentifier: ocmtypesv1.ResourceIdentifier{
						Group:     v1alpha1.SchemeGroupVersion.Group,
						Resource:  "workflows", // TODO find the constant value from the argo API for this field
						Namespace: workflow.Namespace,
						Name:      workflow.Name,
					},
					FeedbackRules: []ocmtypesv1.FeedbackRule{
						{Type: ocmtypesv1.JSONPathsType, JsonPaths: []ocmtypesv1.JsonPath{{Name: "phase", Path: ".status.phase"}}},
					},
				},
			},
		},
	}
}

func (ocm *OCMProcessor) newWorkflowTaskSetInformer() wfextvv1alpha1.WorkflowStatusResultInformer {
	informer := externalversions.NewSharedInformerFactoryWithOptions(
		ocm.wfclientset,
		workflowResultStatusResyncPeriod).Argoproj().V1alpha1().WorkflowStatusResults()
	informer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {

				wfResult := obj.(*wfv1.WorkflowStatusResult)
				ocm.processStatusUpdate(context.Background(), wfResult)

				// if this is unstructured, we can imitate cron/controller.go
				/*un, ok := obj.(*unstructured.Unstructured)
				if !ok {
					logCtx.Errorf("malformed workflow status result: expected *unstructured.Unstructured, got %s", reflect.TypeOf(obj).Name())
					return true
				}

				wfStatusResult := &v1alpha1.WorkflowStatusResult{}
				err = util.FromUnstructuredObj(un, wfStatusResult)
				if err != nil {
					cc.eventRecorderManager.Get(un.GetNamespace()).Event(un, apiv1.EventTypeWarning, "Malformed", err.Error())
					logCtx.WithError(err).Error("malformed cron workflow: could not convert from unstructured")
					return true
				}*/
			},
		})
	return informer
}
