package validate

import (
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test"
	"github.com/argoproj/argo/workflow/common"
)

// validate is a test helper to accept YAML as a string and return
// its validation result.
func validate(yamlStr string) error {
	wf := unmarshalWf(yamlStr)
	return ValidateWorkflow(wf, ValidateOpts{})
}

func unmarshalWf(yamlStr string) *wfv1.Workflow {
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(yamlStr), &wf)
	if err != nil {
		panic(err)
	}
	return &wf
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
	err := validate(unresolvedInput)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "failed to resolve")
	}
	err = validate(unresolvedOutput)
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
	err := validate(ioArtifactPaths)
	assert.Nil(t, err)
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
	err := validate(outputParameterPath)
	assert.Nil(t, err)
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
      args: ["{{inputs.parameters.message+123}}"]
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
      - name: "!1"
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
	err := validate(invalidOutputParamNames)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), invalidErr)
	}
	err = validate(invalidOutputMissingValueFrom)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "valueFrom not specified")
	}
	err = validate(invalidOutputMultipleValueFrom)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "multiple valueFrom")
	}
	err = validate(invalidOutputIncompatibleValueFromPath)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), ".path must be specified for Container templates")
	}
	err = validate(invalidOutputIncompatibleValueFromParam)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), ".parameter must be specified for Steps templates")
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
      args: ["echo {{workflow.status}} {{workflow.uid}}"]
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
          repo: https://github.com/argoproj/argo.git
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
	err := ValidateWorkflow(wf, ValidateOpts{})
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "already mounted")
	}
	// tweak the mount path and validation should now be successful
	wf.Spec.Templates[0].Container.VolumeMounts[0].MountPath = "/differentpath"
	err = ValidateWorkflow(wf, ValidateOpts{})
	assert.Nil(t, err)
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
	// ensure {{workflow.status}} is not available when not in exit handler
	err := validate(activeDeadlineSeconds)
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
	err := validate(leafWithParallelism)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "is only valid")
	}
}

var nonLeafWithRetryStrategy = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: non-leaf-with-retry-strategy
spec:
  entrypoint: non-leaf-with-retry-strategy
  templates:
  - name: non-leaf-with-retry-strategy
    retryStrategy:
      limit: 4
    steps:
    - - name: try
        template: try
  - name: try
    container:
      image: debian:9.4
      command: [sh, -c]
      args: ["kubectl version"]
`

func TestNonLeafWithRetryStrategy(t *testing.T) {
	err := validate(nonLeafWithRetryStrategy)
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
    input:
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
    input:
      artifacts:
      - name: art
        path: /tmp/art
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world"]
`

func TestInvalidArgumentNoFromOrLocation(t *testing.T) {
	err := validate(invalidStepsArgumentNoFromOrLocation)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "from or artifact location is required")
	}
	err = validate(invalidDAGArgumentNoFromOrLocation)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "from or artifact location is required")
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
    input:
      parameters:
      - name: art
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world"]
`

func TestInvalidArgumentNoValue(t *testing.T) {
	err := validate(invalidArgumentNoValue)
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

  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

var invalidWithItems = `
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
        - hello world
        - goodbye world
        - [a, b, c]

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
	err := validate(validWithItems)
	assert.Nil(t, err)

	err = validate(invalidWithItems)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "withItems")
	}
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
	err := validate(podNameVariable)
	assert.Nil(t, err)
}

func TestGlobalParamWithVariable(t *testing.T) {
	err := ValidateWorkflow(test.LoadE2EWorkflow("functional/global-outputs-variable.yaml"), ValidateOpts{})
	assert.Nil(t, err)
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
	err := ValidateWorkflow(wf, ValidateOpts{Lint: true})
	assert.Nil(t, err)
	err = ValidateWorkflow(wf, ValidateOpts{})
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
	err := ValidateWorkflow(wf, ValidateOpts{Lint: true})
	assert.Nil(t, err)
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
	err := ValidateWorkflow(wf, ValidateOpts{Lint: true})
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
	err := ValidateWorkflow(wf, ValidateOpts{Lint: true})
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

	for _, executor := range []string{common.ContainerRuntimeExecutorK8sAPI, common.ContainerRuntimeExecutorKubelet, common.ContainerRuntimeExecutorPNS, common.ContainerRuntimeExecutorDocker, ""} {
		switch executor {
		case common.ContainerRuntimeExecutorK8sAPI, common.ContainerRuntimeExecutorKubelet:
			err = ValidateWorkflow(wfBaseOutArt, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.Error(t, err)
			err = ValidateWorkflow(wfBaseOutParam, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.Error(t, err)
			err = ValidateWorkflow(wfBaseWithEmptyDirOutArt, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.Error(t, err)
		case common.ContainerRuntimeExecutorPNS:
			err = ValidateWorkflow(wfBaseOutArt, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.NoError(t, err)
			err = ValidateWorkflow(wfBaseOutParam, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.NoError(t, err)
			err = ValidateWorkflow(wfBaseWithEmptyDirOutArt, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.Error(t, err)
		case common.ContainerRuntimeExecutorDocker, "":
			err = ValidateWorkflow(wfBaseOutArt, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.NoError(t, err)
			err = ValidateWorkflow(wfBaseOutParam, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.NoError(t, err)
			err = ValidateWorkflow(wfBaseWithEmptyDirOutArt, ValidateOpts{ContainerRuntimeExecutor: executor})
			assert.NoError(t, err)
		}
		err = ValidateWorkflow(wfEmptyDirOutArt, ValidateOpts{ContainerRuntimeExecutor: executor})
		assert.NoError(t, err)
	}
}
