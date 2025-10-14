package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

var wfDefaults = `
  metadata: 
    annotations: 
      testAnnotation: test
    labels: 
      testLabel: test
  spec: 
    entrypoint: whalesay
    activeDeadlineSeconds: 7200
    arguments:
      parameters: 
        - 
          name: message
          value: "hello world"
    onExit: whalesay-exit
    serviceAccountName: default
    templates: 
      - 
        container: 
          args: 
            - "hello from the default exit handler"
          command: 
            - cowsay
          image: docker/whalesay
        name: whalesay-exit
    ttlStrategy: 
      secondsAfterCompletion: 60
    volumes: 
      - 
        name: test
        secret: 
          secretName: test
`

var simpleWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  labels:
    workflows.argoproj.io/archive-strategy: "false"
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var wfDefaultResult = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata: 
  annotations: 
    testAnnotation: test
  generateName: hello-world-
  labels: 
    testLabel: test
    workflows.argoproj.io/archive-strategy: "false"
spec: 
  activeDeadlineSeconds: 7200
  arguments:
    parameters: 
      - 
        name: message
        value: "hello world"
  entrypoint: whalesay
  onExit: whalesay-exit
  serviceAccountName: default
  templates: 
    - 
      container: 
        args: 
          - "hello world"
        command: 
          - cowsay
        image: "docker/whalesay:latest"
      name: whalesay
    - 
      container: 
        args: 
          - "hello from the default exit handler"
        command: 
          - cowsay
        image: docker/whalesay
      name: whalesay-exit
  ttlStrategy: 
    secondsAfterCompletion: 60
  volumes: 
    - 
      name: test
      secret: 
        secretName: test
`

var simpleWFT = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-submittable
  namespace: default
spec:
  entrypoint: whalesay-template
  arguments:
    parameters:
      - name: message
        value: hello world
  templates:
    - name: whalesay-template
      inputs:
        parameters:
          - name: message
      container:
        image: docker/whalesay
        command: [cowsay]
        args: ["{{inputs.parameters.message}}"]
`

var storedSpecResult = `
{
   "activeDeadlineSeconds": 7200,
   "arguments": {
      "parameters": [
         {
            "name": "message",
            "value": "hello world"
         }
      ]
   },
   "entrypoint": "whalesay-template",
   "onExit": "whalesay-exit",
   "serviceAccountName": "default",
   "templates": [
      {
         "container": {
            "args": [
               "{{inputs.parameters.message}}"
            ],
            "command": [
               "cowsay"
            ],
            "image": "docker/whalesay"
            
         },
		"inputs": {
				"parameters": [
				   {
					  "name": "message"
				   }
				]
			},
         "name": "whalesay-template"
      },
      {
         "container": {
            "args": [
               "hello from the default exit handler"
            ],
            "command": [
               "cowsay"
            ],
            "image": "docker/whalesay"
         },
         "name": "whalesay-exit"
      }
   ],
   "ttlStrategy": {
      "secondsAfterCompletion": 60
   },
   "volumes": [
      {
         "name": "test",
         "secret": {
            "secretName": "test"
         }
      }
   ]
}
`

func TestWFDefaultsWithWorkflow(t *testing.T) {
	assert := assert.New(t)

	wfDefault := wfv1.MustUnmarshalWorkflow(wfDefaults)
	wf := wfv1.MustUnmarshalWorkflow(simpleWf)
	wf1 := wf.DeepCopy()
	wfResult := wfv1.MustUnmarshalWorkflow(wfDefaultResult)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newControllerWithDefaults(ctx)
	defer cancel()

	controller.Config.WorkflowDefaults = wfDefault
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(woc.wf.Spec, wfResult.Spec)
	assert.Contains(woc.wf.Labels, "testLabel")
	assert.Contains(woc.wf.Annotations, "testAnnotation")

	wf1.Spec.Entrypoint = ""
	woc = newWorkflowOperationCtx(ctx, wf1, controller)
	woc.operate(ctx)
	assert.Equal(woc.wf.Spec, wfResult.Spec)
	assert.Contains(woc.wf.Labels, "testLabel")
	assert.Contains(woc.wf.Annotations, "testAnnotation")
}

func TestWFDefaultWithWFTAndWf(t *testing.T) {
	assert := assert.New(t)
	wfDefault := wfv1.MustUnmarshalWorkflow(wfDefaults)
	wft := wfv1.MustUnmarshalWorkflowTemplate(simpleWFT)
	var resultSpec wfv1.WorkflowSpec
	wfv1.MustUnmarshal([]byte(storedSpecResult), &resultSpec)

	t.Run("SubmitSimpleWorkflowRef", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wft)
		defer cancel()
		controller.Config.WorkflowDefaults = wfDefault

		wf := wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}, Spec: wfv1.WorkflowSpec{WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"}}}
		woc := newWorkflowOperationCtx(ctx, &wf, controller)
		woc.operate(ctx)
		resultSpec.WorkflowTemplateRef = &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"}
		assert.Equal(resultSpec, woc.execWf.Spec)
		assert.Equal(&resultSpec, woc.wf.Status.StoredWorkflowSpec)
	})

	t.Run("SubmitComplexWorkflowRef", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wft)
		defer cancel()
		controller.Config.WorkflowDefaults = wfDefault

		ttlStrategy := wfv1.TTLStrategy{
			SecondsAfterCompletion: ptr.To(int32(10)),
		}

		wf := wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
			Spec: wfv1.WorkflowSpec{
				WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"},
				Entrypoint:          "Test",
				TTLStrategy:         &ttlStrategy,
			},
		}
		resultSpec.Entrypoint = "Test"
		resultSpec.TTLStrategy = &ttlStrategy
		resultSpec.WorkflowTemplateRef = &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"}

		woc := newWorkflowOperationCtx(ctx, &wf, controller)
		woc.operate(ctx)
		assert.Equal(resultSpec, woc.execWf.Spec)
		assert.Equal(&resultSpec, woc.wf.Status.StoredWorkflowSpec)
	})

	t.Run("SubmitComplexWorkflowRefWithArguments", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wft)
		defer cancel()
		controller.Config.WorkflowDefaults = wfDefault

		param := wfv1.Parameter{
			Name:  "Test",
			Value: wfv1.AnyStringPtr("welcome"),
		}
		art := wfv1.Artifact{
			Name: "TestA",
			Path: "tmp/test",
		}

		ttlStrategy := wfv1.TTLStrategy{
			SecondsAfterCompletion: ptr.To(int32(10)),
		}

		wf := wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
			Spec: wfv1.WorkflowSpec{
				WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"},
				Entrypoint:          "Test",
				TTLStrategy:         &ttlStrategy,
				Arguments: wfv1.Arguments{
					Parameters: []wfv1.Parameter{param},
					Artifacts:  wfv1.Artifacts{art},
				},
			},
		}
		// resultSpec.Arguments.Parameters = append(resultSpec.Arguments.Parameters, args.Parameters...)
		resultSpec.Entrypoint = "Test"
		resultSpec.TTLStrategy = &ttlStrategy
		resultSpec.WorkflowTemplateRef = &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"}
		resultSpec.Arguments.Parameters = append(resultSpec.Arguments.Parameters, param)
		resultSpec.Arguments.Artifacts = append(resultSpec.Arguments.Artifacts, art)

		woc := newWorkflowOperationCtx(ctx, &wf, controller)
		woc.operate(ctx)
		assert.Contains(woc.execWf.Spec.Arguments.Parameters, param)
		assert.Contains(woc.wf.Status.StoredWorkflowSpec.Arguments.Artifacts, art)
	})
}
