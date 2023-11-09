package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
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
    podSpecPatch: |
      containers:
      - name: main
        resources:
          limits:
            memory: 1G
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
        name: first-on-completion-1
        path: /tmp/message
        s3:
          accessKeySecret:
            key: accesskey
            name: my-minio-cred-1
          bucket: my-bucket-2
          endpoint: minio:9000
          insecure: true
          key: first-on-completion-1
          secretKeySecret:
            key: secretkey
            name: my-minio-cred-1
      - artifactGC:
          podMetadata:
            annotations:
              annotation-key-1: annotation-value-3
          strategy: OnWorkflowCompletion
        name: first-on-completion-2
        path: /tmp/message
        s3:
          accessKeySecret:
            key: accesskey
            name: my-minio-cred-1
          bucket: my-bucket-3
          endpoint: minio:9000
          insecure: true
          key: first-on-completion-2
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
        key: on-deletion
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
        name: on-deletion
        path: /tmp/message
        s3:
          key: on-deletion
      - artifactGC:
          strategy: OnWorkflowCompletion
        name: second-on-completion
        path: /tmp/message
        s3:
          accessKeySecret:
            key: accesskey
            name: my-minio-cred-2
          bucket: my-bucket-3
          endpoint: minio:9000
          insecure: true
          key: second-on-completion
          secretKeySecret:
            key: secretkey
            name: my-minio-cred-2
