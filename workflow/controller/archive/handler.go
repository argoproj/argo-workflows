package archive

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/hydrator"
	"github.com/argoproj/argo/workflow/util"
)

type handler struct {
	hydrator          hydrator.Interface
	workflowArchive   sqldb.WorkflowArchive
	workflowInterface versioned.Interface
}

func (h handler) OnAdd(obj interface{}) {
	h.archive(obj)
}

func (h handler) OnUpdate(_, newObj interface{}) {
	h.archive(newObj)
}

func (h handler) OnDelete(obj interface{}) {
	h.archive(obj)
}

func (h handler) archive(obj interface{}) {
	un, ok := obj.(*unstructured.Unstructured)
	if ok && un.GetLabels()[common.LabelKeyArchiveStatus] == "Pending" {
		logCtx := log.WithFields(log.Fields{"namespace": un.GetNamespace(), "name": un.GetName()})
		archiveStatus := "Error"
		wf, err := util.FromUnstructured(un)
		if err != nil {
			logCtx.WithError(err).Error("Failed to convert from unstructured to workflow")
		} else {
			err := h.hydrator.Hydrate(wf)
			if err != nil {
				logCtx.WithError(err).Error("Failed to hydrate workflow")
			} else {
				err := h.workflowArchive.ArchiveWorkflow(wf)
				if err != nil {
					logCtx.WithError(err).Error("Failed to archive workflow")
				} else {
					archiveStatus = "Succeeded"
				}
			}
		}
		patch, _ := json.Marshal(map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels": map[string]string{
					common.LabelKeyArchiveStatus: archiveStatus,
				},
			},
		})
		logCtx.WithField("archiveStatus", archiveStatus).Info("Updating workflow archive status")
		_, err = h.workflowInterface.ArgoprojV1alpha1().Workflows(un.GetNamespace()).Patch(un.GetName(), types.MergePatchType, patch)
		if err != nil {
			logCtx.WithError(err).Error("Failed to mark workflow as archived")
		}

	}
}

func NewHandler(hydrator hydrator.Interface, workflowArchive sqldb.WorkflowArchive, workflowInterface versioned.Interface) cache.FilteringResourceEventHandler {
	return cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			un, ok := obj.(*unstructured.Unstructured)
			return ok && un.GetLabels()[common.LabelKeyArchiveStatus] != ""
		},
		Handler: &handler{hydrator, workflowArchive, workflowInterface},
	}
}

var _ cache.ResourceEventHandler = &handler{}
