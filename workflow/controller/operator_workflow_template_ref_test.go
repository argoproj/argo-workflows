package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util"
)

func TestWorkflowTemplateRef(t *testing.T) {
	cancel, controller := newController(wfv1.MustUnmarshalWorkflow(wfWithTmplRef), wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wfv1.MustUnmarshalWorkflow(wfWithTmplRef), controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl).Spec.Templates, woc.execWf.Spec.Templates)
	assert.Equal(t, woc.wf.Spec.Entrypoint, woc.execWf.Spec.Entrypoint)
	// verify we copy these values
	assert.Len(t, woc.volumes, 1, "volumes from workflow template")
	// and these
	assert.Equal(t, "my-sa", woc.globalParams["workflow.serviceAccountName"])
	assert.Equal(t, "77", woc.globalParams["workflow.priority"])
}

func TestWorkflowTemplateRefWithArgs(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(wfTmpl)

	ctx := context.Background()
	t.Run("CheckArgumentPassing", func(t *testing.T) {
		args := []wfv1.Parameter{
			{
				Name:  "param1",
				Value: wfv1.AnyStringPtr("test"),
			},
		}
		wf.Spec.Arguments.Parameters = util.MergeParameters(wf.Spec.Arguments.Parameters, args)
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, "test", woc.globalParams["workflow.parameters.param1"])
	})
}

func TestWorkflowTemplateRefWithWorkflowTemplateArgs(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(wfTmpl)

	ctx := context.Background()
	t.Run("CheckArgumentFromWFT", func(t *testing.T) {
		args := []wfv1.Parameter{
			{
				Name:  "param1",
				Value: wfv1.AnyStringPtr("test"),
			},
		}
		wftmpl.Spec.Arguments.Parameters = util.MergeParameters(wf.Spec.Arguments.Parameters, args)
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, "test", woc.globalParams["workflow.parameters.param1"])
	})

	t.Run("CheckMergingWFDefaults", func(t *testing.T) {
		wfDefaultActiveS := int64(5)
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
		controller.Config.WorkflowDefaults = &wfv1.Workflow{
			Spec: wfv1.WorkflowSpec{
				ActiveDeadlineSeconds: &wfDefaultActiveS,
			},
		}
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfDefaultActiveS, *woc.execWf.Spec.ActiveDeadlineSeconds)
	})
	t.Run("CheckMergingWFTandWF", func(t *testing.T) {
		wfActiveS := int64(10)
		wftActiveS := int64(10)
		wfDefaultActiveS := int64(5)

		wftmpl.Spec.ActiveDeadlineSeconds = &wftActiveS
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
		controller.Config.WorkflowDefaults = &wfv1.Workflow{
			Spec: wfv1.WorkflowSpec{
				ActiveDeadlineSeconds: &wfDefaultActiveS,
			},
		}
		wf.Spec.ActiveDeadlineSeconds = &wfActiveS
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfActiveS, *woc.execWf.Spec.ActiveDeadlineSeconds)

		wf.Spec.ActiveDeadlineSeconds = nil
		woc = newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wftActiveS, *woc.execWf.Spec.ActiveDeadlineSeconds)
	})
}

const invalidWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: ui-workflow-error
  namespace: argo
spec:
  entrypoint: main
  workflowTemplateRef:
    name: not-exists
`

func TestWorkflowTemplateRefInvalidWF(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(invalidWF)
	t.Run("ProcessWFWithStoredWFT", func(t *testing.T) {
		cancel, controller := newController(wf)
		defer cancel()
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	})
}

var wftWithParam = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: params-test-1
  namespace: default
spec:
  entrypoint: main
  arguments:
    parameters:
      - name: a-a
        value: "10"
      - name: b
        value: ""
      - name: c-c
        value: "0"
      - name: d
        value: ""
      - name: e-e
        value: "10"
      - name: f
        value: ""
      - name: g-g
        value: "1"
      - name: h
        value: ""
      - name: i-i
        value: "{}"
      - name: things
        value: "[]"

  templates:
    - name: main
      steps:
        - - name: echoitems
            template: echo

    - name: echo
      container:
        image: busybox
        command: [echo]
        args: ["{{workflows.parameters.a-a}} = {{workflows.parameters.g-g}}"]
`

var wfWithParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: params-test-1-grx2n
  namespace: default
spec:
  arguments:
    parameters:
    - name: f
      value: f
    - name: g-g
      value: 2
    - name: h
      value: h
    - name: i-i
      value: '{}'
    - name: things
      value: '[{"a":"1","nested":{"B":"3"}},{"a":"2"}]'
    - name: a-a
      value: 5
  workflowTemplateRef:
    name: params-test-1
