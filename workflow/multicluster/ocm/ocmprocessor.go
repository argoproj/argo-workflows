package ocm

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/rest"

	v1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"k8s.io/client-go/tools/cache"
	ocmclusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	ocmclusterinterface "open-cluster-management.io/api/client/cluster/clientset/versioned/typed/cluster/v1beta1"
	ocmworkclient "open-cluster-management.io/api/client/work/clientset/versioned"
	ocmworkinterface "open-cluster-management.io/api/client/work/clientset/versioned/typed/work/v1"
	ocmclustertypesv1 "open-cluster-management.io/api/cluster/v1beta1"
	ocmworktypesv1 "open-cluster-management.io/api/work/v1"
)

type OCMProcessor struct {
	wfInformer         cache.SharedIndexInformer                   // this one gets passed in
	wfStatusInformer   wfextvv1alpha1.WorkflowStatusResultInformer // this one gets constructed locally
	restConfig         *rest.Config
	wfclientset        wfclientset.Interface
	ocmWorkClient      ocmworkinterface.WorkV1Interface
	ocmPlacementClient ocmclusterinterface.ClusterV1beta1Interface
}

var (
	workflowResultStatusResyncPeriod = 1 * time.Minute
)

func NewOCMProcessor(ctx context.Context, wfInformer cache.SharedIndexInformer, restConfig *rest.Config, wfclientset wfclientset.Interface) *OCMProcessor {
	ocm := &OCMProcessor{wfInformer: wfInformer,
		restConfig:  restConfig,
		wfclientset: wfclientset}

	ocmClient := ocmworkclient.NewForConfigOrDie(ocm.restConfig)
	ocm.ocmWorkClient = ocmClient.WorkV1()
	ocm.ocmPlacementClient = ocmclusterclient.NewForConfigOrDie(ocm.restConfig).ClusterV1beta1()

	// construct wfStatusInformer and register processStatusUpdate() to be called when there's a Status Update
	ocm.wfStatusInformer = ocm.newWorkflowStatusResultInformer()
	go ocm.wfStatusInformer.Informer().Run(ctx.Done())

	return ocm
}

// process Workflow additions and updates
func (ocm *OCMProcessor) ProcessWorkflow(ctx context.Context, wf *wfv1.Workflow) error {

	log.Infof("processing Workflow in OCM Processor: %+v\n", wf)

	mwNamespace, err := ocm.getTargetNamespace(ctx, wf)
	if err != nil {
		return err
	}

	// use the Workflow UUID to derive the ManifestWork name
	mwName := string(wf.UID)

	return ocm.createManifestWork(ctx, mwName, mwNamespace, wf)
}

