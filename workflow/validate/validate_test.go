package validate

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

var (
	wfClientset   = fakewfclientset.NewSimpleClientset()
	wftmplGetter  = templateresolution.WrapWorkflowTemplateInterface(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault))
	cwftmplGetter = templateresolution.WrapClusterWorkflowTemplateInterface(wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates())
)

func createWorkflowTemplate(yamlStr string) error {
	ctx := context.Background()
	wftmpl := unmarshalWftmpl(yamlStr)
	_, err := wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault).Create(ctx, wftmpl, metav1.CreateOptions{})
	if err != nil && apierr.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// validate is a test helper to accept Workflow YAML as a string and return
// its validation result.
func validate(yamlStr string) (*wfv1.Conditions, error) {
	wf := unmarshalWf(yamlStr)
	return ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})
}

func validateWithOptions(yamlStr string, opts ValidateOpts) (*wfv1.Conditions, error) {
	wf := unmarshalWf(yamlStr)
	return ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, opts)
}

// validateWorkflowTemplate is a test helper to accept WorkflowTemplate YAML as a string and return
// its validation result.
func validateWorkflowTemplate(yamlStr string, opts ValidateOpts) error {
	wftmpl := unmarshalWftmpl(yamlStr)
	_, err := ValidateWorkflowTemplate(wftmplGetter, cwftmplGetter, wftmpl, opts)
	return err
}

func unmarshalWf(yamlStr string) *wfv1.Workflow {
	var wf wfv1.Workflow
	wfv1.MustUnmarshal([]byte(yamlStr), &wf)
	return &wf
}

func unmarshalWftmpl(yamlStr string) *wfv1.WorkflowTemplate {
	var wftmpl wfv1.WorkflowTemplate
	wfv1.MustUnmarshal([]byte(yamlStr), &wftmpl)
	return &wftmpl
}

const invalidErr = "is invalid"

var dupTemplateNames = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
  - name: whalesay
    container:
      image: docker/whalesay:latest
`

var dupInputNames = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: dup
        value: "value"
      - name: dup
        value: "value"
    container:
      image: docker/whalesay:latest
`

var emptyName = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: ""
        value: "value"
    container:
      image: docker/whalesay:latest
`

func TestDuplicateOrEmptyNames(t *testing.T) {
	_, err := validate(dupTemplateNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "not unique")
	}
	_, err = validate(dupInputNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "not unique")
	}
	_, err = validate(emptyName)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "name is required")
	}
}

var unresolvedInput = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:{{inputs.parameters.unresolved}}
`

var unresolvedStepInput = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: entry-step
  arguments:
    parameters: []
  templates:
    - steps:
        - - name: a
            arguments:
              parameters:
                - name: message
                  value: "{{inputs.parameters.message}}"
            template: whalesay
      name: entry-step
      inputs:
        parameters:
          - name: message
            value: hello world
    - name: whalesay
      container:
        image: docker/whalesay
        command: [cowsay]
        args: ["{{inputs.parameters.message}}"]
`

var unresolvedOutput = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: unresolved-output-steps
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
  - name: unresolved-output-steps
    steps:
    - - name: whalesay
        template: whalesay
    outputs:
      parameters:
      - name: unresolved
        valueFrom:
          parameter: "{{steps.whalesay.outputs.parameters.unresolved}}"
`

func TestUnresolved(t *testing.T) {
	_, err := validate(unresolvedInput)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "failed to resolve")
	}
	_, err = validate(unresolvedStepInput)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "failed to resolve")
	}
	_, err = validate(unresolvedOutput)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "failed to resolve")
	}
}

var ioArtifactPaths = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: artifact-path-placeholders-
spec:
  entrypoint: head-lines
  arguments:
    parameters:
    - name: lines-count
      value: 3
    artifacts:
    - name: text
      raw:
        data: |
          1
          2
          3
          4
          5
  templates:
  - name: head-lines
    inputs:
      parameters:
      - name: lines-count
      artifacts:
      - name: text
        path: /inputs/text/data
    outputs:
      parameters:
      - name: actual-lines-count
        valueFrom:
          path: /outputs/actual-lines-count/data
      artifacts:
      - name: text
        path: /outputs/text/data
    container:
      image: busybox
      command: [sh, -c, 'head -n {{inputs.parameters.lines-count}} <"{{inputs.artifacts.text.path}}" | tee "{{outputs.artifacts.text.path}}" | wc -l > "{{outputs.parameters.actual-lines-count.path}}"']
`

func TestResolveIOArtifactPathPlaceholders(t *testing.T) {
	_, err := validate(ioArtifactPaths)
	assert.NoError(t, err)
}

var outputParameterPath = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: get-current-date-
spec:
  entrypoint: get-current-date
  templates:
  - name: get-current-date
    outputs:
      parameters:
      - name: current-date
        valueFrom:
          path: /tmp/current-date
    container:
      image: busybox
      command: [sh, -c, 'date > {{outputs.parameters.current-date.path}}']
`

func TestResolveOutputParameterPathPlaceholder(t *testing.T) {
	_, err := validate(outputParameterPath)
	assert.NoError(t, err)
}

var stepOutputReferences = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: step-output-ref-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
        value: "value"
    container:
      image: docker/whalesay:latest
    outputs:
      parameters:
      - name: outparam
        valueFrom:
          path: /etc/hosts
  - name: stepref
    steps:
    - - name: one
        template: whalesay
    - - name: two
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "{{steps.one.outputs.parameters.outparam}}"
`

func TestStepOutputReference(t *testing.T) {
	_, err := validate(stepOutputReferences)
	assert.NoError(t, err)
}

var stepStatusReferences = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: status-ref-
spec:
  entrypoint: statusref
  templates:
  - name: statusref
    steps:
    - - name: one
        template: say
        arguments:
          parameters:
          - name: message
            value: "Hello, world"
    - - name: two
        template: say
        arguments:
          parameters:
          - name: message
            value: "{{steps.one.status}}"
  - name: say
    inputs:
      parameters:
      - name: message
        value: "value"
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo {{inputs.parameters.message}}"]
`

func TestStepStatusReference(t *testing.T) {
	_, err := validate(stepStatusReferences)
	assert.NoError(t, err)
}

var stepStatusReferencesNoFutureReference = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: status-ref-
spec:
  entrypoint: statusref
  templates:
  - name: statusref
    steps:
    - - name: one
        template: say
        arguments:
          parameters:
          - name: message
            value: "{{steps.two.status}}"
    - - name: two
        template: say
        arguments:
          parameters:
          - name: message
            value: "{{steps.one.status}}"
  - name: say
    inputs:
      parameters:
      - name: message
        value: "value"
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo {{inputs.parameters.message}}"]
`

func TestStepStatusReferenceNoFutureReference(t *testing.T) {
	_, err := validate(stepStatusReferencesNoFutureReference)
	// Can't reference the status of steps that have not run yet
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "failed to resolve {{steps.two.status}}")
	}
}

var stepArtReferences = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: step-art-ref-
spec:
  entrypoint: stepref
  templates:
  - name: generate
    container:
      image: alpine:3.7
      command: [echo, generate]
    outputs:
      artifacts:
      - name: generated_hosts
        path: /etc/hosts

  - name: echo
    inputs:
      parameters:
      - name: message
      artifacts:
      - name: passthrough
        path: /tmp/passthrough
    container:
      image: alpine:3.7
      command: [echo, "{{inputs.parameters.message}}"]
    outputs:
      parameters:
      - name: hosts
        valueFrom:
          path: /etc/hosts
      artifacts:
      - name: someoutput
        path: /tmp/passthrough

  - name: stepref
    steps:
    - - name: one
        template: generate
    - - name: two
        template: echo
        arguments:
          parameters:
          - name: message
            value: val
          artifacts:
          - name: passthrough
            from: "{{steps.one.outputs.artifacts.generated_hosts}}"
`

