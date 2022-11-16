package ocm

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

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
	wfInformer       cache.SharedIndexInformer                   // this one gets passed in
	wfStatusInformer wfextvv1alpha1.WorkflowStatusResultInformer // this one gets constructed locally
	restConfig       *rest.Config
	wfclientset      wfclientset.Interface
	ocmworkclient    ocmworkinterface.WorkV1Interface
}

var (
	workflowResultStatusResyncPeriod = 1 * time.Minute
)

func NewOCMProcessor(ctx context.Context, wfInformer cache.SharedIndexInformer, restConfig *rest.Config, wfclientset wfclientset.Interface) *OCMProcessor {
	ocm := &OCMProcessor{wfInformer: wfInformer,
		restConfig:  restConfig,
		wfclientset: wfclientset}

	ocmClient := ocmworkclient.NewForConfigOrDie(ocm.restConfig)
	ocm.ocmworkclient = ocmClient.WorkV1()

	// construct wfStatusInformer and register processStatusUpdate() to be called when there's a Status Update
	ocm.wfStatusInformer = ocm.newWorkflowStatusResultInformer()
	go ocm.wfStatusInformer.Informer().Run(ctx.Done())

	return ocm
}

// process Workflow additions and updates
func (ocm *OCMProcessor) ProcessWorkflow(ctx context.Context, wf *wfv1.Workflow) error {

	log.Infof("processing Workflow in OCM Processor: %+v\n", wf)

	// locate the label which indicates the cluster name (which is the namespace that our Manifest Work will go)
	mwNamespace, found := wf.Labels[common.LabelKeyCluster]
	if !found {
		return fmt.Errorf("In multicluster mode, the Workflow Controller requires all Workflows to contain label %s", common.LabelKeyCluster)
	}

	wf.ResourceVersion = "" //todo: why do I need to do this?
	wf.GenerateName = ""

	// use the Workflow UUID to derive the ManifestWork name
	mwName := string(wf.UID)
	wflabels := wf.GetLabels()
	wflabels[common.LabelKeyHubWorkflowUID] = string(wf.UID)
	wf.SetLabels(wflabels)
	manifestWork := ocm.generateManifestWork(mwName, mwNamespace, wf)
	log.Debugf("generated Manifest Work in OCM Processor: %+v\n", manifestWork)

	// attempt to create ManifestWork with this name/namespace
	_, err := ocm.ocmworkclient.ManifestWorks(mwNamespace).Create(ctx, manifestWork, metav1.CreateOptions{})
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			log.Infof("Found existing ManifestWork: name=%s, namespace=%s", mwName, mwNamespace)
		} else {
			return fmt.Errorf("Failed creating ManifestWork: %+v, err=%v", manifestWork, err)
		}
	} else {
		log.Infof("successfully created Manifest Work: name=%s, namespace=%s", mwName, mwNamespace)
	}

	return nil
}

func (ocm *OCMProcessor) ProcessWorkflowDeletion(ctx context.Context, wf *wfv1.Workflow) error {
	log.Infof("processing Workflow Deletion in OCM Processor: %+v\n", wf)

	// locate the label which indicates the cluster name (namespace of ManifestWork)
	mwNamespace, found := wf.Labels[common.LabelKeyCluster]
	if !found {
		return fmt.Errorf("In multicluster mode, the Workflow Controller requires all Workflows to contain label %s", common.LabelKeyCluster)
	}

	// use the Workflow UUID to derive the ManifestWork name
	mwName := string(wf.UID)

	// delete the ManifestWork
	err := ocm.ocmworkclient.ManifestWorks(mwNamespace).Delete(ctx, mwName, metav1.DeleteOptions{})

	return err
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
			Labels:    map[string]string{common.LabelKeyHubWorkflowUID: string(workflow.ObjectMeta.UID)},
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

func (ocm *OCMProcessor) newWorkflowStatusResultInformer() wfextvv1alpha1.WorkflowStatusResultInformer {
	log.Info("constructing WorkflowStatusResultInformer")
	informer := externalversions.NewSharedInformerFactoryWithOptions(
		ocm.wfclientset,
		workflowResultStatusResyncPeriod).Argoproj().V1alpha1().WorkflowStatusResults()
	informer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {

				log.Info("noticed new WorkflowStatusResult")
				wfResult := obj.(*wfv1.WorkflowStatusResult)
				log.Infof("cast to WorkflowStatusResult: %+v\n", wfResult)
				ocm.processStatusUpdate(context.Background(), wfResult)
			},
		})
	return informer
}
