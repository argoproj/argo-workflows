package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var artgcWorkflow = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  annotations:
    workflows.argoproj.io/pod-name-format: v2
  creationTimestamp: "2022-08-01T22:29:58Z"
  finalizers:
  - workflows.argoproj.io/artifact-gc
  generateName: two-artgc-
  generation: 12
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Succeeded
  name: two-artgc-8tcvt
  namespace: argo
  resourceVersion: "7738582"
  uid: 9fe40595-8612-4312-ba2c-d64bad1fb3ee
spec:
  activeDeadlineSeconds: 300
  arguments: {}
  artifactGC:
    podMetadata:
      annotations:
        annotation-key-1: annotation-value-1
        annotation-key-2: annotation-value-2
    serviceAccountName: default
  entrypoint: entrypoint
  podGC: {}
  podSpecPatch: |
    terminationGracePeriodSeconds: 3
  templates:
  - inputs: {}
    metadata: {}
    name: entrypoint
    outputs: {}
    steps:
    - - arguments: {}
        name: call-first
        template: first
    - - arguments: {}
        name: call-second
        template: second
  - container:
      args:
      - |
        echo "hello world" > /tmp/message
      command:
      - sh
      - -c
      image: argoproj/argosay:v2
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: first
    outputs:
      artifacts:
      - artifactGC:
          strategy: OnWorkflowCompletion
        name: on-completion-first-1
        path: /tmp/message
        s3:
          accessKeySecret:
            key: accesskey
            name: my-minio-cred-1
          bucket: my-bucket-2
          endpoint: minio:9000
          insecure: true
          key: on-completion-first-1
          secretKeySecret:
            key: secretkey
            name: my-minio-cred-1
      - artifactGC:
          podMetadata:
            annotations:
              annotation-key-1: annotation-value-3
          strategy: OnWorkflowCompletion
        name: on-completion-first-2
        path: /tmp/message
        s3:
          accessKeySecret:
            key: accesskey
            name: my-minio-cred-1
          bucket: my-bucket-3
          endpoint: minio:9000
          insecure: true
          key: on-completion-first-2
          secretKeySecret:
            key: secretkey
            name: my-minio-cred-1
  - archiveLocation:
      s3:
        accessKeySecret:
          key: accesskey
          name: my-minio-cred
        bucket: my-bucket-3
        endpoint: minio:9000
        insecure: true
        key: on-deletion-key-{{pod.name}}
        secretKeySecret:
          key: secretkey
          name: my-minio-cred
    container:
      args:
      - |
        echo "hello world" > /tmp/message
      command:
      - sh
      - -c
      image: argoproj/argosay:v2
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: second
    outputs:
      artifacts:
      - artifactGC:
          strategy: OnWorkflowDeletion
        name: on-deletion-key-{{pod.name}}
        path: /tmp/message
        s3:
          key: on-deletion-key-{{pod.name}}
      - artifactGC:
          strategy: OnWorkflowCompletion
        name: on-completion-second
        path: /tmp/message
        s3:
          accessKeySecret:
            key: accesskey
            name: my-minio-cred-2
          bucket: my-bucket-3
          endpoint: minio:9000
          insecure: true
          key: on-completion-second
          secretKeySecret:
            key: secretkey
            name: my-minio-cred-2