func TestStepArtReference(t *testing.T) {
	_, err := validate(stepArtReferences)
	assert.NoError(t, err)
}

var paramWithValueFromConfigMapRef = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
        valueFrom:
          configMapKeyRef:
            name: simple-config
            key: msg
    container:
      image: docker/whalesay:latest
`

func TestParamWithValueFromConfigMapRef(t *testing.T) {
	_, err := validate(paramWithValueFromConfigMapRef)
	assert.NoError(t, err)
}

var paramWithoutValue = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
`

func TestParamWithoutValue(t *testing.T) {
	_, err := validate(paramWithoutValue)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "not supplied")
	}
}

var globalParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: global-parameters-complex-
spec:
  priority: 100
  entrypoint: test-workflow
  arguments:
    parameters:
    - name: message1
      value: hello world
    - name: message2
      value: foo bar

  templates:
  - name: test-workflow
    inputs:
      parameters:
      - name: message1
      - name: message-internal
        value: "{{workflow.parameters.message1}}"
    steps:
    - - name: step1
        template: whalesay
        arguments:
          parameters:
          - name: message1
            value: world hello
          - name: message2
            value: "{{inputs.parameters.message1}}"
          - name: message3
            value: "{{workflow.parameters.message2}}"
          - name: message4
            value: "{{inputs.parameters.message-internal}}"


  - name: whalesay
    inputs:
      parameters:
      - name: message1
      - name: message2
      - name: message3
      - name: message4
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["Global 1: {{workflow.parameters.message1}} Input 1: {{inputs.parameters.message1}} Input 2/Steps Input 1/Global 1: {{inputs.parameters.message2}} Input 3/Global 2: {{inputs.parameters.message3}} Input4/Steps Input 2 internal/Global 1: {{inputs.parameters.message4}}"]
`

var unsuppliedArgValue = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: step-with-unsupplied-param-
spec:
  arguments:
    parameters:
    - name: missing
  entrypoint: whalesay
  templates:
  - 
    container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
`

var nestedGlobalParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: global-output
spec:
  entrypoint: global-output
  templates:
  - name: global-output
    steps:
    - - name: nested
        template: nested-level1
    - - name: consume-global
        template: consume-global
        arguments:
          artifacts:
          - name: art
            from: "{{workflow.outputs.artifacts.global-art}}"

  - name: nested-level1
    steps:
      - - name: nested
          template: nested-level2

  - name: nested-level2
    steps:
      - - name: nested
          template: output-global

  - name: output-global
    container:
      image: alpine:3.7
      command: [sh, -c]
      args: ["sleep 1; echo -n art > /tmp/art.txt; echo -n param > /tmp/param.txt"]
    outputs:
      artifacts:
      - name: hello-art
        path: /tmp/art.txt
        globalName: global-art

  - name: consume-global
    inputs:
      artifacts:
      - name: art
        path: /art
    container:
      image: alpine:3.7
      command: [sh, -c]
      args: ["cat /art"]
`

func TestGlobalParam(t *testing.T) {
	_, err := validate(globalParam)
	assert.NoError(t, err)

	_, err = validate(nestedGlobalParam)
	assert.NoError(t, err)

	_, err = validate(unsuppliedArgValue)
	assert.EqualError(t, err, "spec.arguments.missing.value is required")
}

var invalidTemplateNames = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay_d
  templates:
  - name: whalesay_d
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
`

func TestInvalidTemplateName(t *testing.T) {
	_, err := validate(invalidTemplateNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), invalidErr)
	}
}

var invalidArgParamNames = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  arguments:
    parameters:
    - name: param#1
      value: paramValue
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
`

func TestInvalidArgParamName(t *testing.T) {
	_, err := validate(invalidArgParamNames)
	assert.NotNil(t, err)
}

var invalidArgArtNames = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: arguments-artifacts-
spec:
  entrypoint: kubectl-input-artifact
  arguments:
    artifacts:
    - name: "&-kubectl"
      http:
        url: https://storage.googleapis.com/kubernetes-release/release/v1.8.0/bin/linux/amd64/kubectl

  templates:
  - name: kubectl-input-artifact
    inputs:
      artifacts:
      - name: "&-kubectl"
        path: /usr/local/bin/kubectl
        mode: 0755
    container:
      image: debian:9.4
      command: [sh, -c]
      args: ["kubectl version"]
`

func TestInvalidArgArtName(t *testing.T) {
	_, err := validate(invalidArgArtNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), invalidErr)
	}
}

var invalidStepNames = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-
spec:
  entrypoint: hello-hello-hello

  templates:
  - name: hello-hello-hello
    steps:
    - - name: hello1.blah
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello1"
    - - name: hello2a
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello2a"
      - name: hello2b
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello2b"

  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

func TestInvalidStepName(t *testing.T) {
	_, err := validate(invalidStepNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), invalidErr)
	}
}

var invalidInputParamNames = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message+123
        default: "abc"
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message+123}}"]
`

func TestInvalidInputParamName(t *testing.T) {
	_, err := validate(invalidInputParamNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), invalidErr)
	}
}

var invalidInputArtNames = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-
spec:
  entrypoint: hello-hello-hello

  templates:
  - name: hello-hello-hello
    steps:
    - - name: hello1
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello1"
    - - name: hello2a
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello2a"
      - name: hello2b
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello2b"

  - name: whalesay
    inputs:
      parameters:
      - name: message
      artifacts:
      - name: test(jpg
        path: /test.jpg
        http:
          url: https://commons.wikimedia.org/wiki/File:Example.jpg
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

func TestInvalidInputArtName(t *testing.T) {
	_, err := validate(invalidInputArtNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), invalidErr)
	}
}

var invalidOutputArtNames = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: output-artifact-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      artifacts:
      - name: "!1"
        path: /tmp/hello_world.txt
`

func TestInvalidOutputArtName(t *testing.T) {
	_, err := validate(invalidOutputArtNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), invalidErr)
	}
}

var invalidOutputParamNames = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: output-artifact-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      parameters:
      - name: blah-122lsfj}
        valueFrom:
          path: /tmp/hello_world.txt
`

var invalidOutputMissingValueFrom = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: output-param-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      parameters:
      - name: outparam
`

var invalidOutputMultipleValueFrom = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: output-param-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      parameters:
      - name: outparam
        valueFrom:
          path: /abc
          jqFilter: abc
`

var invalidOutputIncompatibleValueFromPath = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: output-param-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      parameters:
      - name: outparam
        valueFrom:
          parameter: abc
`

var invalidOutputIncompatibleValueFromParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: output-param-
spec:
  entrypoint: my-steps
  templates:
  - name: my-steps
    steps:
    - - name: step1
        template: whalesay
    outputs:
      parameters:
      - name: myoutput
        valueFrom:
          path: /abc
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      parameters:
      - name: outparam
        valueFrom:
          path: /abc
`

func TestInvalidOutputParam(t *testing.T) {
	_, err := validate(invalidOutputParamNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), invalidErr)
	}
	_, err = validate(invalidOutputMissingValueFrom)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "does not have valueFrom or value specified")
	}
	_, err = validate(invalidOutputMultipleValueFrom)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "multiple valueFrom")
	}
	_, err = validate(invalidOutputIncompatibleValueFromPath)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), ".path must be specified for Container templates")
	}
	_, err = validate(invalidOutputIncompatibleValueFromParam)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), ".parameter or expression must be specified for Steps templates")
	}
}

var multipleTemplateTypes = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: multiple-template-types-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        import random
        i = random.randint(1, 100)
        print(i)
`

