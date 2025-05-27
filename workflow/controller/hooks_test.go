package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
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
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("lifecycle-hook-bgsf6.hooks.error")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
	node = woc.wf.Status.Nodes.FindByDisplayName("lifecycle-hook-bgsf6.hooks.running")
	assert.Nil(t, node)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
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
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("step1.hooks.error")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
	node = woc.wf.Status.Nodes.FindByDisplayName("step1.hooks.running")
	assert.Nil(t, node)
}

const wftWithHook = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-whalesay-template
  namespace: default
spec:
  hooks:
    exit:
      template: http
    running:
      expression: workflow.status == "Running"
      template: http
  templates:
    - name: main
      steps:
        - - name: step-1
            hooks:
              exit:
                expression: steps["step-1"].status == "Running"
                template: http
              error:
                expression: steps["step-1"].status == "Error"
                template: http
            template: echo
        - - name: step2
            hooks:
              exit:
                expression: steps.step2.status == "Running"
                template: http
              error:
                expression: steps.step2.status == "Error"
                template: http
            template: echo

    - name: echo
      container:
        image: alpine:3.6
        command: [sh, -c]
        args: ["echo \"it was heads\""]

    - name: http
      http:
        # url: http://dummy.restapiexample.com/api/v1/employees
        url: "https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json"
  volumes:
    - name: data
      emptyDir: {}
`

func TestWorkflowTemplateRefWithHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: workflow-template-whalesay-template-hjgcg
spec:
  arguments:
    parameters:
    - name: message
      value: test
  entrypoint: main
  workflowTemplateRef:
    name: workflow-template-whalesay-template
status:
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  estimatedDuration: 6
  finishedAt: "2022-01-28T00:09:11Z"
  nodes:
    workflow-template-whalesay-template-hjgcg:
      children:
      - workflow-template-whalesay-template-hjgcg-2088741593
      displayName: workflow-template-whalesay-template-hjgcg
      estimatedDuration: 4
      finishedAt: "2022-01-28T00:09:11Z"
      id: workflow-template-whalesay-template-hjgcg
      name: workflow-template-whalesay-template-hjgcg
      outboundNodes:
      - workflow-template-whalesay-template-hjgcg-2492429498
      phase: Running
      progress: 2/2
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2022-01-28T00:09:02Z"
      templateName: main
      templateScope: local/
      type: Steps
    workflow-template-whalesay-template-hjgcg-1408741:
      boundaryID: workflow-template-whalesay-template-hjgcg
      children:
      - workflow-template-whalesay-template-hjgcg-1411106350
      - workflow-template-whalesay-template-hjgcg-412929452
      - workflow-template-whalesay-template-hjgcg-948010596
      displayName: step-1
      finishedAt: "2022-01-28T00:09:05Z"
      hostNodeName: k3d-k3s-default-server-0
      id: workflow-template-whalesay-template-hjgcg-1408741
      name: workflow-template-whalesay-template-hjgcg[0].step-1
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: workflow-template-whalesay-template-hjgcg/workflow-template-whalesay-template-hjgcg-1408741/main.log
        exitCode: "0"
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2022-01-28T00:09:02Z"
      templateName: echo
      templateScope: local/
      type: Pod
    workflow-template-whalesay-template-hjgcg-948010596:
      boundaryID: workflow-template-whalesay-template-hjgcg
      children:
      - workflow-template-whalesay-template-hjgcg-2492429498
      displayName: '[1]'
      finishedAt: "2022-01-28T00:09:11Z"
      id: workflow-template-whalesay-template-hjgcg-948010596
      name: workflow-template-whalesay-template-hjgcg[1]
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2022-01-28T00:09:07Z"
      templateScope: local/
      type: StepGroup
    workflow-template-whalesay-template-hjgcg-2088741593:
      boundaryID: workflow-template-whalesay-template-hjgcg
      children:
      - workflow-template-whalesay-template-hjgcg-1408741
      displayName: '[0]'
      finishedAt: "2022-01-28T00:09:07Z"
      id: workflow-template-whalesay-template-hjgcg-2088741593
      name: workflow-template-whalesay-template-hjgcg[0]
      phase: Running
      progress: 2/2
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2022-01-28T00:09:02Z"
      templateScope: local/
      type: StepGroup
    workflow-template-whalesay-template-hjgcg-2492429498:
      boundaryID: workflow-template-whalesay-template-hjgcg
      children:
      - workflow-template-whalesay-template-hjgcg-369778373
      - workflow-template-whalesay-template-hjgcg-2107567333
      displayName: step2
      finishedAt: "2022-01-28T00:09:10Z"
      hostNodeName: k3d-k3s-default-server-0
      id: workflow-template-whalesay-template-hjgcg-2492429498
      name: workflow-template-whalesay-template-hjgcg[1].step2
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: workflow-template-whalesay-template-hjgcg/workflow-template-whalesay-template-hjgcg-2492429498/main.log
        exitCode: "0"
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2022-01-28T00:09:07Z"
      templateName: echo
      templateScope: local/
      type: Pod
  phase: Running
  progress: 2/2
  resourcesDuration:
    cpu: 2
    memory: 0
  startedAt: "2022-01-28T00:09:02Z"
  storedTemplates:
    namespaced/workflow-template-whalesay-template/echo:
      container:
        args:
        - echo "it was heads"
        command:
        - sh
        - -c
        image: alpine:3.6
        name: ""
      inputs: {}
      metadata: {}
      name: echo
      outputs: {}
    namespaced/workflow-template-whalesay-template/main:
      inputs: {}
      metadata: {}
      name: main
      outputs: {}
      steps:
      - - hooks:
            exit:
              expression: steps["step-1"].status == "Running"
              template: http
            success:
              expression: steps["step-1"].status == "Succeeded"
              template: http
          name: step-1
          template: echo
      - - hooks:
            exit:
              expression: steps.step2.status == "Running"
              template: http
            success:
              expression: steps.step2.status == "Succeeded"
              template: http
          name: step2
          template: echo
  storedWorkflowTemplateSpec:
    activeDeadlineSeconds: 300
    arguments:
      parameters:
      - name: message
        value: test
    entrypoint: main
    hooks:
      exit:
        template: http
      running:
        expression: workflow.status == "Running"
        template: http
    podSpecPatch: |
      terminationGracePeriodSeconds: 3
    templates:
    - name: main
      steps:
      - - hooks:
            exit:
              template: http
            error:
              expression: steps["step-1"].status == "Error"
              template: http
          name: step-1
          template: echo
      - - hooks:
            exit:
              expression: steps.step2.status == "Running"
              template: http
            error:
              expression: steps.step2.status == "Error"
              template: http
          name: step2
          template: echo
    - container:
        args:
        - echo "it was heads"
        command:
        - sh
        - -c
        image: alpine:3.6
        name: ""
      name: echo
    - http:
        url: https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json
      name: http
    ttlStrategy:
      secondsAfterCompletion: 600
    volumes:
    - emptyDir: {}
      name: data
    workflowTemplateRef:
      name: workflow-template-whalesay-template
`)
	cancel, controller := newController(wf, wfv1.MustUnmarshalWorkflowTemplate(wftWithHook))
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("step-1.hooks.error")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
}

func TestTemplateRefWithHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: workflow-template-whalesay-template-1
  namespace: default
spec:

  entrypoint: main

  templates:
  - name: main
    steps:
    - - hooks:
          exit:
            expression: steps["step-1"].status == "Running"
            templateRef:
              name: workflow-template-whalesay-template
              template: http
          error:
            expression: steps["step-1"].status == "Error"
            template: ""
            templateRef:
              name: workflow-template-whalesay-template
              template: http
        name: step-1
        templateRef:
          name: workflow-template-whalesay-template
          template: main
  ttlStrategy:
    secondsAfterCompletion: 600
status:
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2022-01-28T07:01:04Z"
  nodes:
    workflow-template-whalesay-template-1:
      children:
      - workflow-template-whalesay-template-1-1761665519
      displayName: workflow-template-whalesay-template-1
      finishedAt: "2022-01-28T07:01:04Z"
      id: workflow-template-whalesay-template-1
      name: workflow-template-whalesay-template-1
      outboundNodes:
      - workflow-template-whalesay-template-1-79192944
      phase: Running
      progress: 2/2
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2022-01-28T07:00:55Z"
      templateName: main
      templateScope: local/workflow-template-whalesay-template-1
      type: Steps
    workflow-template-whalesay-template-1-1236010667:
      boundaryID: workflow-template-whalesay-template-1-1438125631
      children:
      - workflow-template-whalesay-template-1-2715724931
      displayName: '[0]'
      finishedAt: "2022-01-28T07:01:00Z"
      id: workflow-template-whalesay-template-1-1236010667
      name: workflow-template-whalesay-template-1[0].step-1[0]
      phase: Running
      progress: 2/2
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2022-01-28T07:00:55Z"
      templateScope: namespaced/workflow-template-whalesay-template
      type: StepGroup
    workflow-template-whalesay-template-1-1438125631:
      boundaryID: workflow-template-whalesay-template-1
      children:
      - workflow-template-whalesay-template-1-1236010667
      - workflow-template-whalesay-template-1-986640140
      - workflow-template-whalesay-template-1-1359034694
      displayName: step-1
      finishedAt: "2022-01-28T07:01:04Z"
      id: workflow-template-whalesay-template-1-1438125631
      name: workflow-template-whalesay-template-1[0].step-1
      outboundNodes:
      - workflow-template-whalesay-template-1-79192944
      phase: Running
      progress: 2/2
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2022-01-28T07:00:55Z"
      templateRef:
        name: workflow-template-whalesay-template
        template: main
      templateScope: local/workflow-template-whalesay-template-1
      type: Steps
    workflow-template-whalesay-template-1-1761665519:
      boundaryID: workflow-template-whalesay-template-1
      children:
      - workflow-template-whalesay-template-1-1438125631
      displayName: '[0]'
      finishedAt: "2022-01-28T07:01:04Z"
      id: workflow-template-whalesay-template-1-1761665519
      name: workflow-template-whalesay-template-1[0]
      phase: Running
      progress: 2/2
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2022-01-28T07:00:55Z"
      templateScope: local/workflow-template-whalesay-template-1
      type: StepGroup
    workflow-template-whalesay-template-1-2377035854:
      boundaryID: workflow-template-whalesay-template-1-1438125631
      children:
      - workflow-template-whalesay-template-1-79192944
      displayName: '[1]'
      finishedAt: "2022-01-28T07:01:04Z"
      id: workflow-template-whalesay-template-1-2377035854
      name: workflow-template-whalesay-template-1[0].step-1[1]
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2022-01-28T07:01:00Z"
      templateScope: namespaced/workflow-template-whalesay-template
      type: StepGroup
    workflow-template-whalesay-template-1-2715724931:
      boundaryID: workflow-template-whalesay-template-1-1438125631
      children:
      - workflow-template-whalesay-template-1-3456567280
      - workflow-template-whalesay-template-1-1403479850
      - workflow-template-whalesay-template-1-2377035854
      displayName: step-1
      finishedAt: "2022-01-28T07:00:58Z"
      hostNodeName: k3d-k3s-default-server-0
      id: workflow-template-whalesay-template-1-2715724931
      name: workflow-template-whalesay-template-1[0].step-1[0].step-1
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: workflow-template-whalesay-template-1/workflow-template-whalesay-template-1-2715724931/main.log
        exitCode: "0"
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2022-01-28T07:00:55Z"
      templateName: echo
      templateScope: namespaced/workflow-template-whalesay-template
      type: Pod
  phase: Running
  progress: 2/2
  resourcesDuration:
    cpu: 3
    memory: 0
  startedAt: "2022-01-28T07:00:55Z"
  storedTemplates:
    namespaced/workflow-template-whalesay-template/echo:
      container:
        args:
        - echo "it was heads"
        command:
        - sh
        - -c
        image: alpine:3.6
        name: ""
        resources: {}
      inputs: {}
      metadata: {}
      name: echo
      outputs: {}
    namespaced/workflow-template-whalesay-template/http:
      http:
        url: https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json
      inputs: {}
      metadata: {}
      name: http
      outputs: {}
    namespaced/workflow-template-whalesay-template/main:
      inputs: {}
      metadata: {}
      name: main
      outputs: {}
      steps:
      - - arguments: {}
          hooks:
            exit:
              arguments: {}
              expression: steps["step-1"].status == "Running"
              template: http
            error:
              arguments: {}
              expression: steps["step-1"].status == "Error"
              template: http
          name: step-1
          template: echo
      - - arguments: {}
          hooks:
            exit:
              arguments: {}
              expression: steps.step2.status == "Running"
              template: http
            error:
              arguments: {}
              expression: steps.step2.status == "Error"
              template: http
          name: step2
          template: echo

