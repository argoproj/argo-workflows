package archive

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"

	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/hydrator"
	"github.com/argoproj/argo/workflow/util"
)

// This handle will make many attempts to archive a workflow if there is a problem.
// Firstly, when it is marked as pending.
// Secondly, when it is re-synced.
// Finally, when it is deleted.
type handler struct {
	hydrator          hydrator.Interface
	workflowArchive   sqldb.WorkflowArchive
	workflowInterface versioned.Interface
}

func (h handler) OnAdd(obj interface{}) {
	h.archive(obj)
}

func (h handler) OnUpdate(_, obj interface{}) {
	h.archive(obj)
}

func (h handler) OnDelete(obj interface{}) {
	h.archive(obj)
}

func (h handler) archive(obj interface{}) {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok && un.GetLabels()[common.LabelKeyArchiveStatus] != "Pending" {
		return
	}
	logCtx := log.WithFields(log.Fields{"namespace": un.GetNamespace(), "name": un.GetName()})
	wf, err := util.FromUnstructured(un)
	if err != nil {
		logCtx.WithError(err).Error("failed to convert from unstructured to workflow")
		return
	}
	err = h.hydrator.Hydrate(wf)
	if err != nil {
		logCtx.WithError(err).Error("failed to hydrate workflow")
		return
	}
	err = wait.ExponentialBackoff(retry.DefaultBackoff, func() (bool, error) {
		err := h.workflowArchive.ArchiveWorkflow(wf)
		return err == nil, err
	})
	if err != nil {
		logCtx.WithError(err).Error("failed to archive workflow")
		return
	}
	patch, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]string{
				common.LabelKeyArchiveStatus: "Succeeded",
			},
		},
	})
	logCtx.Info("Marking workflow archiving as succeeded")
	_, err = h.workflowInterface.ArgoprojV1alpha1().Workflows(un.GetNamespace()).Patch(un.GetName(), types.MergePatchType, patch)
	if err != nil {
		logCtx.WithError(err).Error("failed to mark workflow as archived")
		return
	}
}

func NewHandler(hydrator hydrator.Interface, workflowArchive sqldb.WorkflowArchive, workflowInterface versioned.Interface) cache.FilteringResourceEventHandler {
	return cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			un, ok := obj.(*unstructured.Unstructured)
			return ok && un.GetLabels()[common.LabelKeyArchiveStatus] == "Pending"
		},
		Handler: &handler{hydrator, workflowArchive, workflowInterface},
	}
}

var _ cache.ResourceEventHandler = &handler{}