func TestMultipleTemplateTypes(t *testing.T) {
	_, err := validate(multipleTemplateTypes)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "multiple template types specified")
	}
}

var exitHandlerWorkflowStatusOnExit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: exit-handlers-
spec:
  entrypoint: pass
  onExit: fail
  templates:
  - name: pass
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["exit 0"]
  - name: fail
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo {{workflow.status}} {{workflow.uid}} {{workflow.duration}}"]
`

var workflowStatusNotOnExit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: exit-handlers-
spec:
  entrypoint: pass
  templates:
  - name: pass
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo {{workflow.failures}}"]
`

func TestExitHandler(t *testing.T) {
	// ensure {{workflow.status}} is not available when not in exit handler
	_, err := validate(workflowStatusNotOnExit)
	assert.NotNil(t, err)

	// ensure {{workflow.status}} is available in exit handler
	_, err = validate(exitHandlerWorkflowStatusOnExit)
	assert.NoError(t, err)
}

var workflowWithPriority = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: with-priority-
spec:
  entrypoint: whalesay
  priority: 100
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["{{workflow.priority}}"]
`

func TestPriorityVariable(t *testing.T) {
	_, err := validate(workflowWithPriority)
	assert.NoError(t, err)
}

var volumeMountArtifactPathCollision = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: path-collision-
spec:
  volumeClaimTemplates:
  - metadata:
      name: workdir
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 1Gi
  entrypoint: pass
  templates:
  - name: pass
    inputs:
      artifacts:
      - name: argo-source
        path: /src
        git:
          repo: https://github.com/argoproj/argo-workflows.git
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["exit 0"]
      volumeMounts:
      - name: workdir
        mountPath: /src
`

func TestVolumeMountArtifactPathCollision(t *testing.T) {
	// ensure we detect and reject path collisions
	wf := unmarshalWf(volumeMountArtifactPathCollision)

	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})

	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "already mounted")
	}
	// tweak the mount path and validation should now be successful
	wf.Spec.Templates[0].Container.VolumeMounts[0].MountPath = "/differentpath"

	_, err = ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})

	assert.NoError(t, err)
}

var activeDeadlineSeconds = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: active-deadline-seconds-
spec:
  entrypoint: pass
  templates:
  - name: pass
    activeDeadlineSeconds: -1
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["exit 0"]
`

func TestValidActiveDeadlineSeconds(t *testing.T) {
	_, err := validate(activeDeadlineSeconds)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "activeDeadlineSeconds must be a positive integer > 0")
	}
}

var leafWithParallelism = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: leaf-with-parallelism
spec:
  entrypoint: leaf-with-parallelism
  templates:
  - name: leaf-with-parallelism
    parallelism: 2
    container:
      image: debian:9.4
      command: [sh, -c]
      args: ["kubectl version"]
`

func TestLeafWithParallelism(t *testing.T) {
	_, err := validate(leafWithParallelism)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "is only valid")
	}
}

var invalidStepsArgumentNoFromOrLocation = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: output-artifact-
spec:
  entrypoint: no-location-or-from
  templates:
  - name: no-location-or-from
    steps:
    - - name: whalesay
        template: whalesay
        arguments:
          artifacts:
          - name: art

  - name: whalesay
    inputs:
      artifacts:
      - name: art
        path: /tmp/art
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world"]
`

var invalidDAGArgumentNoFromOrLocation = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: output-artifact-
spec:
  entrypoint: no-location-or-from
  templates:
  - name: no-location-or-from
    dag:
      tasks:
      - name: whalesay
        template: whalesay
        arguments:
          artifacts:
          - name: art

  - name: whalesay
    inputs:
      artifacts:
      - name: art
        path: /tmp/art
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world"]
`

func TestInvalidArgumentNoFromOrLocation(t *testing.T) {
	_, err := validate(invalidStepsArgumentNoFromOrLocation)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "from, artifact location, or key is required")
	}
	_, err = validate(invalidDAGArgumentNoFromOrLocation)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "from, artifact location, or key is required")
	}
}

var invalidArgumentNoValue = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: output-artifact-
spec:
  entrypoint: no-location-or-from
  templates:
  - name: no-location-or-from
    steps:
    - - name: whalesay
        template: whalesay
        arguments:
          parameters:
          - name: art

  - name: whalesay
    inputs:
      parameters:
      - name: art
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world"]
`

func TestInvalidArgumentNoValue(t *testing.T) {
	_, err := validate(invalidArgumentNoValue)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), ".value is required")
	}
}

var validWithItems = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: loops-
spec:
  entrypoint: loop-example
  templates:
  - name: loop-example
    steps:
    - - name: print-message
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "{{item}}"
        withItems:
        - 0
        - false
        - string
        - 1.2
        - os: "debian"
          version: "9.0"

  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

func TestValidWithItems(t *testing.T) {
	_, err := validate(validWithItems)
	assert.NoError(t, err)
}

var podNameVariable = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: pod-name-variable
spec:
  entrypoint: pod-name-variable
  templates:
  - name: pod-name-variable
    container:
      image: debian:9.4
      command: [sh, -c]
      args: ["kubectl {{pod.name}}"]
    outputs:
      artifacts:
      - name: my-out
        path: /tmp/hello_world.txt
        s3:
          endpoint: s3.amazonaws.com
          bucket: my-bucket
          key: path/{{pod.name}}/hello_world.tgz
          accessKeySecret:
            name: my-s3-credentials
            key: accessKey
          secretKeySecret:
            name: my-s3-credentials
            key: secretKey
`

func TestPodNameVariable(t *testing.T) {
	_, err := validate(podNameVariable)
	assert.NoError(t, err)
}

func TestGlobalParamWithVariable(t *testing.T) {
	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wfv1.MustUnmarshalWorkflow("@../../test/e2e/functional/global-outputs-variable.yaml"), ValidateOpts{})

	assert.NoError(t, err)
}

var specArgumentNoValue = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: spec-arg-no-value-
spec:
  entrypoint: whalesay
  arguments:
    parameters:
    - name: required-param
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
`

// TestSpecArgumentNoValue we allow parameters to have no value at the spec level during linting
func TestSpecArgumentNoValue(t *testing.T) {
	wf := unmarshalWf(specArgumentNoValue)

	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{Lint: true})
	assert.NoError(t, err)
	_, err = ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})

	assert.NotNil(t, err)
}

var specArgumentSnakeCase = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: spec-arg-snake-case-
spec:
  entrypoint: whalesay
  arguments:
    artifacts:
    - name: __kubectl
      http:
        url: https://storage.googleapis.com/kubernetes-release/release/v1.8.0/bin/linux/amd64/kubectl
    parameters:
    - name: my_snake_case_param
      value: "hello world"
  templates:
  - name: whalesay
    inputs:
      artifacts:
      - name: __kubectl
        path: /usr/local/bin/kubectl
        mode: 0755
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay {{workflow.parameters.my_snake_case_param}} | tee /tmp/hello_world.txt && ls  /usr/local/bin/kubectl"]
`

// TestSpecArgumentSnakeCase we allow parameter and artifact names to be snake case
func TestSpecArgumentSnakeCase(t *testing.T) {
	wf := unmarshalWf(specArgumentSnakeCase)

	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{Lint: true})

	assert.NoError(t, err)
}

var specBadSequenceCountAndEnd = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: loops-sequence-
spec:
  entrypoint: loops-sequence
  templates:
  - name: loops-sequence
    steps:
    - - name: print-num
        template: echo
        arguments:
          parameters:
          - name: num
            value: "{{item}}"
        withSequence:
          count: "10"
          end: "10"
  - name: echo
    inputs:
      parameters:
      - name: num
    container:
      image: alpine:latest
      command: [echo, "{{inputs.parameters.num}}"]
