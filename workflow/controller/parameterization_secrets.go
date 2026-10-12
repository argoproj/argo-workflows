package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// Check only a newly dispatched Pod of an opted-in Workflow. Completed nodes
// and already created Pods retain native semantics and are never failed by a
// later credential expiry during an unrelated reconciliation.
func (woc *wfOperationCtx) validateParameterPodSecrets(ctx context.Context, pod *corev1.Pod) error {
	if woc.wf.Annotations[p.Group+"/submission"] == "" {
		return nil
	}
	if pod.Namespace != woc.wf.Namespace {
		return fmt.Errorf("parameterized Pod credential namespace mismatch")
	}
	secretCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err := p.ValidatePodExternalSecretLeases(secretCtx, pod, func(ctx context.Context, name string) (*corev1.Secret, error) {
		return woc.controller.kubeclientset.CoreV1().Secrets(woc.wf.Namespace).Get(ctx, name, metav1.GetOptions{})
	})
	cancel()
	if err == nil {
		return nil
	}
	// The informer can lag a successful Create whose response was lost. A later
	// expiry must not fail that existing execution; native AlreadyExists recovery
	// will read it. A different Workflow (including name reuse) cannot bypass this.
	podCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	existing, readErr := woc.controller.kubeclientset.CoreV1().Pods(pod.Namespace).Get(podCtx, pod.Name, metav1.GetOptions{})
	if readErr == nil && woc.wf.UID != "" {
		owner := metav1.GetControllerOf(existing)
		if owner != nil && owner.APIVersion == "argoproj.io/v1alpha1" && owner.Kind == "Workflow" && owner.UID == woc.wf.UID {
			return nil
		}
	}
	return err
}