`)
	cancel, controller := newController(wf, wfv1.MustUnmarshalWorkflowTemplate(wftWithHook))
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for _, node := range woc.wf.Status.Nodes {
		fmt.Println(node.DisplayName, node.Phase)
	}
	node := woc.wf.Status.Nodes.FindByDisplayName("step-1.hooks.error")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
}

func TestWfTemplateRefWithHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: lifecycle-hook-fh7t4
  namespace: argo
spec:
  workflowTemplateRef:
    name: lifecycle-hook
status:
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  estimatedDuration: 7
  nodes:
    lifecycle-hook-fh7t4:
      children:
      - lifecycle-hook-fh7t4-2818004451
      - lifecycle-hook-fh7t4-4144954648
      displayName: lifecycle-hook-fh7t4
      estimatedDuration: 5
      finishedAt: "2022-03-25T05:45:49Z"
      id: lifecycle-hook-fh7t4
      name: lifecycle-hook-fh7t4
      outboundNodes:
      - lifecycle-hook-fh7t4-3883827164
      phase: Running
      progress: 2/2
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2022-03-25T05:45:45Z"
      templateName: main
      templateScope: local/
      type: Steps
    lifecycle-hook-fh7t4-2818004451:
      boundaryID: lifecycle-hook-fh7t4
      children:
      - lifecycle-hook-fh7t4-3883827164
      displayName: '[0]'
      estimatedDuration: 5
      finishedAt: "2022-03-25T05:45:49Z"
      id: lifecycle-hook-fh7t4-2818004451
      name: lifecycle-hook-fh7t4[0]
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2022-03-25T05:45:45Z"
      templateScope: local/
      type: StepGroup
    lifecycle-hook-fh7t4-3883827164:
      boundaryID: lifecycle-hook-fh7t4
      displayName: step1
      estimatedDuration: 4
      finishedAt: "2022-03-25T05:45:47Z"
      hostNodeName: k3d-k3s-default-server-0
      id: lifecycle-hook-fh7t4-3883827164
      name: lifecycle-hook-fh7t4[0].step1
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: lifecycle-hook-fh7t4/lifecycle-hook-fh7t4-3883827164/main.log
        exitCode: "0"
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2022-03-25T05:45:45Z"
      templateName: heads
      templateScope: local/
      type: Pod
  phase: Running
  storedTemplates:
    namespaced/lifecycle-hook/heads:
      container:
        args:
        - echo "it was heads"
        command:
        - sh
        - -c
        image: alpine:3.6
        name: ""
        resources: {}
      inputs: {}
      metadata: {}
      name: heads
      outputs: {}
    namespaced/lifecycle-hook/main:
      inputs: {}
      metadata: {}
      name: main
      outputs: {}
      steps:
      - - arguments: {}
          name: step1
          template: heads
  storedWorkflowTemplateSpec:
    activeDeadlineSeconds: 300
    arguments: {}
    entrypoint: main
    hooks:
      exit:
        arguments: {}
        template: http
      Failed:
        arguments: {}
        expression: workflow.status == "Failed"
        template: http
    podSpecPatch: |
      terminationGracePeriodSeconds: 3
    templates:
    - inputs: {}
      metadata: {}
      name: main
      outputs: {}
      steps:
      - - arguments: {}
          name: step1
          template: heads
    - container:
        args:
        - echo "it was heads"
        command:
        - sh
        - -c
        image: alpine:3.6
        name: ""
        resources: {}
      inputs: {}
      metadata: {}
      name: heads
      outputs: {}
    - http:
        url: http://dummy.restapiexample.com/api/v1/employees
      inputs: {}
      metadata: {}
      name: http
      outputs: {}
    ttlStrategy:
      secondsAfterCompletion: 600
    workflowTemplateRef:
      name: lifecycle-hook
`)
	cancel, controller := newController(wf, wfv1.MustUnmarshalWorkflowTemplate(wftWithHook))
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("lifecycle-hook-fh7t4.hooks.Failed")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
}

