package controller

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
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
		selector, err := labels.Parse(labelSelector)
		if err != nil {
			return err
		}
		// the wfc.wfInformer is nil if it is not the leader
		if wfc.wfInformer == nil {
			log.Info("healthz: current pod is not the leader")
			return nil
		}

		// establish a list of unreconciled workflows
		unreconciledWorkflows := []*wfv1.Workflow{}
		err = cache.ListAllByNamespace(wfc.wfInformer.GetIndexer(), wfc.managedNamespace, selector, func(m interface{}) {
			// Informer holds Workflows as type *Unstructured
			un := m.(*unstructured.Unstructured)
			// verify it's of type *Workflow (if not, it's an incorrectly formatted Workflow spec)
			wf, err := util.FromUnstructured(un)
			if err != nil {
				log.Warnf("Healthz check found an incorrectly formatted Workflow: %q (namespace %q)", un.GetName(), un.GetNamespace())
				return
			}

			unreconciledWorkflows = append(unreconciledWorkflows, wf)
		})
		if err != nil {
			return fmt.Errorf("Healthz check failed to list Workflows using Informer, err=%v", err)
		}
		// go through the unreconciled workflows to determine if any of them exceed the max allowed age
		for _, wf := range unreconciledWorkflows {
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