`

func TestWorkflowTemplateRefParamMerge(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfWithParam)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(wftWithParam)

	t.Run("CheckArgumentFromWF", func(t *testing.T) {
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wf.Spec.Arguments.Parameters, woc.wf.Spec.Arguments.Parameters)
	})
}

var wftWithArtifact = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: artifact-test-1
  namespace: test-namespace
spec:
  entrypoint: main
  arguments:
    artifacts:
    - name: binary-file
      http:
        url: https://a.server.io/file
    - name: data-file
      http:
        url: https://b.server.io/data

  templates:
    - name: main
      steps:
        - - name: process-data
            template: process

    - name: process
      inputs:
        artifacts:
          - name: binary-file
            path: /usr/local/bin/binfile
            mode: 0755
          - name: data-file
            path: /tmp/data
            mode: 0755
      container:
        image: busybox
        command: [sh, -c]
        args: ["binary-file /tmp/data"]
`

const wfWithTemplateWithArtifact = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-from-artifact-test-1-
  namespace: test-namespace
spec:
  arguments:
    artifacts:
    - name: own-file
      http:
        url: https://local/blob
  workflowTemplateRef:
    name: artifact-test-1
`

func TestWorkflowTemplateRefGetArtifactsFromTemplate(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfWithTemplateWithArtifact)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(wftWithArtifact)

	t.Run("CheckArtifactArgumentFromWF", func(t *testing.T) {
		cancel, controller := newController(wf, wftmpl)
		defer cancel()
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Len(t, woc.execWf.Spec.Arguments.Artifacts, 3)

		assert.Equal(t, "own-file", woc.execWf.Spec.Arguments.Artifacts[0].Name)
		assert.Equal(t, "binary-file", woc.execWf.Spec.Arguments.Artifacts[1].Name)
		assert.Equal(t, "data-file", woc.execWf.Spec.Arguments.Artifacts[2].Name)
	})
}

func TestWorkflowTemplateRefWithShutdownAndSuspend(t *testing.T) {
	t.Run("EntryPointMissingInStoredWfSpec", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
		cancel, controller := newController(wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
		defer cancel()
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.StoredWorkflowSpec.Suspend)
		wf1 := woc.wf.DeepCopy()
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodPending)
		wf1.Status.StoredWorkflowSpec.Entrypoint = ""
		woc1 := newWorkflowOperationCtx(wf1, controller)
		woc1.operate(ctx)
		assert.NotNil(t, woc1.wf.Status.StoredWorkflowSpec.Entrypoint)
		assert.Equal(t, woc.wf.Spec.Entrypoint, woc1.wf.Status.StoredWorkflowSpec.Entrypoint)
	})

	t.Run("WorkflowTemplateRefWithSuspend", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
		cancel, controller := newController(wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
		defer cancel()
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Nil(t, woc.wf.Status.StoredWorkflowSpec.Suspend)
		wf1 := woc.wf.DeepCopy()
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodPending)
		wf1.Spec.Suspend = pointer.BoolPtr(true)
		woc1 := newWorkflowOperationCtx(wf1, controller)
		woc1.operate(ctx)
		assert.NotNil(t, woc1.wf.Status.StoredWorkflowSpec.Suspend)
		assert.True(t, *woc1.wf.Status.StoredWorkflowSpec.Suspend)
	})
	t.Run("WorkflowTemplateRefWithShutdownTerminate", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
		cancel, controller := newController(wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
		defer cancel()
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Empty(t, woc.wf.Status.StoredWorkflowSpec.Shutdown)
		wf1 := woc.wf.DeepCopy()
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodPending)
		wf1.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
		woc1 := newWorkflowOperationCtx(wf1, controller)
		woc1.operate(ctx)
		assert.NotEmpty(t, woc1.wf.Status.StoredWorkflowSpec.Shutdown)
		assert.Equal(t, wfv1.ShutdownStrategyTerminate, woc1.wf.Status.StoredWorkflowSpec.Shutdown)
		for _, node := range woc1.wf.Status.Nodes {
			if assert.NotNil(t, node) {
				assert.Contains(t, node.Message, "workflow shutdown with strategy")
				assert.Contains(t, node.Message, "Terminate")
			}
		}
	})
	t.Run("WorkflowTemplateRefWithShutdownStop", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
		cancel, controller := newController(wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
		defer cancel()
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Empty(t, woc.wf.Status.StoredWorkflowSpec.Shutdown)
		wf1 := woc.wf.DeepCopy()
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodPending)
		wf1.Spec.Shutdown = wfv1.ShutdownStrategyStop
		woc1 := newWorkflowOperationCtx(wf1, controller)
		woc1.operate(ctx)
		assert.NotEmpty(t, woc1.wf.Status.StoredWorkflowSpec.Shutdown)
		assert.Equal(t, wfv1.ShutdownStrategyStop, woc1.wf.Status.StoredWorkflowSpec.Shutdown)
		for _, node := range woc1.wf.Status.Nodes {
			if assert.NotNil(t, node) {
				assert.Contains(t, node.Message, "workflow shutdown with strategy")
				assert.Contains(t, node.Message, "Stop")
			}
		}
	})
}

var suspendwf = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: workflow-template-whalesay-template-z56dm
  namespace: default
spec:
  arguments:
    parameters:
    - name: message
      value: tt
  entrypoint: whalesay-template
  suspend: true
  workflowTemplateRef:
    name: workflow-template-whalesay-template
status:
  artifactRepositoryRef:
    default: true
  conditions:
  - status: "False"
    type: PodRunning
  finishedAt: null
  phase: Running
  progress: 0/0
  startedAt: "2021-05-13T22:56:17Z"
  storedTemplates:
    namespaced/workflow-template-whalesay-template/whalesay-template:
      container:
        args:
        - sleep
        command:
        - cowsay
        image: docker/whalesay
        name: ""
      name: whalesay-template
  storedWorkflowTemplateSpec:
    entrypoint: whalesay-template
    suspend: true
    templates:
    - container:
        args:
        - sleep
        command:
        - cowsay
        image: docker/whalesay
        name: ""
      name: whalesay-template
    volumes:
    - emptyDir: {}
      name: data
    workflowTemplateRef:
      name: workflow-template-whalesay-template
`