`

// TestSpecBadSequenceCountAndEnd verifies both count and end cannot be defined
func TestSpecBadSequenceCountAndEnd(t *testing.T) {
	wf := unmarshalWf(specBadSequenceCountAndEnd)

	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{Lint: true})

	assert.Error(t, err)
}

var customVariableInput = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:{{user.username}}
`

// TestCustomTemplatVariable verifies custom template variable
func TestCustomTemplatVariable(t *testing.T) {
	wf := unmarshalWf(customVariableInput)

	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{Lint: true})

	assert.Equal(t, err, nil)
}

var baseImageOutputArtifact = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: base-image-out-art-
spec:
  entrypoint: base-image-out-art
  templates:
  - name: base-image-out-art
    container:
      image: alpine:latest
      command: [echo, hello]
    outputs:
      artifacts:
      - name: tmp
        path: /tmp
`

var baseImageOutputParameter = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: base-image-out-art-
spec:
  entrypoint: base-image-out-art
  templates:
  - name: base-image-out-art
    container:
      image: alpine:latest
      command: [echo, hello]
    outputs:
      parameters:
      - name: tmp
        valueFrom:
          path: /tmp/file
`

var volumeMountOutputArtifact = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: base-image-out-art-
spec:
  entrypoint: base-image-out-art
  volumes:
  - name: workdir
    emptyDir: {}
  templates:
  - name: base-image-out-art
    container:
      image: alpine:latest
      command: [echo, hello]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol
    outputs:
      artifacts:
      - name: workdir
        path: /mnt/vol
`

var baseImageDirWithEmptyDirOutputArtifact = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: base-image-out-art-
spec:
  entrypoint: base-image-out-art
  volumes:
  - name: workdir
    emptyDir: {}
  templates:
  - name: base-image-out-art
    container:
      image: alpine:latest
      command: [echo, hello]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol
    outputs:
      artifacts:
      - name: workdir
        path: /mnt
`