func (ocm *OCMProcessor) getTargetNamespace(ctx context.Context, wf *wfv1.Workflow) (string, error) {
	// either the cluster label or the placement label should be set

	// see if the label which indicates the cluster name (which is the namespace that our Manifest Work will go) is present
	mwNamespace, found := wf.Labels[common.LabelKeyCluster]
	if found {
		log.Debugf("found cluster label: will apply ManifestWork to namespace %q", mwNamespace)
		return mwNamespace, nil
	}

	log.Debug("did not find cluster label, will try placement")

	// the placement label and placement namespace label need to be present then
	placementLabel, placementLabelFound := wf.Labels[common.LabelKeyPlacement]
	placementNamespaceLabel, nsFound := wf.Labels[common.LabelKeyPlacementNamespace]
	if !placementLabelFound || !nsFound {
		return "", fmt.Errorf("In multicluster mode, the Workflow Controller requires all Workflows to contain either label %s or both labels %s and %s",
			common.LabelKeyCluster, common.LabelKeyPlacement, common.LabelKeyPlacementNamespace)
	}

	// look for any PlacementDecisions that exist for this Placement and delete them so we can refresh immediately
	requirement, err := labels.NewRequirement(ocmclustertypesv1.PlacementLabel, selection.Equals, []string{placementLabel})
	if err != nil {
		return "", fmt.Errorf("unable to create new PlacementDecision label requirement: err=%v", err)
	}

	labelSelector := labels.NewSelector().Add(*requirement)
	placementDecisions := &ocmclustertypesv1.PlacementDecisionList{}
	listopts := metav1.ListOptions{}
	listopts.LabelSelector = labelSelector.String()
	//listopts.Namespace = placementNamespaceLabel

	// delete any existing PlacementDecision so that a new one will get created
	placementDecisions, err = ocm.ocmPlacementClient.PlacementDecisions(placementNamespaceLabel).List(ctx, listopts)
	if err != nil {
		log.Error(err, "unable to list PlacementDecisions")
		return "", err
	}
	for _, pd := range placementDecisions.Items {
		log.Debugf("found PlacementDecision %q, deleting it", pd.Name)
		if err := ocm.ocmPlacementClient.PlacementDecisions(placementNamespaceLabel).Delete(ctx, pd.Name, metav1.DeleteOptions{}); err != nil {
			log.Error(err, "unable to delete PlacementDecision")
			return "", err
		}
	}

	placementDecisions = &ocmclustertypesv1.PlacementDecisionList{}

	// give it 10 seconds to create a new one
	log.Debug("seeing if a new PlacementDecision gets created")
	err = wait.PollImmediate(time.Second, time.Second*10, func() (bool, error) {
		placementDecisions, err = ocm.ocmPlacementClient.PlacementDecisions(placementNamespaceLabel).List(ctx, listopts)
		if err != nil {
			return false, err
		}

		if len(placementDecisions.Items) == 0 {
			return false, nil
		}
		pd := placementDecisions.Items[0]
		if len(pd.Status.Decisions) == 0 {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return "", errors.New("unable to check if PlacementDecision exist")
	}
	if len(placementDecisions.Items) == 0 {
		return "", fmt.Errorf("no PlacementDecision found for Placement name=%q, namespace=%q", placementLabel, placementNamespaceLabel)
	}
	pd := placementDecisions.Items[0]
	if len(pd.Status.Decisions) == 0 {
		return "", fmt.Errorf("PlacementDecision for Placement name=%q, namespace=%q found but has no Decisions", placementLabel, placementNamespaceLabel)
	}

	managedClusterName := pd.Status.Decisions[0].ClusterName
	if len(managedClusterName) == 0 {
		return "", fmt.Errorf("PlacementDecision for Placement name=%q, namespace=%q has a Decision whose ClusterName is empty: %+v",
			placementLabel, placementNamespaceLabel, pd.Status.Decisions[0])
	}
	log.Debugf("found PlacementDecision %q indicating we should place on managed cluster (namespace) %q", pd.Name, managedClusterName)

	// managedClusterName is the name of the namespace used for that cluster
	return managedClusterName, nil
}

func (ocm *OCMProcessor) createManifestWork(ctx context.Context, mwName string, mwNamespace string, wf *wfv1.Workflow) error {
	wf.ResourceVersion = ""
	wf.GenerateName = ""
	// todo: we shouldn't need these lines
	//wflabels := wf.GetLabels()
	//wflabels[common.LabelKeyHubWorkflowUID] = string(wf.UID)
	//wf.SetLabels(wflabels)

	manifestWork := ocm.generateManifestWork(mwName, mwNamespace, wf)
	log.Debugf("generated Manifest Work in OCM Processor: %+v\n", manifestWork)

	// attempt to create ManifestWork with this name/namespace
	_, err := ocm.ocmWorkClient.ManifestWorks(mwNamespace).Create(ctx, manifestWork, metav1.CreateOptions{})
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
	err := ocm.ocmWorkClient.ManifestWorks(mwNamespace).Delete(ctx, mwName, metav1.DeleteOptions{})

	return err
}

// find Workflow associated with WorkflowStatusResult and update it
/*func (ocm *OCMProcessor) processStatusUpdate(ctx context.Context, wfStatus *wfv1.WorkflowStatusResult) error {

	return nil
}*/

// generateManifestWork creates the ManifestWork that wraps the Workflow as payload
// With the status sync feedback of Workflow's phase
func (ocm *OCMProcessor) generateManifestWork(name, namespace string, workflow *wfv1.Workflow) *ocmworktypesv1.ManifestWork {
	return &ocmworktypesv1.ManifestWork{ // TODO use OCM API helper to generate manifest work.
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{common.LabelKeyHubWorkflowUID: string(workflow.ObjectMeta.UID)},
			//Labels:    map[string]string{LabelKeyEnableOCMStatusSync: strconv.FormatBool(true)}, // todo: why is Mike using this?
			//Annotations: map[string]string{AnnotationKeyHubWorkflowNamespace: workflow.Namespace,
			//	AnnotationKeyHubWorkflowName: workflow.Name},
		},
		Spec: ocmworktypesv1.ManifestWorkSpec{
			Workload: ocmworktypesv1.ManifestsTemplate{
				Manifests: []ocmworktypesv1.Manifest{{RawExtension: runtime.RawExtension{Object: workflow}}},
			},
			ManifestConfigs: []ocmworktypesv1.ManifestConfigOption{
				{
					ResourceIdentifier: ocmworktypesv1.ResourceIdentifier{
						Group:     v1alpha1.SchemeGroupVersion.Group,
						Resource:  "workflows", // TODO find the constant value from the argo API for this field
						Namespace: workflow.Namespace,
						Name:      workflow.Name,
					},
					FeedbackRules: []ocmworktypesv1.FeedbackRule{
						{Type: ocmworktypesv1.JSONPathsType, JsonPaths: []ocmworktypesv1.JsonPath{{Name: "phase", Path: ".status.phase"}}},
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

				err := ocm.processStatusUpdate(context.Background(), wfResult)
				if err != nil {
					log.Errorf("failed to process WorkflowStatusResult: err=%v", err)
				}
			},
			UpdateFunc: func(oldobj interface{}, newObj interface{}) {
				log.Info("noticed updated WorkflowStatusResult")
				wfResult := newObj.(*wfv1.WorkflowStatusResult)
				log.Infof("cast to WorkflowStatusResult: %+v\n", wfResult)
				ocm.processStatusUpdate(context.Background(), wfResult)
			},
		})
	return informer

}