func TestSuspendResumeWorkflowTemplateRef(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(suspendwf)
	cancel, controller := newController(wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
	defer cancel()
	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.True(t, *woc.wf.Status.StoredWorkflowSpec.Suspend)
	woc.wf.Spec.Suspend = nil
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	assert.Nil(t, woc.wf.Status.StoredWorkflowSpec.Suspend)
}

const wfTmplUpt = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-whalesay-template
  namespace: default
spec:
  templates:
  - name: hello-hello-hello
    steps:
    - - name: hello1
        template: whalesay
        arguments:
          parameters: [{name: message, value: "hello1"}]
    - - name: hello2a
        template: whalesay
        arguments:
          parameters: [{name: message, value: "hello2a"}]
      - name: hello2b
        template: whalesay
        arguments:
          parameters: [{name: message, value: "hello2b"}]

  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

func TestWorkflowTemplateUpdateScenario(t *testing.T) {

	wf := wfv1.MustUnmarshalWorkflow(wfWithTmplRef)
	cancel, controller := newController(wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
	defer cancel()
	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.NotEmpty(t, woc.wf.Status.StoredWorkflowSpec)
	assert.NotEmpty(t, woc.wf.Status.StoredWorkflowSpec.Templates[0].Container)

	cancel, controller = newController(woc.wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmplUpt))
	defer cancel()
	ctx = context.Background()
	woc1 := newWorkflowOperationCtx(woc.wf, controller)
	woc1.operate(ctx)
	assert.NotEmpty(t, woc1.wf.Status.StoredWorkflowSpec)
	assert.Equal(t, woc.wf.Status.StoredWorkflowSpec, woc1.wf.Status.StoredWorkflowSpec)
}

const wfTmplWithVol = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-whalesay-template-with-volume
  namespace: default
spec:
  volumeClaimTemplates:
  - metadata:
      name: workdir
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 1Gi
  entrypoint: whalesay-template
  templates:
  - name: whalesay-template
    container:
      image: docker/whalesay
      command: [cowsay]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol
`

func TestWFTWithVol(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfTmplWithVol)
	cancel, controller := newController(wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
	defer cancel()
	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pvc, err := controller.kubeclientset.CoreV1().PersistentVolumeClaims("default").List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, pvc.Items, 1)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	pvc, err = controller.kubeclientset.CoreV1().PersistentVolumeClaims("default").List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Len(t, pvc.Items, 0)
}

const wfTmp = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: cluster-workflow-template-hello-world-
spec:
  entrypoint: whalesay-template
  arguments:
    parameters:
      - name: message
        value: "hello world"
  workflowTemplateRef:
    name: cluster-workflow-template-whalesay-template
    clusterScope: true
`

func TestSubmitWorkflowTemplateRefWithoutRBAC(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfTmp)
	cancel, controller := newController(wf, wfv1.MustUnmarshalWorkflowTemplate(wfTmpl))
	defer cancel()
	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.controller.cwftmplInformer = nil
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
}