// TestBaseImageOutputVerify verifies we error when we detect the condition when the container
// runtime executor doesn't support output artifacts from a base image layer, and fails validation
func TestBaseImageOutputVerify(t *testing.T) {
	wfBaseOutArt := unmarshalWf(baseImageOutputArtifact)
	wfBaseOutParam := unmarshalWf(baseImageOutputParameter)
	wfEmptyDirOutArt := unmarshalWf(volumeMountOutputArtifact)
	wfBaseWithEmptyDirOutArt := unmarshalWf(baseImageDirWithEmptyDirOutputArtifact)
	var err error

	for _, executor := range []string{common.ContainerRuntimeExecutorPNS, common.ContainerRuntimeExecutorEmissary, ""} {
		switch executor {
		case common.ContainerRuntimeExecutorPNS:
			_, err = ValidateWorkflow(wftmplGetter, cwftmplGetter, wfBaseOutArt, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.NoError(t, err)
			_, err = ValidateWorkflow(wftmplGetter, cwftmplGetter, wfBaseOutParam, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.NoError(t, err)
			_, err = ValidateWorkflow(wftmplGetter, cwftmplGetter, wfBaseWithEmptyDirOutArt, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.Error(t, err)
		case common.ContainerRuntimeExecutorEmissary, "":
			_, err = ValidateWorkflow(wftmplGetter, cwftmplGetter, wfBaseOutArt, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.NoError(t, err)
			_, err = ValidateWorkflow(wftmplGetter, cwftmplGetter, wfBaseOutParam, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.NoError(t, err)
			_, err = ValidateWorkflow(wftmplGetter, cwftmplGetter, wfBaseWithEmptyDirOutArt, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.NoError(t, err)
		}
		_, err = ValidateWorkflow(wftmplGetter, cwftmplGetter, wfEmptyDirOutArt, ValidateOpts{ContainerRuntimeExecutor: executor})
		assert.NoError(t, err)
	}
}

var templateRefTarget = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: template-ref-target
spec:
  templates:
  - name: A
    container:
      image: alpine:latest
      command: [echo, hello]
`

func TestWorkflowTemplate(t *testing.T) {
	err := validateWorkflowTemplate(templateRefTarget, ValidateOpts{})
	assert.NoError(t, err)
}

var templateRefNestedTarget = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: template-ref-nested-target
spec:
  templates:
  - name: A
    steps:
      - - name: call-A
          templateRef:
            name: template-ref-target
            template: A
`

var nestedTemplateRef = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: template-ref-
spec:
  entrypoint: A
  templates:
  - name: A
    steps:
      - - name: call-A
          templateRef:
            name: template-ref-target
            template: A
`

func TestNestedTemplateRef(t *testing.T) {
	err := createWorkflowTemplate(templateRefTarget)
	assert.NoError(t, err)
	err = createWorkflowTemplate(templateRefNestedTarget)
	assert.NoError(t, err)
	wfConditions, err := validate(nestedTemplateRef)
	assert.NoError(t, err)
	assert.Empty(t, wfConditions)
}

var undefinedTemplateRef = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: undefined-template-ref-
spec:
  entrypoint: A
  templates:
  - name: A
    steps:
      - - name: call-A
          templateRef:
            name: foo
            template: echo
`

func TestUndefinedTemplateRef(t *testing.T) {
	_, err := validate(undefinedTemplateRef)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "not found")
	}
}

var validResourceWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: valid-resource-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    resource:
      action: create
      manifest: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: whalesay-cm
`

// TestValidResourceWorkflow verifies a workflow of a valid resource.
func TestValidResourceWorkflow(t *testing.T) {
	wf := unmarshalWf(validResourceWorkflow)

	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})

	assert.Equal(t, err, nil)
}

var invalidResourceWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: invalid-resource-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    resource:
      action: apply
      manifest: |
        invalid-yaml-line
        kind: ConfigMap
        metadata:
          name: whalesay-cm
`

var invalidActionResourceWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: invalid-resource-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    resource:
      action: foo
      manifest: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: whalesay-cm
`

// TestInvalidResourceWorkflow verifies an error against a workflow of an invalid resource.
func TestInvalidResourceWorkflow(t *testing.T) {
	wf := unmarshalWf(invalidResourceWorkflow)
	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})
	assert.EqualError(t, err, "templates.whalesay.resource.manifest must be a valid yaml")

	wf = unmarshalWf(invalidActionResourceWorkflow)
	_, err = ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})
	assert.EqualError(t, err, "templates.whalesay.resource.action must be one of: get, create, apply, delete, replace, patch")
}

var invalidPodGC = `
metadata:
  generateName: pod-gc-strategy-unknown-
spec:
  podGC:
    strategy: Foo
  entrypoint: main
  templates:
  - name: main
    container:
      image: docker/whalesay
`

// TestIncorrectPodGCStrategy verifies pod gc strategy is correct.
func TestIncorrectPodGCStrategy(t *testing.T) {
	wf := unmarshalWf(invalidPodGC)
	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})
	assert.EqualError(t, err, "podGC.strategy unknown strategy 'Foo'")
}

func TestInvalidPodGCLabelSelector(t *testing.T) {
	wf := unmarshalWf(`
metadata:
  generateName: pod-gc-strategy-unknown-
spec:
  podGC:
    labelSelector:
      matchExpressions:
        - {key: environment, operator: InvalidOperator, values: [dev]}
  entrypoint: main
  templates:
  - name: main
    container:
      image: docker/whalesay
`)
	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})
	assert.EqualError(t, err, "podGC.labelSelector invalid: \"InvalidOperator\" is not a valid pod selector operator")
}

var allowPlaceholderInVariableTakenFromInputs = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: argo-datadog-agent-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: kube-state-metrics-deployment
      value: |
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: kube-state-metrics
          namespace: kube-system
        spec:
          selector:
            matchLabels:
              k8s-app: kube-state-metrics
          replicas: 1
          template:
            metadata:
              labels:
                k8s-app: kube-state-metrics
            spec:
              serviceAccountName: kube-state-metrics
              containers:
              - name: kube-state-metrics
                image: quay.io/coreos/kube-state-metrics:v1.3.1
                ports:
                - name: http-metrics
                  containerPort: 8080
                - name: telemetry
                  containerPort: 8081
                readinessProbe:
                  httpGet:
                    path: /healthz
                    port: 8080
                  initialDelaySeconds: 5
                  timeoutSeconds: 5

    - name: kube-state-metrics-service
      value: |
        apiVersion: v1
        kind: Service
        metadata:
          name: kube-state-metrics
          namespace: kube-system
          labels:
            k8s-app: kube-state-metrics
          annotations:
            prometheus.io/scrape: 'true'
        spec:
          ports:
          - name: http-metrics
            port: 8080
            targetPort: http-metrics
            protocol: TCP
          - name: telemetry
            port: 8081
            targetPort: telemetry
            protocol: TCP
          selector:
            k8s-app: kube-state-metrics

  templates:
  - name: manifest
    inputs:
      parameters:
      - name: action
      - name: manifest
    resource:
      action: "{{inputs.parameters.action}}"
      manifest: "{{inputs.parameters.manifest}}"

  - name: main
    inputs:
      parameters:
      - name: kube-state-metrics-deployment
      - name: kube-state-metrics-service
    steps:
    - - name: kube-state-metrics-setup
        template: manifest
        arguments:
          parameters:
          - name: action
            value: "apply"
          - name: manifest
            value: "{{item}}"
        withItems:
        - "{{inputs.parameters.kube-state-metrics-deployment}}"
        - "{{inputs.parameters.kube-state-metrics-service}}"
`

func TestAllowPlaceholderInVariableTakenFromInputs(t *testing.T) {
	{
		wf := unmarshalWf(allowPlaceholderInVariableTakenFromInputs)
		_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})

		assert.NoError(t, err)
	}
}

var runtimeResolutionOfVariableNames = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: global-parameter-passing-
spec:
  entrypoint: plan
  templates:
  - name: plan
    steps:
    - - name: pass-parameter
        template: global-parameter-passing
        arguments:
          parameters:
          - name: global-parameter-name
            value: key
          - name: global-parameter-value
            value: value
    - - name: print-parameter
        template: parameter-printing
        arguments:
          parameters:
          - name: parameter
            value: "{{workflow.outputs.parameters.key}}"

  - name: global-parameter-passing
    inputs:
      parameters:
      - name: global-parameter-name
      - name: global-parameter-value
    container:
      image: alpine:3.11
      command: [sh, -c]
      args: ["exit 0"]
    outputs:
      parameters:
      - name: global-parameter
        value: "{{inputs.parameters.global-parameter-value}}"
        globalName: "{{inputs.parameters.global-parameter-name}}"

  - name: parameter-printing
    inputs:
      parameters:
      - name: parameter
    container:
      image: alpine:3.11
      command: [sh, -c]
      args: ["echo {{inputs.parameters.parameter}}"]
`

// TestInvalidResourceWorkflow verifies an error against a workflow of an invalid resource.
func TestRuntimeResolutionOfVariableNames(t *testing.T) {
	wf := unmarshalWf(runtimeResolutionOfVariableNames)
	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})

	assert.NoError(t, err)
}

var stepWithItemParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: loops-maps-
spec:
  entrypoint: loop-map-example
  templates:
    - name: loop-map-example
      steps:
        - - name: hello-world
            template: whalesay
          - name: test-linux
            template: cat-os-release
            arguments:
              parameters:
                - name: image
                  value: "{{item.image}}"
                - name: tag
                  value: "{{item.tag}}"
            withItems:
              - { image: "debian", tag: "9.1" }
              - { image: "debian", tag: "8.9" }
              - { image: "alpine", tag: "3.6" }
              - { image: "ubuntu", tag: "17.10" }

    - name: cat-os-release
      inputs:
        parameters:
          - name: image
          - name: tag
      container:
        image: "{{inputs.parameters.image}}:{{inputs.parameters.tag}}"
        command: [cat]
        args: [/etc/os-release]

    - name: whalesay
      container:
        image: docker/whalesay:latest
        command: [cowsay]
        args: ["hello world"]
`

func TestStepWithItemParam(t *testing.T) {
	_, err := validate(stepWithItemParam)
	assert.NoError(t, err)
}

var invalidMetricName = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    metrics:
      prometheus:
        - name: invalid.metric.name
          help: "invalid"
          gauge:
            value: 1
    container:
      image: docker/whalesay:latest
`

func TestInvalidMetricName(t *testing.T) {
	_, err := validate(invalidMetricName)
	assert.EqualError(t, err, "templates.whalesay metric name 'invalid.metric.name' is invalid. Metric names must contain alphanumeric characters, '_', or ':'")
}

var invalidMetricLabelName = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    metrics:
      prometheus:
        - name: valid
          help: "invalid"
          labels:
            - key: invalid.key
              value: hi
          gauge:
            value: 1
    container:
      image: docker/whalesay:latest
`

func TestInvalidMetricLabelName(t *testing.T) {
	_, err := validate(invalidMetricLabelName)
	assert.EqualError(t, err, "metric label 'invalid.key' is invalid: keys may only contain alphanumeric characters, '_', or ':'")
}

var invalidMetricHelp = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    metrics:
      prometheus:
        - name: metric_name
          gauge:
            value: 1
    container:
      image: docker/whalesay:latest
`

func TestInvalidMetricHelp(t *testing.T) {
	_, err := validate(invalidMetricHelp)
	assert.EqualError(t, err, "templates.whalesay metric 'metric_name' must contain a help string under 'help: ' field")
}

var invalidRealtimeMetricGauge = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    metrics:
      prometheus:
        - name: metric_name
          help: please
          gauge:
            realtime: true
            value: "{{resourcesDuration.cpu}}/{{resourcesDuration.memory}}"
    container:
      image: docker/whalesay:latest
`

func TestInvalidMetricGauge(t *testing.T) {
	_, err := validate(invalidRealtimeMetricGauge)
	assert.EqualError(t, err, "templates.whalesay metric 'metric_name' error: 'resourcesDuration.*' metrics cannot be used in real-time")
}

var invalidNoValueMetricGauge = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    metrics:
      prometheus:
        - name: metric_name
          help: please
          gauge:
            realtime: false
    container:
      image: docker/whalesay:latest
`

func TestInvalidNoValueMetricGauge(t *testing.T) {
	_, err := validate(invalidNoValueMetricGauge)
	assert.EqualError(t, err, "templates.whalesay metric 'metric_name' error: missing gauge.value")
}

var validMetricGauges = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    metrics:
      prometheus:
        - name: metric_one
          help: please
          gauge:
            realtime: true
            value: "{{duration}}/{{workflow.duration}}"
        - name: metric_two
          help: please
          gauge:
            realtime: false
            value: "{{resourcesDuration.cpu}}/{{resourcesDuration.memory}}/{{duration}}/{{workflow.duration}}"
        - name: metric_three
          help: please
          gauge:
            value: "{{resourcesDuration.cpu}}/{{resourcesDuration.memory}}/{{duration}}/{{workflow.duration}}"
    container:
      image: docker/whalesay:latest
`

func TestValidMetricGauge(t *testing.T) {
	_, err := validate(validMetricGauges)
	assert.NoError(t, err)
}

var globalVariables = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: global-variables-
spec:
  priority: 100
  entrypoint: test-workflow

  templates:
  - name: test-workflow
    steps:
    - - name: step1
        template: whalesay
        arguments:
          parameters:
          - name: name
            value: "{{workflow.name}}"
          - name: namespace
            value: "{{workflow.namespace}}"
          - name: serviceAccountName
            value: "{{workflow.serviceAccountName}}"
          - name: uid
            value: "{{workflow.uid}}"
          - name: priority
            value: "{{workflow.priority}}"
          - name: duration
            value: "{{workflow.duration}}"

  - name: whalesay
    inputs:
      parameters:
      - name: name
      - name: namespace
      - name: serviceAccountName
      - name: uid
      - name: priority
      - name: duration
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["name: {{inputs.parameters.name}} namespace: {{inputs.parameters.namespace}} serviceAccountName: {{inputs.parameters.serviceAccountName}} uid: {{inputs.parameters.uid}} priority: {{inputs.parameters.priority}} duration: {{inputs.parameters.duration}}"]
`

func TestWorkflowGlobalVariables(t *testing.T) {
	_, err := validate(globalVariables)
	assert.NoError(t, err)
}

var wfTemplateWithEntrypoint = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: template-with-entrypoint
spec:
  entrypoint: whalesay-template
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

func TestWorkflowTemplateWithEntrypoint(t *testing.T) {
	err := validateWorkflowTemplate(wfTemplateWithEntrypoint, ValidateOpts{})
	assert.NoError(t, err)
}

var wfWithWFTRefNoEntrypoint = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  namespace: default
spec:
  workflowTemplateRef:
    name: template-ref-with-entrypoint
`

var templateWithEntrypoint = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: template-ref-with-entrypoint
  namespace: default
spec:
  entrypoint: A
  templates:
  - name: A
    container:
      image: alpine:latest
      command: [echo, hello]
`

func TestWorkflowWithWFTRefWithEntrypoint(t *testing.T) {
	err := createWorkflowTemplate(templateWithEntrypoint)
	assert.NoError(t, err)
	_, err = validate(wfWithWFTRefNoEntrypoint)
	assert.NoError(t, err)
}

const wfWithWFTRef = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: A
  serviceAccountName: argo
  parallelism: 1
  volumes:
  - name: workdir
    emptyDir: {}
  podGC:
    strategy: OnPodSuccess
  nodeSelector:
    beta.kubernetes.io/arch: "{{inputs.parameters.arch}}"
  arguments:
    parameters:
    - name: lines-count
      value: 3
  workflowTemplateRef:
    name: template-ref-target
`

func TestWorkflowWithWFTRef(t *testing.T) {
	err := createWorkflowTemplate(templateRefTarget)
	assert.NoError(t, err)
	_, err = validate(wfWithWFTRef)
	assert.NoError(t, err)
}

const invalidWFWithWFTRef = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: A
  arguments:
    parameters:
    - name: lines-count
      value: 3
  workflowTemplateRef:
    name: template-ref-target
  templates:
  - name: A
    container:
      image: alpine:latest
      command: [echo, hello]
`

func TestValidateFieldsWithWFTRef(t *testing.T) {
	err := createWorkflowTemplate(templateRefTarget)
	assert.NoError(t, err)
	_, err = validate(invalidWFWithWFTRef)
	assert.Error(t, err)
}

var invalidWfNoImage = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-right-env-12
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      command:
      - cowsay
      args:
      - hello world
      env: []`

func TestInvalidWfNoImageField(t *testing.T) {
	_, err := validate(invalidWfNoImage)
	assert.EqualError(t, err, "templates.whalesay.container.image may not be empty")
}

var invalidWfNoImageScript = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-right-env-12
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    script:
      command:
      - cowsay
      args:
      - hello world
      env: []`

func TestInvalidWfNoImageFieldScript(t *testing.T) {
	_, err := validate(invalidWfNoImageScript)
	assert.EqualError(t, err, "templates.whalesay.script.image may not be empty")
}

var templateRefWithParam = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: template-ref-with-param
spec:
  entrypoint: A
  arguments:
    parameters:
    - name: some-param
  templates:
  - name: A
    container:
      image: alpine:latest
      command: [echo, hello]
`

var wfWithWFTRefOverrideParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  namespace: default
spec:
  arguments:
    parameters:
    - name: some-param
      value: a-value
  workflowTemplateRef:
    name: template-ref-with-param
`

func TestWorkflowWithWFTRefWithOverrideParam(t *testing.T) {
	err := createWorkflowTemplate(templateRefWithParam)
	assert.NoError(t, err)
	_, err = validate(wfWithWFTRefOverrideParam)
	assert.NoError(t, err)
}

var dagAndStepLevelOutputArtifacts = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-target-
spec:
  entrypoint: main
  templates:
  - name: main
    outputs:
      artifacts:
        - name: artifact
          from: "{{tasks.artifact-svn-retrieve.outputs.artifacts.artifact}}"
    dag:
      tasks:
      - name: artifact-svn-retrieve
        template: artifact-svn-retrieve
      - name: step-tmpl
        template: step

  - name: step
    outputs:
      artifacts:
        - name: artifact
          from: "{{steps.artifact-svn-retrieve.outputs.artifacts.artifact}}"
    steps:
    - - name: artifact-svn-retrieve
        template: artifact-svn-retrieve

  - name: artifact-svn-retrieve
    outputs:
      artifacts:
      - name: artifact
        path: "/vol/hello_world.txt"
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["sleep 1; cowsay hello world | tee /vol/hello_world.txt"]
      volumeMounts:
      - name: vol
        mountPath: "/vol"
    volumes:
    - name: vol
      emptyDir: {}
`

func TestDagAndStepLevelOutputArtifactsForDiffExecutor(t *testing.T) {
	t.Run("DefaultExecutor", func(t *testing.T) {
		_, err := validateWithOptions(dagAndStepLevelOutputArtifacts, ValidateOpts{ContainerRuntimeExecutor: ""})
		assert.NoError(t, err)
	})
	t.Run("EmissaryExecutor", func(t *testing.T) {
		_, err := validateWithOptions(dagAndStepLevelOutputArtifacts, ValidateOpts{ContainerRuntimeExecutor: common.ContainerRuntimeExecutorEmissary})
		assert.NoError(t, err)
	})
	t.Run("PNSExecutor", func(t *testing.T) {
		_, err := validateWithOptions(dagAndStepLevelOutputArtifacts, ValidateOpts{ContainerRuntimeExecutor: common.ContainerRuntimeExecutorPNS})
		assert.NoError(t, err)
	})
}

var testWorkflowTemplateLabels = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  generateName: test-foobar-
  labels:
    testLabel: foobar
spec:
  entrypoint: whalesay
  templates:
    - name: whalesay
      container:
        image: docker/whalesay
      metrics:
        prometheus:
          - name: intuit_data_persistplat_dppselfservice_workflow_test_duration
            help: Duration of workflow
            labels:
              - key: label
                value: "{{workflow.labels.testLabel}}"
            gauge:
              realtime: true
              value: "{{duration}}"
`

func TestWorkflowTemplateLabels(t *testing.T) {
	err := validateWorkflowTemplate(testWorkflowTemplateLabels, ValidateOpts{})
	assert.NoError(t, err)
}

const templateRefWithArtifactArgument = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: template-ref-with-artifact
spec:
  entrypoint: A
  arguments:
    artifacts:
    - name: binary-file
      http:
        url: https://a.server.io/file
  templates:
  - name: A
    inputs:
      artifacts:
      - name: binary-file
        path: /usr/local/bin/binfile
        mode: 0755
    container:
      image: alpine:latest
      command: [echo, hello]
`

const wfWithWFTRefAndNoOwnArtifact = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  namespace: default
spec:
  workflowTemplateRef:
    name: template-ref-with-artifact
`

func TestWorkflowWithWFTRefWithOutOwnArtifactArgument(t *testing.T) {
	err := createWorkflowTemplate(templateRefWithArtifactArgument)
	assert.NoError(t, err)
	_, err = validate(wfWithWFTRefAndNoOwnArtifact)
	assert.NoError(t, err)
}

const wfWithWFTRefAndOwnArtifactArgument = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  namespace: default
spec:
  arguments:
    artifacts:
    - name: binary-file
      http:
        url: http://localserver/file
  workflowTemplateRef:
    name: template-ref-with-artifact
`

func TestWorkflowWithWFTRefWithArtifactArgument(t *testing.T) {
	err := createWorkflowTemplate(templateRefWithArtifactArgument)
	assert.NoError(t, err)
	_, err = validate(wfWithWFTRefAndOwnArtifactArgument)
	assert.NoError(t, err)
}

var workflowTeamplateWithEnumValues = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  generateName: test-enum-1-
  labels:
    testLabel: foobar
spec:
  entrypoint: argosay
  arguments:
    parameters:
      - name: message
        value: one
        enum:
            - one
            - two
            - three
  templates:
    - name: argosay
      inputs:
        parameters:
          - name: message
            value: '{{workflow.parameters.message}}'
      container:
        name: main
        image: 'argoproj/argosay:v2'
        command:
          - /argosay
        args:
          - echo
          - '{{inputs.parameters.message}}'
`

var workflowTemplateWithEmptyEnumList = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  generateName: test-enum-1-
  labels:
    testLabel: foobar
spec:
  entrypoint: argosay
  arguments:
    parameters:
      - name: message
        value: one
        enum: []
  templates:
    - name: argosay
      inputs:
        parameters:
          - name: message
            value: '{{workflow.parameters.message}}'
      container:
        name: main
        image: 'argoproj/argosay:v2'
        command:
          - /argosay
        args:
          - echo
          - '{{inputs.parameters.message}}'
`

var workflowTemplateWithArgumentValueNotFromEnumList = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  generateName: test-enum-1-
  labels:
    testLabel: foobar
spec:
  entrypoint: argosay
  arguments:
    parameters:
      - name: message
        value: one
        enum:
            -   two
            -   three
            -   four
  templates:
    - name: argosay
      inputs:
        parameters:
          - name: message
            value: '{{workflow.parameters.message}}'
      container:
        name: main
        image: 'argoproj/argosay:v2'
        command:
          - /argosay
        args:
          - echo
          - '{{inputs.parameters.message}}'
`

func TestWorkflowTemplateWithEnumValue(t *testing.T) {
	err := validateWorkflowTemplate(workflowTeamplateWithEnumValues, ValidateOpts{})
	assert.NoError(t, err)
	err = validateWorkflowTemplate(workflowTeamplateWithEnumValues, ValidateOpts{Lint: true})
	assert.NoError(t, err)
	err = validateWorkflowTemplate(workflowTeamplateWithEnumValues, ValidateOpts{Submit: true})
	assert.NoError(t, err)
}

func TestWorkflowTemplateWithEmptyEnumList(t *testing.T) {
	err := validateWorkflowTemplate(workflowTemplateWithEmptyEnumList, ValidateOpts{})
	assert.EqualError(t, err, "spec.arguments.message.enum should contain at least one value")
	err = validateWorkflowTemplate(workflowTemplateWithEmptyEnumList, ValidateOpts{Lint: true})
	assert.EqualError(t, err, "spec.arguments.message.enum should contain at least one value")
	err = validateWorkflowTemplate(workflowTemplateWithEmptyEnumList, ValidateOpts{Submit: true})
	assert.EqualError(t, err, "spec.arguments.message.enum should contain at least one value")
}

func TestWorkflowTemplateWithArgumentValueNotFromEnumList(t *testing.T) {
	err := validateWorkflowTemplate(workflowTemplateWithArgumentValueNotFromEnumList, ValidateOpts{})
	assert.EqualError(t, err, "spec.arguments.message.value should be present in spec.arguments.message.enum list")
	err = validateWorkflowTemplate(workflowTemplateWithArgumentValueNotFromEnumList, ValidateOpts{Lint: true})
	assert.EqualError(t, err, "spec.arguments.message.value should be present in spec.arguments.message.enum list")
	err = validateWorkflowTemplate(workflowTemplateWithArgumentValueNotFromEnumList, ValidateOpts{Submit: true})
	assert.EqualError(t, err, "spec.arguments.message.value should be present in spec.arguments.message.enum list")
}

var validActiveDeadlineSecondsArgoVariable = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: timeout-bug-
spec:
  entrypoint: main

  templates:
    - name: main
      dag:
        tasks:
          - name: print-timeout
            template: print-timeout
          - name: use-timeout
            template: use-timeout
            dependencies: [print-timeout]
            arguments:
              parameters:
                - name: timeout
                  value: "{{tasks.print-timeout.outputs.result}}"

    - name: print-timeout
      container:
        image: alpine
        command: [sh, -c]
        args: ['echo 5']

    - name: use-timeout
      inputs:
        parameters:
          - name: timeout
      activeDeadlineSeconds: "{{inputs.parameters.timeout}}"
      container:
        image: alpine
        command: [sh, -c]
        args: ["echo sleeping for 1m; sleep 60; echo done"]
`

func TestValidActiveDeadlineSecondsArgoVariable(t *testing.T) {
	err := validateWorkflowTemplate(validActiveDeadlineSecondsArgoVariable, ValidateOpts{})
	assert.NoError(t, err)
}

func TestMaxLengthName(t *testing.T) {
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: strings.Repeat("a", 70)}}
	_, err := ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, ValidateOpts{})
	assert.EqualError(t, err, "workflow name \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\" must not be more than 63 characters long (currently 70)")

	wftmpl := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: strings.Repeat("a", 70)}}
	_, err = ValidateWorkflowTemplate(wftmplGetter, cwftmplGetter, wftmpl, ValidateOpts{})
	assert.EqualError(t, err, "workflow template name \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\" must not be more than 63 characters long (currently 70)")

	cwftmpl := &wfv1.ClusterWorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: strings.Repeat("a", 70)}}
	_, err = ValidateClusterWorkflowTemplate(wftmplGetter, cwftmplGetter, cwftmpl, ValidateOpts{})
	assert.EqualError(t, err, "cluster workflow template name \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\" must not be more than 63 characters long (currently 70)")

	cwf := &wfv1.CronWorkflow{ObjectMeta: metav1.ObjectMeta{Name: strings.Repeat("a", 60)}}
	err = ValidateCronWorkflow(wftmplGetter, cwftmplGetter, cwf)
	assert.EqualError(t, err, "cron workflow name \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\" must not be more than 52 characters long (currently 60)")
}