status:
  artifactGCStatus:
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
          name: first-on-completion-1
          path: /tmp/message
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred-1
            bucket: my-bucket-2
            endpoint: minio:9000
            insecure: true
            key: first-on-completion-1
            secretKeySecret:
              key: secretkey
              name: my-minio-cred-1
        - artifactGC:
            podMetadata:
              annotations:
                annotation-key-1: annotation-value-3
            strategy: OnWorkflowCompletion
          name: first-on-completion-2
          path: /tmp/message
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred-1
            bucket: my-bucket-3
            endpoint: minio:9000
            insecure: true
            key: first-on-completion-2
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
          name: on-deletion
          path: /tmp/message
          s3:
            key: on-deletion
        - artifactGC:
            strategy: OnWorkflowCompletion
          name: second-on-completion
          path: /tmp/message
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred-2
            bucket: my-bucket-3
            endpoint: minio:9000
            insecure: true
            key: second-on-completion
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

	err := woc.processArtifactGCStrategy(ctx, wfv1.ArtifactGCOnWorkflowCompletion)
	assert.Nil(t, err)

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

	// We should have one Pod per:
	//  [ServiceAccount,PodMetadata]
	// and it should only consist of artifacts labeled with OnWorkflowCompletion

	assert.NotNil(t, pods)
	assert.Equal(t, 2, len((*pods).Items))
	var pod1 *corev1.Pod
	var pod2 *corev1.Pod
	for i, pod := range (*pods).Items {
		switch pod.Name {
		case "two-artgc-8tcvt-artgc-wfcomp-592587874":
			pod1 = &(*pods).Items[i]
		case "two-artgc-8tcvt-artgc-wfcomp-3953780960":
			pod2 = &(*pods).Items[i]
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
	//  verify patched pod spec
	assert.Equal(t, pod1.Spec.ServiceAccountName, "default")
	assert.Contains(t, pod1.Annotations, "annotation-key-1")
	assert.Equal(t, "annotation-value-1", pod1.Annotations["annotation-key-1"])
	volumesMap1 := make(map[string]struct{})
	for _, v := range pod1.Spec.Volumes {
		volumesMap1[v.Name] = struct{}{}
	}
	assert.Contains(t, volumesMap1, "my-minio-cred-1")
	assert.Contains(t, volumesMap1, "my-minio-cred-2")

	assert.Equal(t, pod2.Spec.ServiceAccountName, "default")
	assert.Contains(t, pod2.Annotations, "annotation-key-1")
	assert.Equal(t, "annotation-value-3", pod2.Annotations["annotation-key-1"])
	volumesMap2 := make(map[string]struct{})
	for _, v := range pod2.Spec.Volumes {
		volumesMap2[v.Name] = struct{}{}
	}
	assert.Contains(t, volumesMap2, "my-minio-cred-1")
	assert.NotContains(t, volumesMap2, "my-minio-cred-2")
	assert.Equal(t, "1G", pod1.Spec.Containers[0].Resources.Limits.Memory().String())

	///////////////////////////////////////////////////////////////////////////////////////////
	// Verify WorkflowArtifactGCTasks
	///////////////////////////////////////////////////////////////////////////////////////////
	wfats, err := wfatcs.List(ctx, metav1.ListOptions{}) //todo: add ListOptions if this works
	if err != nil {
		panic(err)
	}

	// We should have on WFAT per Pod (for now until we implement the capability to have multiple)

	assert.NotNil(t, wfats)
	assert.Equal(t, 2, len((*wfats).Items))

	var wfat1 *wfv1.WorkflowArtifactGCTask
	var wfat2 *wfv1.WorkflowArtifactGCTask
	for i, wfat := range (*wfats).Items {
		switch wfat.Name {
		case "two-artgc-8tcvt-artgc-wfcomp-592587874-0":
			wfat1 = &(*wfats).Items[i]
		case "two-artgc-8tcvt-artgc-wfcomp-3953780960-0":
			wfat2 = &(*wfats).Items[i]
		default:
			assert.Fail(t, fmt.Sprintf("WorkflowArtifactGCTask name '%s' doesn't match expected", wfat.Name))
		}
	}

	assert.Condition(t, func() bool {
		return wfat1 != nil && wfat2 != nil
	})

	// Verify that the ArchiveLocation and list of artifacts on each is correct
	assert.Contains(t, wfat1.Spec.ArtifactsByNode, "two-artgc-8tcvt-802059674")
	assert.Contains(t, wfat1.Spec.ArtifactsByNode["two-artgc-8tcvt-802059674"].Artifacts, "first-on-completion-1")
	assert.NotContains(t, wfat1.Spec.ArtifactsByNode["two-artgc-8tcvt-802059674"].Artifacts, "on-deletion")
	assert.Contains(t, wfat1.Spec.ArtifactsByNode, "two-artgc-8tcvt-1079173309")
	assert.Equal(t, "my-bucket-3", wfat1.Spec.ArtifactsByNode["two-artgc-8tcvt-1079173309"].ArchiveLocation.S3.Bucket)
	assert.Contains(t, wfat1.Spec.ArtifactsByNode["two-artgc-8tcvt-1079173309"].Artifacts, "second-on-completion")
	assert.NotContains(t, wfat1.Spec.ArtifactsByNode["two-artgc-8tcvt-1079173309"].Artifacts, "on-deletion")

	assert.Contains(t, wfat2.Spec.ArtifactsByNode, "two-artgc-8tcvt-802059674")
	assert.Contains(t, wfat2.Spec.ArtifactsByNode["two-artgc-8tcvt-802059674"].Artifacts, "first-on-completion-2")
	assert.NotContains(t, wfat2.Spec.ArtifactsByNode["two-artgc-8tcvt-802059674"].Artifacts, "on-deletion")

}

var artgcTask = `apiVersion: argoproj.io/v1alpha1
kind: WorkflowArtifactGCTask
metadata:
  creationTimestamp: "2022-08-03T20:29:01Z"
  generation: 1
  labels:
    workflows.argoproj.io/artifact-gc-pod: two-artgc-8tcvt-artgc-wfcomp-2166136261
  name: two-artgc-8tcvt-artgc-wfcomp-2166136261-0
  namespace: argo
  ownerReferences:
  - apiVersion: argoproj.io/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: Workflow
    name: two-artgc-8tcvt
    uid: 98ecc84d-5aed-4bcd-bc9d-01daaa2b9948
  resourceVersion: "7950481"
  uid: 1a988e8b-25c3-45a2-8a71-3b75da48679d
spec:
  artifactsByNode:
    two-artgc-8tcvt-1079173309:
      archiveLocation:
        s3:
          accessKeySecret:
            key: accesskey
            name: my-minio-cred
          bucket: my-bucket-3
          endpoint: minio:9000
          insecure: true
          key: default-to-be-overridden
          secretKeySecret:
            key: secretkey
            name: my-minio-cred
      artifacts:
        second-on-completion:
          artifactGC:
            strategy: OnWorkflowCompletion
          name: second-on-completion
          path: /tmp/message
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket-2
            endpoint: minio:9000
            insecure: true
            key: second-on-completion
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
    two-artgc-8tcvt-4033701975:
      archiveLocation:
        archiveLogs: true
        s3:
          accessKeySecret:
            key: accesskey
            name: my-minio-cred
          bucket: my-bucket
          endpoint: minio:9000
          insecure: true
          key: '{{workflow.name}}/{{pod.name}}'
          secretKeySecret:
            key: secretkey
            name: my-minio-cred
      artifacts:
        first-on-completion-2:
          name: first-on-completion-2
          path: /tmp/message
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket-3
            endpoint: minio:9000
            insecure: true
            key: first-on-completion-2
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        main-logs:
          name: main-logs
          s3:
            key: two-artgc-8tcvt/two-artgc-8tcvt-first-4033701975/main.log
status:
  artifactResultsByNode:
    two-artgc-8tcvt-1079173309:
      artifactResults:
        second-on-completion:
          name: second-on-completion
          success: true
    two-artgc-8tcvt-802059674:
      artifactResults:
        first-on-completion-2:
          name: first-on-completion-2
          success: false
          error: 'something went wrong'
        main-logs:
          name: main-logs
          success: true
`

func TestProcessCompletedWorkflowArtifactGCTask(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(artgcWorkflow)
	wfat := wfv1.MustUnmarshalWorkflowArtifactGCTask(artgcTask)
	cancel, controller := newController(wf)
	defer cancel()

	woc := newWorkflowOperationCtx(wf, controller)
	woc.wf.Status.ArtifactGCStatus = &wfv1.ArtGCStatus{}

	// verify that we update these Status fields:
	// - Artifact.Deleted
	// - Conditions

	_, err := woc.processCompletedWorkflowArtifactGCTask(wfat, "OnWorkflowCompletion")
	assert.Nil(t, err)

	for _, expectedArtifact := range []struct {
		nodeName     string
		artifactName string
		deleted      bool
	}{
		{
			"two-artgc-8tcvt-1079173309",
			"second-on-completion",
			true,
		},
		{
			"two-artgc-8tcvt-802059674",
			"first-on-completion-2",
			false,
		},
		{
			"two-artgc-8tcvt-802059674",
			"main-logs",
			true,
		},
	} {

		node := woc.wf.Status.Nodes[expectedArtifact.nodeName]
		artifact := node.Outputs.Artifacts.GetArtifactByName(expectedArtifact.artifactName)
		if artifact == nil {
			panic(fmt.Sprintf("can't find artifact named %s in node %s", expectedArtifact.artifactName, expectedArtifact.nodeName))
		}
		assert.Equal(t, expectedArtifact.deleted, artifact.Deleted)

		if expectedArtifact.deleted {
			var gcFailureCondition *wfv1.Condition
			for i, condition := range woc.wf.Status.Conditions {
				if condition.Type == wfv1.ConditionTypeArtifactGCError {
					gcFailureCondition = &woc.wf.Status.Conditions[i]
					break
				}
			}
			assert.NotNil(t, gcFailureCondition)
			assert.Equal(t, metav1.ConditionTrue, gcFailureCondition.Status)
			assert.Contains(t, gcFailureCondition.Message, "something went wrong")
		}
	}

}

func TestWorkflowHasArtifactGC(t *testing.T) {
	tests := []struct {
		name                      string
		workflowArtGCStrategySpec string
		artifactGCStrategySpec    string
		expectedResult            bool
	}{
		{
			name: "WorkflowSpecGC_Completion",
			workflowArtGCStrategySpec: `
              artifactGC:
                strategy: OnWorkflowCompletion`,
			artifactGCStrategySpec: "",
			expectedResult:         true,
		},
		{
			name:                      "ArtifactSpecGC_Completion",
			workflowArtGCStrategySpec: "",
			artifactGCStrategySpec: `
                      artifactGC:
                        strategy: OnWorkflowCompletion`,
			expectedResult: true,
		},
		{
			name: "WorkflowSpecGC_Deletion",
			workflowArtGCStrategySpec: `
              artifactGC:
                strategy: OnWorkflowDeletion`,
			artifactGCStrategySpec: "",
			expectedResult:         true,
		},
		{
			name:                      "ArtifactSpecGC_Deletion",
			workflowArtGCStrategySpec: "",
			artifactGCStrategySpec: `
                      artifactGC:
                        strategy: OnWorkflowDeletion`,
			expectedResult: true,
		},
		{
			name:                      "NoGC",
			workflowArtGCStrategySpec: "",
			artifactGCStrategySpec:    "",
			expectedResult:            false,
		},
		{
			name: "WorkflowSpecGC_None",
			workflowArtGCStrategySpec: `
              artifactGC:
                strategy: ""`,
			artifactGCStrategySpec: "",
			expectedResult:         false,
		},
		{
			name: "ArtifactSpecGC_None",
			workflowArtGCStrategySpec: `
              artifactGC:
                strategy: OnWorkflowDeletion`,
			artifactGCStrategySpec: `
                      artifactGC:
                        strategy: Never`,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			workflowSpec := fmt.Sprintf(`
            apiVersion: argoproj.io/v1alpha1
            kind: Workflow
            metadata:
              generateName: artifact-passing-
            spec:
              entrypoint: whalesay
              %s
              templates:
              - name: whalesay
                container:
                  image: docker/whalesay:latest
                  command: [sh, -c]
                  args: ["sleep 1; cowsay hello world | tee /tmp/hello_world.txt"]
                outputs:
                  artifacts:
                    - name: out
                      path: /out
                      s3:
                        key: out
                        %s`, tt.workflowArtGCStrategySpec, tt.artifactGCStrategySpec)

			wf := wfv1.MustUnmarshalWorkflow(workflowSpec)
			cancel, controller := newController(wf)
			defer cancel()
			woc := newWorkflowOperationCtx(wf, controller)

			hasArtifact := woc.HasArtifactGC()

			assert.Equal(t, hasArtifact, tt.expectedResult)
		})
	}

}
