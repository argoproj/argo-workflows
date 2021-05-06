package controller

import (
	"context"
	"testing"

	"github.com/ghodss/yaml"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func getWfOperationCtx() *wfOperationCtx {
	return &wfOperationCtx{
		wf: &v1alpha1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "1",
				Namespace: "default",
			},
			Status: v1alpha1.WorkflowStatus{
				Nodes: map[string]v1alpha1.NodeStatus{
					"1":  {ID: "1", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeSucceeded, BoundaryID: "1"},
					"2":  {ID: "2", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeFailed, BoundaryID: "1"},
					"3":  {Type: v1alpha1.NodeTypeSteps, Phase: v1alpha1.NodeFailed, BoundaryID: "1"},
					"4":  {Type: v1alpha1.NodeTypeDAG, Phase: v1alpha1.NodeError, BoundaryID: "1"},
					"5":  {ID: "1", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeRunning, BoundaryID: "1"},
					"5a": {ID: "2", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeRunning, BoundaryID: "1", SynchronizationStatus: &v1alpha1.NodeSynchronizationStatus{Waiting: "yes"}},
					"6":  {ID: "1", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodePending, BoundaryID: "1"},
					"7":  {ID: "2", Type: v1alpha1.NodeTypeSteps, Phase: v1alpha1.NodeRunning, BoundaryID: "1"},
					"8":  {ID: "1", Type: v1alpha1.NodeTypeDAG, Phase: v1alpha1.NodePending, BoundaryID: "1"},

					"9":  {Type: v1alpha1.NodeTypeSteps, Phase: v1alpha1.NodeFailed, BoundaryID: "2"},
					"10": {Type: v1alpha1.NodeTypeDAG, Phase: v1alpha1.NodeError, BoundaryID: "2"},
					"11": {ID: "1", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodeRunning, BoundaryID: "2"},
					"12": {ID: "2", Type: v1alpha1.NodeTypePod, Phase: v1alpha1.NodePending, BoundaryID: "2"},
				},
			},
		},
	}
}

var podStr = `apiVersion: v1
kind: Pod
metadata:
  labels:
    workflows.argoproj.io/completed: "false"
    workflows.argoproj.io/workflow: steps-tt9wq
  name: 1
  namespace: default
spec:
  containers:
  - args:
    - hello1
    command:
    - cowsay
    env:
    - name: ARGO_CONTAINER_NAME
      value: main
    - name: ARGO_INCLUDE_SCRIPT_OUTPUT
      value: "false"
    image: docker/whalesay
    imagePullPolicy: Always
    name: main
    resources: {}
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: default-token-mgv4v
      readOnly: true
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeSelector:
    spot: "true"
  priority: 0
  restartPolicy: Never
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  shareProcessNamespace: true
  terminationGracePeriodSeconds: 30
  tolerations:
  - effect: NoSchedule
    key: spot
    operator: Equal
    value: "true"
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
  volumes:
  - downwardAPI:
      defaultMode: 420
      items:
      - fieldRef:
          apiVersion: v1
          fieldPath: metadata.annotations
        path: annotations
    name: podmetadata
  - name: my-minio-cred
    secret:
      defaultMode: 420
      items:
      - key: accesskey
        path: accesskey
      - key: secretkey
        path: secretkey
      secretName: my-minio-cred
  - name: default-token-mgv4v
    secret:
      defaultMode: 420
      secretName: default-token-mgv4v
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2021-05-04T21:35:34Z"
    message: '0/1 nodes are available: 1 node(s) didn''t match node selector.'
    reason: Unschedulable
    status: "False"
    type: PodScheduled
  phase: Pending
  qosClass: Burstable
`

func TestCounters(t *testing.T) {
	woc := getWfOperationCtx()
	var pod v1.Pod
	err := yaml.Unmarshal([]byte(podStr), &pod)
	assert.NoError(t, err)
	cancel, controller := newController(pod)
	defer cancel()
	woc.controller = controller
	_, err = controller.kubeclientset.CoreV1().Pods("default").Create(context.Background(), &pod, metav1.CreateOptions{})
	assert.NoError(t, err)
	pod.Name = "2"
	_, err = controller.kubeclientset.CoreV1().Pods("default").Create(context.Background(), &pod, metav1.CreateOptions{})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), woc.getActivePods("1"))
	// No BoundaryID requested
	assert.Equal(t, int64(4), woc.getActivePods(""))
	assert.Equal(t, int64(5), woc.getActiveChildren("1"))
	assert.Equal(t, int64(3), woc.getUnsuccessfulChildren("1"))
	assert.Equal(t, int64(2), woc.getActivePods("2"))
	assert.Equal(t, int64(2), woc.getActiveChildren("2"))
	assert.Equal(t, int64(2), woc.getUnsuccessfulChildren("2"))
}
