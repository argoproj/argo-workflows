package controller

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
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
    serviceAccountName: argo
    templates: 
      - 
        container: 
          args: 
            - "hello from the default exit handler"
          command: 
            - cowsay
          image: docker/whalesay
        name: whalesay-exit
    ttlSecondsAfterFinished: 86400
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
var wf_wfdefaultResult = `
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
  serviceAccountName: argo
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
  ttlSecondsAfterFinished: 86400
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
   "serviceAccountName": "argo",
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
   "ttlSecondsAfterFinished": 86400,
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

	wfDefault := unmarshalWF(wfDefaults)
	wf := unmarshalWF(simpleWf)
	wf1 := wf.DeepCopy()
	wfResult := unmarshalWF(wf_wfdefaultResult)
	cancel, controller := newControllerWithDefaults()
	defer cancel()
	controller.Config.WorkflowDefaults = wfDefault
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.Equal(woc.wf.Spec, wfResult.Spec)
	assert.Contains(woc.wf.Labels, "testLabel")
	assert.Contains(woc.wf.Annotations, "testAnnotation")
	wf1.Spec.Entrypoint = ""
	woc = newWorkflowOperationCtx(wf1, controller)
	woc.operate()
	assert.Equal(woc.wf.Spec, wfResult.Spec)
	assert.Contains(woc.wf.Labels, "testLabel")
	assert.Contains(woc.wf.Annotations, "testAnnotation")
}

func TestWFDefaultWithWFTAndWf(t *testing.T) {
	assert := assert.New(t)
	wfDefault := unmarshalWF(wfDefaults)
	wft := unmarshalWFTmpl(simpleWFT)
	var resultSpec wfv1.WorkflowSpec
	err := json.Unmarshal([]byte(storedSpecResult), &resultSpec)
	assert.NoError(err)
	cancel, controller := newControllerWithDefaults()
	defer cancel()
	controller.Config.WorkflowDefaults = wfDefault
	_, err = controller.wfclientset.ArgoprojV1alpha1().WorkflowTemplates("default").Create(wft)
	assert.NoError(err)
	t.Run("SubmitSimpleWorkflowRef", func(t *testing.T) {
		wf := wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}, Spec: wfv1.WorkflowSpec{WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"}}}
		woc := newWorkflowOperationCtx(&wf, controller)
		woc.operate()
		resultSpec.WorkflowTemplateRef = &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"}
		assert.Equal(resultSpec, woc.execWf.Spec)
		assert.Equal(&resultSpec, woc.wf.Status.StoredWorkflowSpec)
	})

	t.Run("SubmitComplexWorkflowRef", func(t *testing.T) {
		ttlStrategy := wfv1.TTLStrategy{
			SecondsAfterCompletion: pointer.Int32Ptr(10),
		}

		wf := wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
			Spec: wfv1.WorkflowSpec{
				WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"},
				Entrypoint:          "Test",
				TTLStrategy:         &ttlStrategy,
			},
		}
		resultSpec.Entrypoint = "Test"
		resultSpec.TTLStrategy = &ttlStrategy
		resultSpec.WorkflowTemplateRef = &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"}

		woc := newWorkflowOperationCtx(&wf, controller)
		woc.operate()
		assert.Equal(resultSpec, woc.execWf.Spec)
		assert.Equal(&resultSpec, woc.wf.Status.StoredWorkflowSpec)
	})

	t.Run("SubmitComplexWorkflowRefWithArguments", func(t *testing.T) {
		param := wfv1.Parameter{
			Name:  "Test",
			Value: wfv1.AnyStringPtr("welcome"),
		}
		art := wfv1.Artifact{
			Name: "TestA",
			Path: "tmp/test",
		}

		ttlStrategy := wfv1.TTLStrategy{
			SecondsAfterCompletion: pointer.Int32Ptr(10),
		}

		wf := wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
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
		//resultSpec.Arguments.Parameters = append(resultSpec.Arguments.Parameters, args.Parameters...)
		resultSpec.Entrypoint = "Test"
		resultSpec.TTLStrategy = &ttlStrategy
		resultSpec.WorkflowTemplateRef = &wfv1.WorkflowTemplateRef{Name: "workflow-template-submittable"}
		resultSpec.Arguments.Parameters = append(resultSpec.Arguments.Parameters, param)
		resultSpec.Arguments.Artifacts = append(resultSpec.Arguments.Artifacts, art)

		woc := newWorkflowOperationCtx(&wf, controller)
		woc.operate()
		assert.Contains(woc.execWf.Spec.Arguments.Parameters, param)
		assert.Contains(woc.wf.Status.StoredWorkflowSpec.Arguments.Artifacts, art)
	})

}