var invalidContainerSetDependencyNotFound = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: pod
spec:
  entrypoint: main
  templates:
    - name: main
      volumes:
        - name: workspace
          emptyDir: { }
      containerSet:
        volumeMounts:
          - name: workspace
            mountPath: /workspace
        containers:
          - name: a
            image: argoproj/argosay:v2
          - name: b
            image: argoproj/argosay:v2
            dependencies:
              - c
`

func TestInvalidContainerSetDependencyNotFound(t *testing.T) {
	_, err := validate(invalidContainerSetDependencyNotFound)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "templates.main.containerSet.containers.b dependency 'c' not defined")
	}
}

func TestInvalidContainerSetNoMainContainer(t *testing.T) {
	invalidContainerSetTemplateWithInputArtifacts := `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: workflow
spec:
  entrypoint: main
  templates:
    - name: main
      inputs:
        artifacts:
          - name: message
            path: /tmp/message
      containerSet:
        containers:
          - name: a
            image: argoproj/argosay:v2
`
	invalidContainerSetTemplateWithOutputArtifacts := `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: workflow
spec:
  entrypoint: main
  templates:
    - name: main
      outputs:
        artifacts:
          - name: message
            path: /tmp/message
      containerSet:
        containers:
          - name: a
            image: argoproj/argosay:v2
