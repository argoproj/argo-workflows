package controller

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/argoproj/argo-workflows/v3/pkg/client/listers/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

var (
	age = env.LookupEnvDurationOr("HEALTHZ_AGE", 5*time.Minute)
)

// https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-http-request
// If we are in a state where there are any workflows that have not been reconciled in the last 2m, we've gone wrong.
func (wfc *WorkflowController) Healthz(w http.ResponseWriter, r *http.Request) {
	instanceID := wfc.Config.InstanceID
	instanceIDSelector := func() string {
		if instanceID != "" {
			return common.LabelKeyControllerInstanceID + "=" + instanceID
		}
		return "!" + common.LabelKeyControllerInstanceID
	}()
	labelSelector := "!" + common.LabelKeyPhase + "," + instanceIDSelector
	err := func() error {
		seletor, err := labels.Parse(labelSelector)
		if err != nil {
			return err
		}
		// the wfc.wfInformer is nil if it is not the leader
		if wfc.wfInformer == nil {
			log.Info("healthz: current pod is not the leader")
			return nil
		}
		lister := v1alpha1.NewWorkflowLister(wfc.wfInformer.GetIndexer())
		list, err := lister.Workflows(wfc.managedNamespace).List(seletor)
		if err != nil {
			return err
		}
		for _, wf := range list {
			if time.Since(wf.GetCreationTimestamp().Time) > age {
				return fmt.Errorf("workflow never reconciled: %s", wf.Name)
			}
		}
		return nil
	}()
	if err != nil {
		log.WithField("err", err).
			WithField("managedNamespace", wfc.managedNamespace).
			WithField("instanceID", instanceID).
			WithField("labelSelector", labelSelector).
			WithField("age", age).
			Info("healthz")
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}
}