func TestWfHookHasFailures(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hook-failures
  namespace: argo
spec:
  entrypoint: intentional-fail
  hooks:
    failure:
      expression: workflow.status == "Failed"
      template: message
      arguments:
        parameters:
          - name: message
            value: |
              Workflow {{ workflow.name }} {{ workflow.status }} {{ workflow.failures }}
  templates:
    - name: intentional-fail
      container:
        image: alpine:latest
        command: [sh, -c]
        args: ["echo intentional failure; exit 1"]
    - name: message
      inputs:
        parameters:
          - name: message
      script:
        image: alpine:latest
        command: [sh]
        source: |
          echo {{ inputs.parameters.message }}
`)

	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("hook-failures.hooks.failure")
	assert.NotNil(t, node)
	assert.Contains(t,
		woc.globalParams[common.GlobalVarWorkflowFailures],
		`[{\"displayName\":\"hook-failures\",\"message\":\"Pod failed\",\"templateName\":\"intentional-fail\",\"phase\":\"Failed\",\"podName\":\"hook-failures\"`,
	)
	assert.Equal(t, wfv1.NodePending, node.Phase)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	err, _ := woc.podReconciliation(ctx)
	require.NoError(t, err)
	node = woc.wf.Status.Nodes.FindByDisplayName("hook-failures.hooks.failure")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
}

func TestWfHookNoExpression(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hook-failures
  namespace: argo
spec:
  entrypoint: message
  hooks:
    failure:
      template: message
  templates:
    - name: message
      script:
        image: alpine:latest
        command: [sh]
        source: |
          echo Hi
`)

	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Equal(t, "invalid spec: hooks.failure Expression required", woc.wf.Status.Message)
}

func TestStepHookNoExpression(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hook-failures
  namespace: argo
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: step-1
            template: message
            hooks:
              foo:
                template: message
    - name: message
      script:
        image: alpine:latest
        command: [sh]
        source: |
          echo Hi
`)

	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Equal(t, "invalid spec: templates.main.steps[0].step-1.foo Expression required", woc.wf.Status.Message)
}

func TestWfHookWfWaitForTriggeredHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hook-running
  namespace: argo
spec:
  entrypoint: main
  hooks:
    running:
      expression: workflow.status == "Running"
      template: sleep
    # This hook never triggered by following test.
    # To guarantee workflow does not wait forever for untriggered hooks.
    failure:
      expression: workflow.status == "Failed"
      template: sleep
  templates:
    - name: main
      container:
        image: alpine:latest
        command: [sh, -c]
        args: ["echo", "This template finish fastest"]
    - name: sleep
      script:
        image: alpine:latest
        command: [sh]
        source: |
          sleep 10
`)

	// Setup
	cancel, controller := newController(wf)
	defer cancel()
	ctx := context.Background()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodRunning)

	// Check if running hook is triggered
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("hook-running.hooks.running")
	assert.NotNil(t, node)
	assert.Equal(t, wfv1.NodePending, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)

	// Make all pods running
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	node = woc.wf.Status.Nodes.FindByDisplayName("hook-running.hooks.running")
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)

	// Make main pod completed
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	pod, _ := podcs.Get(ctx, "hook-running", metav1.GetOptions{})
	pod.Status.Phase = apiv1.PodSucceeded
	updatedPod, _ := podcs.Update(ctx, pod, metav1.UpdateOptions{})
	woc.wf.Status.MarkTaskResultComplete(woc.nodeID(pod))
	_ = woc.controller.PodController.TestingPodInformer().GetStore().Update(updatedPod)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.Progress("1/2"), woc.wf.Status.Progress)
	node = woc.wf.Status.Nodes.FindByDisplayName("hook-running")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.Nil(t, node.NodeFlag)
	node = woc.wf.Status.Nodes.FindByDisplayName("hook-running.hooks.running")
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// Make all pod completed
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.Progress("2/2"), woc.wf.Status.Progress)
	node = woc.wf.Status.Nodes.FindByDisplayName("hook-running.hooks.running")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)
	node = woc.wf.Status.Nodes.FindByDisplayName("hook-running")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.Nil(t, node.NodeFlag)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestWfTemplHookWfWaitForTriggeredHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hook-running
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: job
            template: exit0
            hooks:
              running:
                expression: steps['job'].status == "Running"
                template: hook
              failed:
                expression: steps['job'].status == "Failed"
                template: hook
    - name: hook
      script:
        image: alpine:latest
        command: [/bin/sh]
        source: |
          sleep 5
    - name: exit0
      script:
        image: alpine:latest
        command: [/bin/sh]
        source: |
          exit 0
`)

	// Setup
	cancel, controller := newController(wf)
	defer cancel()
	ctx := context.Background()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodRunning)

	// Check if running hook is triggered
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("job.hooks.running")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodePending, node.Phase)

	// Make all pods running
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	node = woc.wf.Status.Nodes.FindByDisplayName("job.hooks.running")
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)

	// Make main pod completed
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	pods, _ := podcs.List(ctx, metav1.ListOptions{})
	pod := pods.Items[0]
	pod.Status.Phase = apiv1.PodSucceeded
	updatedPod, _ := podcs.Update(ctx, &pod, metav1.UpdateOptions{})
	_ = woc.controller.PodController.TestingPodInformer().GetStore().Update(updatedPod)
	woc.wf.Status.MarkTaskResultComplete(woc.nodeID(&pod))
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.Progress("1/2"), woc.wf.Status.Progress)
	node = woc.wf.Status.Nodes.FindByDisplayName("job")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.Nil(t, node.NodeFlag)
	node = woc.wf.Status.Nodes.FindByDisplayName("job.hooks.running")
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// Make all pod completed
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.Progress("2/2"), woc.wf.Status.Progress)
	node = woc.wf.Status.Nodes.FindByDisplayName("job.hooks.running")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)
	node = woc.wf.Status.Nodes.FindByDisplayName("job")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.Nil(t, node.NodeFlag)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}