`
	invalidContainerSetTemplateWithOutputParams := `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: workflow
spec:
  entrypoint: main
  templates:
    - name: main
      outputs:
        parameters:
          - name: output-message
            valueFrom:
              path: /workspace/message
      containerSet:
        containers:
          - name: a
            image: argoproj/argosay:v2
`

	invalidManifests := []string{
		invalidContainerSetTemplateWithInputArtifacts,
		invalidContainerSetTemplateWithOutputArtifacts,
		invalidContainerSetTemplateWithOutputParams,
	}
	for _, manifest := range invalidManifests {
		err := validateWorkflowTemplate(manifest, ValidateOpts{})
		if assert.NotNil(t, err) {
			assert.Contains(t, err.Error(), "containerSet.containers must have a container named \"main\" for input or output")
		}
	}
}

func TestSortDAGTasksWithDepends(t *testing.T) {
	wfUsingDependsManifest := `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: sort-dag-tasks-test-
  namespace: argo
spec:
  entrypoint: main
  templates:
    - dag:
        tasks:
          - name: "8ea51cf2"
            template: 8ea51cf2-template
          - depends: 8ea51cf2
            name: "ba1f414f"
            template: ba1f414f-template
          - depends: ba1f414f.Succeeded || ba1f414f.Failed || ba1f414f.Errored
            name: "f7d273f8"
            template: f7d273f8-template
      name: main`
	wf := unmarshalWf(wfUsingDependsManifest)
	tmpl := wf.Spec.Templates[0]
	nameToTask := make(map[string]wfv1.DAGTask)
	for _, task := range tmpl.DAG.Tasks {
		nameToTask[task.Name] = task
	}

	dagValidationCtx := &dagValidationContext{
		tasks:        nameToTask,
		dependencies: make(map[string]map[string]common.DependencyType),
	}
	err := sortDAGTasks(&tmpl, dagValidationCtx)
	assert.NoError(t, err)
	var taskOrderAfterSort, expectedOrder []string
	expectedOrder = []string{"8ea51cf2", "ba1f414f", "f7d273f8"}
	for _, task := range tmpl.DAG.Tasks {
		taskOrderAfterSort = append(taskOrderAfterSort, task.Name)
	}
	assert.Equal(t, expectedOrder, taskOrderAfterSort)
}

func TestValidateStartedATVariable(t *testing.T) {
	wf := `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-timing-
