package controller

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

var (
	age   = env.LookupEnvDurationOr("HEALTHZ_AGE", 5*time.Minute)
	limit = int64(env.LookupEnvIntOr("HEALTHZ_LIST_LIMIT", 200))
)

// https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-http-request
// If we are in a state where there are any workflows that have not been reconciled in the last 2m, we've gone wrong.
func (wfc *WorkflowController) Healthz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	instanceID := wfc.Config.InstanceID
	instanceIDSelector := func() string {
		if instanceID != "" {
			return common.LabelKeyControllerInstanceID + "=" + instanceID
		}
		return "!" + common.LabelKeyControllerInstanceID
	}()
	labelSelector := "!" + common.LabelKeyPhase + "," + instanceIDSelector
	err := func() error {
		// avoid problems with informers, but directly querying the API
		list, err := wfc.wfclientset.ArgoprojV1alpha1().Workflows(wfc.managedNamespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector, Limit: limit})
		if err != nil {
			return err
		}
		for _, wf := range list.Items {
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
