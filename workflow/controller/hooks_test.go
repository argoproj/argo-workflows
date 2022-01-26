package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestExecuteWfLifeCycleHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: lifecycle-hook-bgsf6
  namespace: argo
  resourceVersion: "33638"
spec:
  entrypoint: main
  hooks:
    exit:
      template: http
    error:
      expression: workflow.status == "Error"
      template: http
    running:
      expression: workflow.status == "Running"
      template: http
  templates:
  - name: main
    steps:
    - - name: step1
        template: heads
  - container:
      args:
      - echo "it was heads"
      command:
      - sh
      - -c
      image: alpine:3.6
      name: ""
    name: heads
  - http:
      url: https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json
    name: http
status:
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
  finishedAt: "2022-01-26T19:25:42Z"
  nodes:
    lifecycle-hook-bgsf6:
      children:
      - lifecycle-hook-bgsf6-2367710970
      displayName: lifecycle-hook-bgsf6
      finishedAt: "2022-01-26T19:25:42Z"
      id: lifecycle-hook-bgsf6
      name: lifecycle-hook-bgsf6
      outboundNodes:
      - lifecycle-hook-bgsf6-3057272397
      phase: Error
      progress: 1/1
      resourcesDuration:
        cpu: 4
        memory: 2
      startedAt: "2022-01-26T19:23:48Z"
      templateName: main
      templateScope: local/lifecycle-hook-bgsf6
      type: Steps
    lifecycle-hook-bgsf6-2367710970:
      boundaryID: lifecycle-hook-bgsf6
      children:
      - lifecycle-hook-bgsf6-3057272397
      displayName: '[0]'
      finishedAt: "2022-01-26T19:25:42Z"
      id: lifecycle-hook-bgsf6-2367710970
      name: lifecycle-hook-bgsf6[0]
      phase: Error
      progress: 1/1
      resourcesDuration:
        cpu: 4
        memory: 2
      startedAt: "2022-01-26T19:23:48Z"
      templateScope: local/lifecycle-hook-bgsf6
      type: StepGroup
    lifecycle-hook-bgsf6-3057272397:
      boundaryID: lifecycle-hook-bgsf6
      displayName: step1
      finishedAt: "2022-01-26T19:25:41Z"
      hostNodeName: k3d-k3s-default-server-0
      id: lifecycle-hook-bgsf6-3057272397
      name: lifecycle-hook-bgsf6[0].step1
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: lifecycle-hook-bgsf6/lifecycle-hook-bgsf6-3057272397/main.log
        exitCode: "0"
      phase: Error
      progress: 1/1
      resourcesDuration:
        cpu: 4
        memory: 2
      startedAt: "2022-01-26T19:23:48Z"
      templateName: heads
      templateScope: local/lifecycle-hook-bgsf6
      type: Pod
  phase: Error
  progress: 1/1
  resourcesDuration:
    cpu: 4
    memory: 2
  startedAt: "2022-01-26T19:23:48Z"
`)

	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("lifecycle-hook-bgsf6.hooks.error")
	assert.NotNil(t, node)
	node = woc.wf.Status.Nodes.FindByDisplayName("lifecycle-hook-bgsf6.hooks.running")
	assert.Nil(t, node)
}

func TestExecuteTmplLifeCycleHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: lifecycle-hook-tmpl-levelg8mqq
  namespace: argo
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - hooks:
          running:
            expression: steps.step1.status == "Running"
            template: http
          error:
            expression: steps.step1.status == "Error"
            template: http
        name: step1
        template: echo
  - container:
      args:
      - echo "it was heads"
      command:
      - sh
      - -c
      image: alpine:3.6
    name: echo
  - http:
      url: https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json
    name: http
status:
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2022-01-26T21:28:37Z"
  nodes:
    lifecycle-hook-tmpl-levelg8mqq:
      children:
      - lifecycle-hook-tmpl-levelg8mqq-2902815070
      displayName: lifecycle-hook-tmpl-levelg8mqq
      finishedAt: "2022-01-26T21:28:37Z"
      id: lifecycle-hook-tmpl-levelg8mqq
      name: lifecycle-hook-tmpl-levelg8mqq
      outboundNodes:
      - lifecycle-hook-tmpl-levelg8mqq-3824639481
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2022-01-26T21:28:33Z"
      templateName: main
      templateScope: local/lifecycle-hook-tmpl-levelg8mqq
      type: Steps
    lifecycle-hook-tmpl-levelg8mqq-2902815070:
      boundaryID: lifecycle-hook-tmpl-levelg8mqq
      children:
      - lifecycle-hook-tmpl-levelg8mqq-3824639481
      displayName: '[0]'
      finishedAt: "2022-01-26T21:28:37Z"
      id: lifecycle-hook-tmpl-levelg8mqq-2902815070
      name: lifecycle-hook-tmpl-levelg8mqq[0]
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2022-01-26T21:28:33Z"
      templateScope: local/lifecycle-hook-tmpl-levelg8mqq
      type: StepGroup
    lifecycle-hook-tmpl-levelg8mqq-3824639481:
      boundaryID: lifecycle-hook-tmpl-levelg8mqq
      children:
      - lifecycle-hook-tmpl-levelg8mqq-4216202210
      displayName: step1
      finishedAt: "2022-01-26T21:28:36Z"
      hostNodeName: k3d-k3s-default-server-0
      id: lifecycle-hook-tmpl-levelg8mqq-3824639481
      name: lifecycle-hook-tmpl-levelg8mqq[0].step1
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: lifecycle-hook-tmpl-levelg8mqq/lifecycle-hook-tmpl-levelg8mqq-3824639481/main.log
        exitCode: "0"
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2022-01-26T21:28:33Z"
      templateName: echo
      templateScope: local/lifecycle-hook-tmpl-levelg8mqq
      type: Pod
  phase: Running
  progress: 1/1
  resourcesDuration:
    cpu: 1
    memory: 0
  startedAt: "2022-01-26T21:28:33Z"
`)

	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("step1.hooks.error")
	assert.NotNil(t, node)
	node = woc.wf.Status.Nodes.FindByDisplayName("step1.hooks.running")
	assert.Nil(t, node)
}