spec:
  entrypoint: steps-timing
  templates:
    
    - name: steps-timing
      steps:
        - - name: one
            template: wait
        - - name: print-processing-time
            template: printer
            arguments:
              parameters:
                - name: startedat
                  value: "{{steps.one.startedAt}}"
                - name: finishedat
                  value: "{{steps.one.finishedAt}}"
                - name: id
                  value: "{{steps.one.id}}"
    
    - name: wait
      container:
        image: alpine:3.7
        command: [sleep, "5"]
    
    - name: printer
      inputs:
        parameters:
          - name: startedat
          - name: finishedat
          - name: id
      container:
        image: alpine:3.7
        command: [echo, "{{inputs.parameters.startedat}}"]`
	_, err := validate(wf)
	assert.NoError(t, err)
}

var templateReferenceWorkflowConfigMapRefArgument = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: arguments-parameters-from-configmap-
spec:
  entrypoint: whalesay
  serviceAccountName: argo
  arguments:
    parameters:
    - name: message
      valueFrom:
        configMapKeyRef:
          name: simple-parameters
          key: msg
  templates:
    - name: whalesay
      inputs:
        parameters:
          - name: message
      container:
        image: docker/whalesay:latest
        command: [cowsay]
        args: ["{{inputs.parameters.message}}"]
`

func TestTemplateReferenceWorkflowConfigMapRefArgument(t *testing.T) {
	_, err := validate(templateReferenceWorkflowConfigMapRefArgument)
	assert.NoError(t, err)
}

var stepsOutputParametersForScript = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: parameter-aggregation-
spec:
  entrypoint: parameter-aggregation
  templates:
    - name: parameter-aggregation
      steps:
        - - name: echo-num
            template: echo-num
            arguments:
              parameters:
                - name: num
                  value: "{{item}}"
            withItems: [1, 2, 3, 4]
        - - name: echo-num-from-param
            template: echo-num
            arguments:
              parameters:
                - name: num
                  value: "{{item.num}}"
            withParam: "{{steps.echo-num.outputs.parameters}}"

    - name: echo-num
      inputs:
        parameters:
          - name: num
      script:
        image: argoproj/argosay:v1
        command: [sh, -x]
        source: |
          sleep 1
          echo {{inputs.parameters.num}} > /tmp/num
      outputs:
        parameters:
          - name: num
            valueFrom:
              path: /tmp/num
`

func TestStepsOutputParametersForScript(t *testing.T) {
	_, err := validate(stepsOutputParametersForScript)
	assert.NoError(t, err)
}

var stepsOutputParametersForContainerSet = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: parameter-aggregation-
spec:
  entrypoint: parameter-aggregation
  templates:
    - name: parameter-aggregation
      steps:
        - - name: echo-num
            template: echo-num
            arguments:
              parameters:
                - name: num
                  value: "{{item}}"
            withItems: [1, 2, 3, 4]
        - - name: echo-num-from-param
            template: echo-num
            arguments:
              parameters:
                - name: num
                  value: "{{item.num}}"
            withParam: "{{steps.echo-num.outputs.parameters}}"

    - name: echo-num
      inputs:
        parameters:
          - name: num
      containerSet:
        containers:
          - name: main
            image: 'docker/whalesay:latest'
            command:
              - sh
              - '-c'
            args:
              - 'sleep 1; echo {{inputs.parameters.num}} > /tmp/num'
      outputs:
        parameters:
          - name: num
            valueFrom:
              path: /tmp/num
`

func TestStepsOutputParametersForContainerSet(t *testing.T) {
	_, err := validate(stepsOutputParametersForContainerSet)
	assert.NoError(t, err)
}

var globalAnnotationsAndLabels = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  labels:
    testLabel: foobar
  annotations:
    workflows.argoproj.io/description: |
      This is a simple hello world example.
spec:
  entrypoint: whalesay1
  arguments:
    parameters:
    - name: message
      value: hello world
  templates:
  - name: whalesay1
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["{{workflow.annotations}},  {{workflow.labels}}"]`

func TestResolveAnnotationsAndLabelsJSson(t *testing.T) {
	_, err := validate(globalAnnotationsAndLabels)
	assert.NoError(t, err)
}

var testInitContainerHasName = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: spurious-
spec:
  entrypoint: main

  templates:
  - name: main
    dag:
      tasks:
      - name: spurious
        template: spurious

  - name: spurious
    retryStrategy:
      retryPolicy: Always
    initContainers:
    - image: alpine:latest
      # name: sleep
      command:
      - sleep
      - "15"
    container:
      image: alpine:latest
      command:
      - echo
      - "i am running"
`

func TestInitContainerHasName(t *testing.T) {
	_, err := validate(testInitContainerHasName)
	assert.EqualError(t, err, "templates.main.tasks.spurious initContainers must all have container name")
}