status:
  artifactGCStatus:
    podsRecouped:
      two-artgc-8tcvt-artgc-wfcomp-592587874: true
      two-artgc-8tcvt-artgc-wfcomp-3953780960: true
    strategiesProcessed:
      OnWorkflowCompletion: true
      OnWorkflowSuccess: true
  artifactRepositoryRef:
    artifactRepository:
      archiveLogs: true
      s3:
        accessKeySecret:
          key: accesskey
          name: my-minio-cred
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        secretKeySecret:
          key: secretkey
          name: my-minio-cred
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2022-08-01T22:30:09Z"
  nodes:
    two-artgc-8tcvt:
      children:
      - two-artgc-8tcvt-292960257
      displayName: two-artgc-8tcvt
      finishedAt: "2022-08-01T22:30:09Z"
      id: two-artgc-8tcvt
      name: two-artgc-8tcvt
      outboundNodes:
      - two-artgc-8tcvt-1079173309
      phase: Succeeded
      progress: 2/2
      resourcesDuration:
        cpu: 7
        memory: 4
      startedAt: "2022-08-01T22:29:58Z"
      templateName: entrypoint
      templateScope: local/two-artgc-8tcvt
      type: Steps
    two-artgc-8tcvt-225996876:
      boundaryID: two-artgc-8tcvt
      children:
      - two-artgc-8tcvt-1079173309
      displayName: '[1]'
      finishedAt: "2022-08-01T22:30:09Z"
      id: two-artgc-8tcvt-225996876
      name: two-artgc-8tcvt[1]
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 4
        memory: 2
      startedAt: "2022-08-01T22:30:04Z"
      templateScope: local/two-artgc-8tcvt
      type: StepGroup
    two-artgc-8tcvt-292960257:
      boundaryID: two-artgc-8tcvt
      children:
      - two-artgc-8tcvt-802059674
      displayName: '[0]'
      finishedAt: "2022-08-01T22:30:04Z"
      id: two-artgc-8tcvt-292960257
      name: two-artgc-8tcvt[0]
      phase: Succeeded
      progress: 2/2
      resourcesDuration:
        cpu: 7
        memory: 4
      startedAt: "2022-08-01T22:29:58Z"
      templateScope: local/two-artgc-8tcvt
      type: StepGroup
    two-artgc-8tcvt-802059674:
      boundaryID: two-artgc-8tcvt
      children:
      - two-artgc-8tcvt-225996876
      displayName: call-first
      finishedAt: "2022-08-01T22:30:02Z"
      hostNodeName: k3d-k3s-default-server-0
      id: two-artgc-8tcvt-802059674
      name: two-artgc-8tcvt[0].call-first
      outputs:
        artifacts:
        - artifactGC:
            strategy: OnWorkflowCompletion
          name: on-completion-first-1
          path: /tmp/message
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred-1
            bucket: my-bucket-2
            endpoint: minio:9000
            insecure: true
            key: on-completion-first-1
            secretKeySecret:
              key: secretkey
              name: my-minio-cred-1
        - artifactGC:
            podMetadata:
              annotations:
                annotation-key-1: annotation-value-3
            strategy: OnWorkflowCompletion
          name: on-completion-first-2
          path: /tmp/message
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred-1
            bucket: my-bucket-3
            endpoint: minio:9000
            insecure: true
            key: on-completion-first-2
            secretKeySecret:
              key: secretkey
              name: my-minio-cred-1
        - name: main-logs
          s3:
            key: two-artgc-8tcvt/two-artgc-8tcvt-first-802059674/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 3
        memory: 2
      startedAt: "2022-08-01T22:29:58Z"
      templateName: first
      templateScope: local/two-artgc-8tcvt
      type: Pod
    two-artgc-8tcvt-1079173309:
      boundaryID: two-artgc-8tcvt
      displayName: call-second
      finishedAt: "2022-08-01T22:30:07Z"
      hostNodeName: k3d-k3s-default-server-0
      id: two-artgc-8tcvt-1079173309
      name: two-artgc-8tcvt[1].call-second
      outputs:
        artifacts:
        - artifactGC:
            strategy: OnWorkflowDeletion
          name: on-deletion-key-two-artgc-8tcvt-second-1079173309
          path: /tmp/message
          s3:
            key: on-deletion-key-two-artgc-8tcvt-second-1079173309
        - artifactGC:
            strategy: OnWorkflowCompletion
          name: on-completion-second
          path: /tmp/message
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred-2
            bucket: my-bucket-3
            endpoint: minio:9000
            insecure: true
            key: on-completion-second
            secretKeySecret:
              key: secretkey
              name: my-minio-cred-2
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 4
        memory: 2
      startedAt: "2022-08-01T22:30:04Z"
      templateName: second
      templateScope: local/two-artgc-8tcvt
      type: Pod
  phase: Succeeded
  progress: 2/2
  resourcesDuration:
    cpu: 7
    memory: 4
  startedAt: "2022-08-01T22:29:58Z"

