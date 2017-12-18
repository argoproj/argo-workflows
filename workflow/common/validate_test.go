package common

import (
	"testing"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
)

// validate is a test helper to accept YAML as a string and return
// its validation result.
func validate(yamlStr string) error {
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(yamlStr), &wf)
	if err != nil {
		return err
	}
	return ValidateWorkflow(&wf)
}

const invalidErr = "is invalid"

var unknownField = `
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
      unknown_field: ""
`

func TestUnknownField(t *testing.T) {
	t.Skip("Cannot detect unknown fields yet")
	err := validate(unknownField)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "invalid keys: unknown_field")
	}
}

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
	var err error
	err = validate(dupTemplateNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "not unique")
	}
	err = validate(dupInputNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "not unique")
	}
	err = validate(emptyName)
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

func TestUnresolved(t *testing.T) {
	err := validate(unresolvedInput)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "failed to resolve")
	}
}

var stepOutputReferences = `
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
        value: "value"
    container:
      image: docker/whalesay:latest
    outputs:
      parameters:
      - name: outparam
  - name: stepref
    steps:
    - - name: one
        template: whalesay
    - - name: two
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "{{steps.one.outparam}}"
`

func TestStepReference(t *testing.T) {
	err := validate(stepOutputReferences)
	assert.Nil(t, err)
}

var unsatisfiedParam = `
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

func TestUnsatisfiedParam(t *testing.T) {
	err := validate(unsatisfiedParam)
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

func TestGlobalParam(t *testing.T) {
	err := validate(globalParam)
	assert.Nil(t, err)
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
	err := validate(invalidTemplateNames)
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
	err := validate(invalidArgParamNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), invalidErr)
	}
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
    - name: -kubectl
      http:
        url: https://storage.googleapis.com/kubernetes-release/release/v1.8.0/bin/linux/amd64/kubectl

  templates:
  - name: kubectl-input-artifact
    inputs:
      artifacts:
      - name: -kubectl
        path: /usr/local/bin/kubectl
        mode: 755
    container:
      image: debian:9.1
      command: [sh, -c]
      args: ["kubectl version"]
`

func TestInvalidArgArtName(t *testing.T) {
	err := validate(invalidArgArtNames)
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
	err := validate(invalidStepNames)
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
      args: ["{{inputs.parameters.message}}"]
`

func TestInvalidInputParamName(t *testing.T) {
	err := validate(invalidInputParamNames)
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
	err := validate(invalidInputArtNames)
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
      - name: __1
        path: /tmp/hello_world.txt
`

func TestInvalidOutputArtName(t *testing.T) {
	err := validate(invalidOutputArtNames)
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
        path: /tmp/hello_world.txt
`

func TestInvalidOutputParamName(t *testing.T) {
	err := validate(invalidOutputParamNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), invalidErr)
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
      image: python:3.6
      command: [python]
      source: |
        import random
        i = random.randint(1, 100)
        print(i)
`

func TestMultipleTemplateTypes(t *testing.T) {
	err := validate(multipleTemplateTypes)
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
      args: ["echo {{workflow.status}} {{workflow.uuid}}"]
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
      args: ["echo {{workflow.status}}"]
`

func TestExitHandler(t *testing.T) {
	// ensure {{workflow.status}} is not available when not in exit handler
	err := validate(workflowStatusNotOnExit)
	assert.NotNil(t, err)

	// ensure {{workflow.status}} is available in exit handler
	err = validate(exitHandlerWorkflowStatusOnExit)
	assert.Nil(t, err)
}
