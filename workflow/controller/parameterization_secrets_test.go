package controller

import (
	"testing"

	"time"

	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func dispatchSecretFixture() (*wfv1.Workflow, *corev1.Secret) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata: {name: dispatch-secret, namespace: default, uid: workflow-uid}
spec:
  entrypoint: consume
  serviceAccountName: default
  templates:
  - name: consume
    container:
      image: alpine:3.22
      command: [echo, credential-consumed]
      env:
      - name: TOKEN
        valueFrom:
          secretKeyRef: {name: version, key: token}
`)
	wf.Annotations = map[string]string{p.Group + "/submission": "submission"}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "version", Namespace: "default", Labels: map[string]string{p.Group + "/external-source": "synthetic-source"}, Annotations: map[string]string{p.Group + "/external-valid-until": time.Now().Add(-time.Hour).Format(time.RFC3339Nano)}}, Immutable: ptr.To(true), Data: map[string][]byte{"token": []byte("synthetic-value-never-in-errors")}}
	return wf, secret
}

func TestParameterDispatchExpiredSecretStopsNewPod(t *testing.T) {
	for _, test := range []struct {
		name         string
		opted, valid bool
		wantPods     int
	}{
		{name: "expired after waiting", opted: true, wantPods: 0},
		{name: "refreshed lease", opted: true, valid: true, wantPods: 1},
		{name: "ordinary native workflow", wantPods: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, secret := dispatchSecretFixture()
			if !test.opted {
				wf.Annotations = nil
			}
			if test.valid {
				secret.Annotations[p.Group+"/external-valid-until"] = time.Now().Add(time.Hour).Format(time.RFC3339Nano)
			}
			before := secret.DeepCopy()
			cancel, controller := newController(ctx, wf, defaultServiceAccount)
			defer cancel()
			_, createErr := controller.kubeclientset.CoreV1().Secrets(wf.Namespace).Create(ctx, secret, metav1.CreateOptions{})
			require.NoError(t, createErr)
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)
			pods, err := controller.kubeclientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			require.Len(t, pods.Items, test.wantPods)
			if test.wantPods == 0 {
				require.Contains(t, woc.wf.Status.Message, "verification expired")
				require.NotContains(t, woc.wf.Status.Message, string(secret.Data["token"]))
			}
			require.Equal(t, before, secret)
		})
	}
}

func TestParameterDispatchExistingPodRecoveryAfterLeaseExpiry(t *testing.T) {
	for _, test := range []struct {
		name       string
		owner      types.UID
		controller bool
		wantError  bool
	}{
		{name: "same workflow informer lag", owner: "workflow-uid", controller: true},
		{name: "same name recreated workflow", owner: "other-workflow-uid", controller: true, wantError: true},
		{name: "non-controller reference", owner: "workflow-uid", wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf, secret := dispatchSecretFixture()
			cancel, controller := newController(ctx, wf, defaultServiceAccount)
			defer cancel()
			_, createErr := controller.kubeclientset.CoreV1().Secrets(wf.Namespace).Create(ctx, secret, metav1.CreateOptions{})
			require.NoError(t, createErr)
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "dispatch-secret-consume", Namespace: wf.Namespace}, Spec: corev1.PodSpec{Containers: []corev1.Container{*wf.Spec.Templates[0].Container}}}
			existing := pod.DeepCopy()
			existing.OwnerReferences = []metav1.OwnerReference{{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow", Name: wf.Name, UID: test.owner, Controller: ptr.To(test.controller)}}
			_, err := controller.kubeclientset.CoreV1().Pods(wf.Namespace).Create(ctx, existing, metav1.CreateOptions{})
			require.NoError(t, err)
			before := pod.DeepCopy()
			err = woc.validateParameterPodSecrets(ctx, pod)
			if test.wantError {
				require.ErrorContains(t, err, "verification expired")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, before, pod)
		})
	}
}

func TestParameterDispatchNativeWorkflowDoesNotReadSecrets(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, _ := dispatchSecretFixture()
	wf.Annotations = nil
	cancel, controller := newController(ctx, wf, defaultServiceAccount)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: wf.Namespace}, Spec: corev1.PodSpec{Containers: []corev1.Container{*wf.Spec.Templates[0].Container}}}
	require.NoError(t, woc.validateParameterPodSecrets(ctx, pod))
	woc.wf.Annotations = map[string]string{p.Group + "/submission": "submission"}
	pod.Namespace = "other"
	require.EqualError(t, woc.validateParameterPodSecrets(ctx, pod), "parameterized Pod credential namespace mismatch")
}