`

func TestProcessArtifactGCStrategy(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(artgcWorkflow)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.wf.Status.ArtifactGCStatus = &wfv1.ArtGCStatus{}

	woc.processArtifactGCStrategy(ctx, wfv1.ArtifactGCOnWorkflowCompletion)

	wfatcs := controller.wfclientset.ArgoprojV1alpha1().WorkflowArtifactGCTasks(woc.wf.GetNamespace())
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())

	// verify that the expected WFATs and Pods get created

	///////////////////////////////////////////////////////////////////////////////////////////
	// Verify Pods
	///////////////////////////////////////////////////////////////////////////////////////////
	pods, err := podcs.List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("deletethis: pods (using kubeclientset): %+v\n\n", pods)

	// We should have one Pod per:
	//  [ServiceAccount,PodMetadata]
	// and it should only consist of artifacts labeled with OnWorkflowCompletion

	assert.NotNil(t, pods)
	assert.Equal(t, 2, len((*pods).Items))
	var pod1 *corev1.Pod
	var pod2 *corev1.Pod
	for _, pod := range (*pods).Items {
		switch pod.Name {
		case "two-artgc-8tcvt-artgc-wfcomp-592587874":
			pod1 = &pod
		case "two-artgc-8tcvt-artgc-wfcomp-3953780960":
			pod2 = &pod
		default:
			assert.Fail(t, fmt.Sprintf("pod name '%s' doesn't match expected", pod.Name))
		}
	}

	assert.Condition(t, func() bool {
		return pod1 != nil && pod2 != nil
	})

	// For each Pod:
	//  verify ServiceAccount and Annotations
	//  verify that the right volume mounts get created
	assert.Equal(t, pod1.Spec.ServiceAccountName, "default")
	assert.Contains(t, pod1.Annotations, "annotation-key-1")
	assert.Equal(t, pod1.Annotations["annotation-key-1"], "annotation-value-1")
	volumesMap1 := make(map[string]struct{})
	for _, v := range pod1.Spec.Volumes {
		volumesMap1[v.Name] = struct{}{}
	}
	assert.Contains(t, volumesMap1, "my-minio-cred-1")
	assert.Contains(t, volumesMap1, "my-minio-cred-2")

	assert.Equal(t, pod2.Spec.ServiceAccountName, "default")
	assert.Contains(t, pod2.Annotations, "annotation-key-1")
	assert.Equal(t, pod2.Annotations["annotation-key-1"], "annotation-value-3")
	volumesMap2 := make(map[string]struct{})
	for _, v := range pod1.Spec.Volumes {
		volumesMap2[v.Name] = struct{}{}
	}
	assert.Contains(t, volumesMap2, "my-minio-cred-1")
	assert.NotContains(t, volumesMap2, "my-minio-cred-2")

	///////////////////////////////////////////////////////////////////////////////////////////
	// Verify WFATs
	///////////////////////////////////////////////////////////////////////////////////////////
	wfats, err := wfatcs.List(ctx, metav1.ListOptions{}) //todo: add ListOptions if this works
	if err != nil {
		panic(err)
	}
	fmt.Printf("deletethis: wfats=%+v\n", wfats.Items)
	// We should have on WFAT per Pod (for now until we implement the capability to have multiple)

	assert.NotNil(t, wfats)
	assert.Equal(t, 2, len((*wfats).Items))

	var wfat1 *wfv1.WorkflowArtifactGCTask
	var wfat2 *wfv1.WorkflowArtifactGCTask
	for _, wfat := range (*wfats).Items {
		switch wfat.Name {
		case "two-artgc-8tcvt-artgc-wfcomp-592587874":
			wfat1 = &wfat
		case "two-artgc-8tcvt-artgc-wfcomp-3953780960":
			wfat2 = &wfat
		default:
			assert.Fail(t, fmt.Sprintf("WorkflowArtifactGCTask name '%s' doesn't match expected", wfat.Name))
		}
	}

	assert.Condition(t, func() bool {
		return wfat1 != nil && wfat2 != nil
	})

	// Verify that the ArchiveLocation and list of artifacts on each is correct

}
