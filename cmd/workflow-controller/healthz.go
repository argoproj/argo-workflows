package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

var (
	age = env.LookupEnvDurationOr("HEALTHZ_AGE", 5*time.Minute)
)

func init() {
	log.WithField("age", "age").Info("healthz config")
}

// https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-http-request
// If we are in a state where there are any workflows that have not been reconciled in the last 2m, we've gone wrong.
func healthz(ctx context.Context, wfclientset wfclientset.Interface, managedNamespace string) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			list, err := wfclientset.ArgoprojV1alpha1().Workflows(managedNamespace).List(ctx, metav1.ListOptions{LabelSelector: "!" + common.LabelKeyPhase})
			if err != nil {
				return err
			}
			for _, wf := range list.Items {
				if time.Since(wf.GetCreationTimestamp().Time) > age {
					return fmt.Errorf("workflow never reconciled")
				}
			}
			return nil
		}()
		if err != nil {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		}
	})
}
