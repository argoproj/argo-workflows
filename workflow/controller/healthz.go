package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

var (
	age = env.LookupEnvDurationOr(logging.InitLoggerInContext(), "HEALTHZ_AGE", 5*time.Minute)
)

func LogMiddleware(logger logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(logging.WithLogger(r.Context(), logger))
		next.ServeHTTP(w, r)
	})
}

// https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-http-request
// If we are in a state where there are any workflows that have not been reconciled in the last 2m, we've gone wrong.
func (wfc *WorkflowController) Healthz(w http.ResponseWriter, r *http.Request) {
	logger := logging.RequireLoggerFromContext(r.Context())

	instanceID := wfc.Config.InstanceID
	instanceIDSelector := func() string {
		if instanceID != "" {
			return common.LabelKeyControllerInstanceID + "=" + instanceID
		}
		return "!" + common.LabelKeyControllerInstanceID
	}()
	labelSelector := "!" + common.LabelKeyPhase + "," + instanceIDSelector
	err := func(ctx context.Context) error {
		selector, err := labels.Parse(labelSelector)
		if err != nil {
			return err
		}
		if !wfc.IsLeader() {
			logger.Info(ctx, "healthz: current pod is not the leader")
			return nil
		}

		// establish a list of unreconciled workflows
		unreconciledWorkflows := make(map[string]*wfv1.Workflow)
		err = cache.ListAllByNamespace(wfc.wfInformer.GetIndexer(), wfc.managedNamespace, selector, func(m interface{}) {
			// Informer holds Workflows as type *Unstructured
			un := m.(*unstructured.Unstructured)
			// verify it's of type *Workflow (if not, it's an incorrectly formatted Workflow spec)
			wf, err := util.FromUnstructured(un)
			if err != nil {
				logger.WithField("name", un.GetName()).WithField("namespace", un.GetNamespace()).Warn(ctx, "Healthz check found an incorrectly formatted Workflow")
				return
			}

			key := wf.Namespace + "/" + wf.Name
			unreconciledWorkflows[key] = wf
		})
		if err != nil {
			return fmt.Errorf("Healthz check failed to list Workflows using Informer, err=%w", err)
		}

		unreconciledExceedAge := false
		var firstExceededWorkflow *wfv1.Workflow

		for _, wf := range unreconciledWorkflows {
			if time.Since(wf.GetCreationTimestamp().Time) > age {
				unreconciledExceedAge = true
				firstExceededWorkflow = wf
				break
			}
		}

		noProgress := true
		if unreconciledExceedAge {
			logger.Info(ctx, "healthz: workflows exceed max age")
			// Check if there is progress by comparing with the last check:
			// If all workflows from last time are still present, it means no progress
			for key := range wfc.lastUnreconciledWorkflows {
				if _, exists := unreconciledWorkflows[key]; !exists {
					// At least one workflow has been reconciled, so there is progress
					noProgress = false
					break
				}
			}

			if noProgress && len(wfc.lastUnreconciledWorkflows) > 0 {
				return fmt.Errorf("workflow exceeds max age and no progress: %s/%s", firstExceededWorkflow.Namespace, firstExceededWorkflow.Name)
			}
		}

		// Update the cache for the next health check
		wfc.lastUnreconciledWorkflows = unreconciledWorkflows

		return nil
	}(r.Context())
	if err != nil {
		logger.WithError(err).
			WithFields(logging.Fields{
				"managedNamespace": wfc.managedNamespace,
				"instanceID":       instanceID,
				"labelSelector":    labelSelector,
				"age":              age,
			}).
			Info(r.Context(), "healthz")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
