package validate

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func createWorkflowTemplateFromSpec(ctx context.Context, yamlStr string) error {
	wftmpl := unmarshalWftmpl(yamlStr)
	return createWorkflowTemplate(ctx, wftmpl)
}

func createWorkflowTemplate(ctx context.Context, wftmpl *wfv1.WorkflowTemplate) error {
	_, err := wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault).Create(ctx, wftmpl, metav1.CreateOptions{})
	if err != nil && apierr.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func deleteWorkflowTemplate(ctx context.Context, name string) error {
	return wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault).Delete(ctx, name, metav1.DeleteOptions{})
}

// validate is a test helper to accept Workflow YAML as a string and return
// its validation result.
func validate(ctx context.Context, yamlStr string) error {
	wf := unmarshalWf(yamlStr)
	return Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{})
}

// validateWorkflowTemplate is a test helper to accept WorkflowTemplate YAML as a string and return
// its validation result.
func validateWorkflowTemplate(ctx context.Context, yamlStr string, opts Opts) error {
	wftmpl := unmarshalWftmpl(yamlStr)
	err := WorkflowTemplate(ctx, wftmplGetter, cwftmplGetter, wftmpl, nil, opts)
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
	ctx := logging.TestContext(t.Context())
	err := validate(ctx, dupTemplateNames)
	require.ErrorContains(t, err, "not unique")

	err = validate(ctx, dupInputNames)
	require.ErrorContains(t, err, "not unique")

	err = validate(ctx, emptyName)
	require.ErrorContains(t, err, "name is required")
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
	ctx := logging.TestContext(t.Context())
	err := validate(ctx, unresolvedInput)
	require.ErrorContains(t, err, "failed to resolve")

	err = validate(ctx, unresolvedStepInput)
	require.ErrorContains(t, err, "failed to resolve")

	err = validate(ctx, unresolvedOutput)
	require.ErrorContains(t, err, "failed to resolve")
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
	ctx := logging.TestContext(t.Context())
	err := validate(ctx, ioArtifactPaths)
	require.NoError(t, err)
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
	ctx := logging.TestContext(t.Context())
	err := validate(ctx, outputParameterPath)
	require.NoError(t, err)
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
	ctx := logging.TestContext(t.Context())
	err := validate(ctx, stepOutputReferences)
	require.NoError(t, err)
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
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo {{inputs.parameters.message}}"]
`

func TestStepStatusReference(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	err := validate(ctx, stepStatusReferences)
	require.NoError(t, err)
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
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo {{inputs.parameters.message}}"]
`

func TestStepStatusReferenceNoFutureReference(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	err := validate(ctx, stepStatusReferencesNoFutureReference)
	// Can't reference the status of steps that have not run yet
	require.ErrorContains(t, err, "failed to resolve {{steps.two.status}}")
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
      image: alpine:3.23
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
      image: alpine:3.23
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
	err := validate(logging.TestContext(t.Context()), stepArtReferences)
	require.NoError(t, err)
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
	err := validate(logging.TestContext(t.Context()), paramWithValueFromConfigMapRef)
	require.NoError(t, err)
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
	err := validate(logging.TestContext(t.Context()), paramWithoutValue)
	require.ErrorContains(t, err, "not supplied")
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
      image: alpine:3.23
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
      image: alpine:3.23
      command: [sh, -c]
      args: ["cat /art"]
`

func TestGlobalParam(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), globalParam)
	require.NoError(t, err)

	err = validate(logging.TestContext(t.Context()), nestedGlobalParam)
	require.NoError(t, err)

	err = validate(logging.TestContext(t.Context()), unsuppliedArgValue)
	require.EqualError(t, err, "spec.arguments.missing.value or spec.arguments.missing.valueFrom is required")
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
	err := validate(logging.TestContext(t.Context()), invalidTemplateNames)
	require.ErrorContains(t, err, invalidErr)
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
	err := validate(logging.TestContext(t.Context()), invalidArgParamNames)
	require.Error(t, err)
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
	err := validate(logging.TestContext(t.Context()), invalidArgArtNames)
	require.ErrorContains(t, err, invalidErr)
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
	err := validate(logging.TestContext(t.Context()), invalidStepNames)
	require.ErrorContains(t, err, invalidErr)
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
	err := validate(logging.TestContext(t.Context()), invalidInputParamNames)
	require.ErrorContains(t, err, invalidErr)
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
	err := validate(logging.TestContext(t.Context()), invalidInputArtNames)
	require.ErrorContains(t, err, invalidErr)
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
	err := validate(logging.TestContext(t.Context()), invalidOutputArtNames)
	require.Error(t, err, invalidErr)
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
	err := validate(logging.TestContext(t.Context()), invalidOutputParamNames)
	require.ErrorContains(t, err, invalidErr)

	err = validate(logging.TestContext(t.Context()), invalidOutputMissingValueFrom)
	require.ErrorContains(t, err, "does not have valueFrom or value specified")

	err = validate(logging.TestContext(t.Context()), invalidOutputMultipleValueFrom)
	require.ErrorContains(t, err, "multiple valueFrom")

	err = validate(logging.TestContext(t.Context()), invalidOutputIncompatibleValueFromPath)
	require.ErrorContains(t, err, ".path must be specified for Container templates")

	err = validate(logging.TestContext(t.Context()), invalidOutputIncompatibleValueFromParam)
	require.ErrorContains(t, err, ".parameter or expression must be specified for Steps templates")
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
      image: python:alpine3.23
      command: [python]
      source: |
        import random
        i = random.randint(1, 100)
        print(i)
`

func TestMultipleTemplateTypes(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), multipleTemplateTypes)
	require.ErrorContains(t, err, "multiple template types specified")
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
      image: alpine:3.23
      command: [sh, -c]
      args: ["exit 0"]
  - name: fail
    container:
      image: alpine:3.23
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
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo {{workflow.failures}}"]
`

func TestExitHandler(t *testing.T) {
	// ensure {{workflow.status}} is not available when not in exit handler
	err := validate(logging.TestContext(t.Context()), workflowStatusNotOnExit)
	require.Error(t, err)

	// ensure {{workflow.status}} is available in exit handler
	err = validate(logging.TestContext(t.Context()), exitHandlerWorkflowStatusOnExit)
	require.NoError(t, err)
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
	err := validate(logging.TestContext(t.Context()), workflowWithPriority)
	require.NoError(t, err)
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
      image: alpine:3.23
      command: [sh, -c]
      args: ["exit 0"]
      volumeMounts:
      - name: workdir
        mountPath: /src
`

func TestVolumeMountArtifactPathCollision(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// ensure we detect and reject path collisions
	wf := unmarshalWf(volumeMountArtifactPathCollision)

	err := Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{})

	require.ErrorContains(t, err, "already mounted")

	// tweak the mount path and validation should now be successful
	wf.Spec.Templates[0].Container.VolumeMounts[0].MountPath = "/differentpath"

	err = Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{})

	require.NoError(t, err)
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
      image: alpine:3.23
      command: [sh, -c]
      args: ["exit 0"]
`

func TestValidActiveDeadlineSeconds(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), activeDeadlineSeconds)
	require.ErrorContains(t, err, "activeDeadlineSeconds must be a positive integer > 0")
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
	err := validate(logging.TestContext(t.Context()), leafWithParallelism)
	require.ErrorContains(t, err, "is only valid")
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
	err := validate(logging.TestContext(t.Context()), invalidStepsArgumentNoFromOrLocation)
	require.ErrorContains(t, err, "from, artifact location, or key is required")

	err = validate(logging.TestContext(t.Context()), invalidDAGArgumentNoFromOrLocation)
	require.ErrorContains(t, err, "from, artifact location, or key is required")
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
	err := validate(logging.TestContext(t.Context()), invalidArgumentNoValue)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ".value or ")
	assert.Contains(t, err.Error(), ".valueFrom is required")
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
	err := validate(logging.TestContext(t.Context()), validWithItems)
	require.NoError(t, err)
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
	err := validate(logging.TestContext(t.Context()), podNameVariable)
	require.NoError(t, err)
}

func TestGlobalParamWithVariable(t *testing.T) {
	err := Workflow(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, wfv1.MustUnmarshalWorkflow("@../../test/e2e/functional/global-outputs-variable.yaml"), nil, Opts{})

	require.NoError(t, err)
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

	ctx := logging.TestContext(t.Context())
	err := Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{Lint: true})
	require.NoError(t, err)
	err = Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{})

	require.Error(t, err)
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
	err := Workflow(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, wf, nil, Opts{Lint: true})
	require.NoError(t, err)
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
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.num}}"]
`

// TestSpecBadSequenceCountAndEnd verifies both count and end cannot be defined
func TestSpecBadSequenceCountAndEnd(t *testing.T) {
	wf := unmarshalWf(specBadSequenceCountAndEnd)
	err := Workflow(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, wf, nil, Opts{Lint: true})
	require.Error(t, err)
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

// TestCustomTemplateVariable verifies custom template variable
func TestCustomTemplateVariable(t *testing.T) {
	wf := unmarshalWf(customVariableInput)
	err := Workflow(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, wf, nil, Opts{Lint: true})
	require.NoError(t, err)
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
      image: alpine:3.23
      command: [echo, hello]
`

func TestWorkflowTemplate(t *testing.T) {
	err := validateWorkflowTemplate(logging.TestContext(t.Context()), templateRefTarget, Opts{})
	require.NoError(t, err)
}

var templateWithGlobalParams = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: template-ref-target
spec:
  templates:
  - name: A
    container:
      image: alpine:3.23
      command: [echo, "{{workflow.parameters.something}}"]
`

func TestWorkflowTemplateWithGlobalParams(t *testing.T) {
	err := validateWorkflowTemplate(logging.TestContext(t.Context()), templateWithGlobalParams, Opts{})
	require.NoError(t, err)
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
	ctx := logging.TestContext(t.Context())
	err := createWorkflowTemplateFromSpec(ctx, templateRefTarget)
	require.NoError(t, err)
	err = createWorkflowTemplateFromSpec(ctx, templateRefNestedTarget)
	require.NoError(t, err)
	err = validate(logging.TestContext(t.Context()), nestedTemplateRef)
	require.NoError(t, err)
}

var templateRefTargetWithInput = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: template-ref-target-with-input
spec:
  templates:
  - name: A
    inputs:
      parameters:
        - name: message
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo {{inputs.parameters.message}}"]
`

var nestedTemplateRefWithError = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: template-ref-
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
      - - name: call-A
          templateRef:
            name: template-ref-target
            template: A
      - - name: call-A-input
          template: A
  - name: A
    steps:
      - - name: call-B
          templateRef:
            name: template-ref-target-with-input
            template: A
`

func TestNestedTemplateRefWithError(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	err := createWorkflowTemplateFromSpec(ctx, templateRefTarget)
	require.NoError(t, err)
	err = createWorkflowTemplateFromSpec(ctx, templateRefTargetWithInput)
	require.NoError(t, err)
	err = validate(logging.TestContext(t.Context()), nestedTemplateRefWithError)
	require.EqualError(t, err, "templates.main.steps[1].call-A-input templates.A.steps[0].call-B templates.A inputs.parameters.message was not supplied")
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
	err := validate(logging.TestContext(t.Context()), undefinedTemplateRef)
	require.ErrorContains(t, err, "not found")
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
	err := Workflow(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, wf, nil, Opts{})
	require.NoError(t, err)
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
	ctx := logging.TestContext(t.Context())
	err := Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{})
	require.EqualError(t, err, "templates.whalesay.resource.manifest must be a valid yaml")

	wf = unmarshalWf(invalidActionResourceWorkflow)
	err = Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{})
	require.EqualError(t, err, "templates.whalesay.resource.action must be one of: get, create, apply, delete, replace, patch")
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
	err := Workflow(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, wf, nil, Opts{})
	require.EqualError(t, err, "podGC.strategy unknown strategy 'Foo'")
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
	err := Workflow(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, wf, nil, Opts{})
	require.EqualError(t, err, "podGC.labelSelector invalid: \"InvalidOperator\" is not a valid label selector operator")
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
		err := Workflow(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, wf, nil, Opts{})
		require.NoError(t, err)
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
      image: alpine:3.23
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
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo {{inputs.parameters.parameter}}"]
`

// TestRuntimeResolutionOfVariableNames verifies an error against a workflow of an invalid resource.
func TestRuntimeResolutionOfVariableNames(t *testing.T) {
	wf := unmarshalWf(runtimeResolutionOfVariableNames)
	err := Workflow(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, wf, nil, Opts{})
	require.NoError(t, err)
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
	err := validate(logging.TestContext(t.Context()), stepWithItemParam)
	require.NoError(t, err)
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
	err := validate(logging.TestContext(t.Context()), invalidMetricName)
	require.EqualError(t, err, "templates.whalesay metric name 'invalid.metric.name' is invalid. Metric names must contain alphanumeric characters or '_'")
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
	err := validate(logging.TestContext(t.Context()), invalidMetricLabelName)
	require.EqualError(t, err, "metric label 'invalid.key' is invalid: keys may only contain alphanumeric characters or '_'")
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
	err := validate(logging.TestContext(t.Context()), invalidMetricHelp)
	require.EqualError(t, err, "templates.whalesay metric 'metric_name' must contain a help string under 'help: ' field")
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
	err := validate(logging.TestContext(t.Context()), invalidRealtimeMetricGauge)
	require.EqualError(t, err, "templates.whalesay metric 'metric_name' error: 'resourcesDuration.*' metrics cannot be used in real-time")
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
	err := validate(logging.TestContext(t.Context()), invalidNoValueMetricGauge)
	require.EqualError(t, err, "templates.whalesay metric 'metric_name' error: missing gauge.value")
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
	err := validate(logging.TestContext(t.Context()), validMetricGauges)
	require.NoError(t, err)
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
	err := validate(logging.TestContext(t.Context()), globalVariables)
	require.NoError(t, err)
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
	err := validateWorkflowTemplate(logging.TestContext(t.Context()), wfTemplateWithEntrypoint, Opts{})
	require.NoError(t, err)
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
      image: alpine:3.23
      command: [echo, hello]
`

func TestWorkflowWithWFTRefWithEntrypoint(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	err := createWorkflowTemplateFromSpec(ctx, templateWithEntrypoint)
	require.NoError(t, err)
	err = validate(ctx, wfWithWFTRefNoEntrypoint)
	require.NoError(t, err)
}

const wfWithWFTRef = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: A
  serviceAccountName: default
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
	ctx := logging.TestContext(t.Context())
	err := createWorkflowTemplateFromSpec(ctx, templateRefTarget)
	require.NoError(t, err)
	err = validate(ctx, wfWithWFTRef)
	require.NoError(t, err)
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
      image: alpine:3.23
      command: [echo, hello]
`

func TestValidateFieldsWithWFTRef(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	err := createWorkflowTemplateFromSpec(ctx, templateRefTarget)
	require.NoError(t, err)
	err = validate(ctx, invalidWFWithWFTRef)
	require.Error(t, err)
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
	err := validate(logging.TestContext(t.Context()), invalidWfNoImage)
	require.EqualError(t, err, "templates.whalesay.container.image may not be empty")
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
	err := validate(logging.TestContext(t.Context()), invalidWfNoImageScript)
	require.EqualError(t, err, "templates.whalesay.script.image may not be empty")
}

var invalidWfNoImageScriptInTemplateDefault = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-right-env-12
spec:
  entrypoint: whalesay
  templateDefaults:
    script:
      command: [cowsay]
  templates:
  - name: whalesay
    script:
      args:
      - hello world
      env: []`

func TestIinvalidWfNoImageScriptInTemplateDefault(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), invalidWfNoImageScriptInTemplateDefault)
	require.EqualError(t, err, "templates.whalesay.script.image may not be empty")
}

var validWfImageScriptInTemplateDefault = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-right-env-12
spec:
  entrypoint: whalesay
  templateDefaults:
    script:
      image: alpine:3.23
  templates:
  - name: whalesay
    script:
      command:
      - cowsay
      args:
      - hello world
      env: []`

func TestValidWfImageScriptInTemplateDefault(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), validWfImageScriptInTemplateDefault)
	require.NoError(t, err)
}

var validWfImageContainerInTemplateDefault = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-right-env-12
spec:
  entrypoint: whalesay
  templateDefaults:
    container:
      image: alpine:3.23
  templates:
  - name: whalesay
    container:
      command:
      - cowsay
      args:
      - hello world
      env: []`

func TestValidWfImageContainerInTemplateDefault(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), validWfImageContainerInTemplateDefault)
	require.NoError(t, err)
}

var templateRefScriptImageDefaultTarget = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: template-ref-no-script-image
spec:
  entrypoint: whalesay
  templateDefaults:
    script:
      image: alpine:3.23
  templates:
  - name: whalesay
    script:
      command: [cowsay]
      args: [hello world]
`

var wfWithWFTRefScriptImageInDefault = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  namespace: default
spec:
  workflowTemplateRef:
    name: template-ref-no-script-image
`

func TestValidateFieldsWithWFTRefScriptImageInDefault(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	err := createWorkflowTemplateFromSpec(ctx, templateRefScriptImageDefaultTarget)
	require.NoError(t, err)
	err = validate(ctx, wfWithWFTRefScriptImageInDefault)
	require.NoError(t, err)
}

var templateRefContainerImageDefaultTarget = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: template-ref-no-container-image
spec:
  entrypoint: whalesay
  templateDefaults:
    container:
      image: alpine:3.23
  templates:
  - name: whalesay
    container:
      command: [cowsay]
      args: [hello world]
`

var wfWithWFTRefContainerImageInDefault = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  namespace: default
spec:
  workflowTemplateRef:
    name: template-ref-no-container-image
`

func TestValidateFieldsWithWFTRefContainerImageInDefault(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	err := createWorkflowTemplateFromSpec(ctx, templateRefContainerImageDefaultTarget)
	require.NoError(t, err)
	err = validate(ctx, wfWithWFTRefContainerImageInDefault)
	require.NoError(t, err)
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
      image: alpine:3.23
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
	ctx := logging.TestContext(t.Context())
	err := createWorkflowTemplateFromSpec(ctx, templateRefWithParam)
	require.NoError(t, err)
	err = validate(ctx, wfWithWFTRefOverrideParam)
	require.NoError(t, err)
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
	err := validateWorkflowTemplate(logging.TestContext(t.Context()), testWorkflowTemplateLabels, Opts{})
	require.NoError(t, err)
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
      image: alpine:3.23
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
	ctx := logging.TestContext(t.Context())
	err := createWorkflowTemplateFromSpec(ctx, templateRefWithArtifactArgument)
	require.NoError(t, err)
	err = validate(ctx, wfWithWFTRefAndNoOwnArtifact)
	require.NoError(t, err)
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
	ctx := logging.TestContext(t.Context())
	err := createWorkflowTemplateFromSpec(ctx, templateRefWithArtifactArgument)
	require.NoError(t, err)
	err = validate(ctx, wfWithWFTRefAndOwnArtifactArgument)
	require.NoError(t, err)
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
var workflowTeamplateWithEnumValuesWithoutValue = `
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

func TestWorkflowTemplateWithEnumValue(t *testing.T) {
	err := validateWorkflowTemplate(logging.TestContext(t.Context()), workflowTeamplateWithEnumValues, Opts{})
	require.NoError(t, err)
	err = validateWorkflowTemplate(logging.TestContext(t.Context()), workflowTeamplateWithEnumValues, Opts{Lint: true})
	require.NoError(t, err)
	err = validateWorkflowTemplate(logging.TestContext(t.Context()), workflowTeamplateWithEnumValues, Opts{Submit: true})
	require.NoError(t, err)
}

func TestWorkflowTemplateWithEmptyEnumList(t *testing.T) {
	err := validateWorkflowTemplate(logging.TestContext(t.Context()), workflowTemplateWithEmptyEnumList, Opts{})
	require.EqualError(t, err, "spec.arguments.message.enum should contain at least one value")
	err = validateWorkflowTemplate(logging.TestContext(t.Context()), workflowTemplateWithEmptyEnumList, Opts{Lint: true})
	require.EqualError(t, err, "spec.arguments.message.enum should contain at least one value")
	err = validateWorkflowTemplate(logging.TestContext(t.Context()), workflowTemplateWithEmptyEnumList, Opts{Submit: true})
	require.EqualError(t, err, "spec.arguments.message.enum should contain at least one value")
}

func TestWorkflowTemplateWithArgumentValueNotFromEnumList(t *testing.T) {
	err := validateWorkflowTemplate(logging.TestContext(t.Context()), workflowTemplateWithArgumentValueNotFromEnumList, Opts{})
	require.EqualError(t, err, "spec.arguments.message.value should be present in spec.arguments.message.enum list")
	err = validateWorkflowTemplate(logging.TestContext(t.Context()), workflowTemplateWithArgumentValueNotFromEnumList, Opts{Lint: true})
	require.EqualError(t, err, "spec.arguments.message.value should be present in spec.arguments.message.enum list")
	err = validateWorkflowTemplate(logging.TestContext(t.Context()), workflowTemplateWithArgumentValueNotFromEnumList, Opts{Submit: true})
	require.EqualError(t, err, "spec.arguments.message.value should be present in spec.arguments.message.enum list")
}

func TestWorkflowTemplateWithEnumValueWithoutValue(t *testing.T) {
	err := validateWorkflowTemplate(logging.TestContext(t.Context()), workflowTeamplateWithEnumValuesWithoutValue, Opts{})
	require.NoError(t, err)
	err = validateWorkflowTemplate(logging.TestContext(t.Context()), workflowTeamplateWithEnumValuesWithoutValue, Opts{Lint: true})
	require.NoError(t, err)
	err = validateWorkflowTemplate(logging.TestContext(t.Context()), workflowTeamplateWithEnumValuesWithoutValue, Opts{Submit: true})
	require.EqualError(t, err, "spec.arguments.message.value or spec.arguments.message.valueFrom is required")
}

var resourceManifestWithExpressions = `
apiVersion: v1
kind: Pod
metadata:
  name: foo
spec:
  restartPolicy: Never
  containers:
  - name: 'foo'
    image: docker/whalesay
    command: [cowsay]
    args: ["{{ = asInt(inputs.parameters.intParam) }}"]
    ports:
    - containerPort: {{=asInt(inputs.parameters.intParam)}}
`

func TestSubstituteResourceManifestExpressions(t *testing.T) {
	replaced := SubstituteResourceManifestExpressions(resourceManifestWithExpressions)
	assert.NotEqual(t, resourceManifestWithExpressions, replaced)

	// despite spacing in the expr itself we should have only 1 placeholder here
	patt := regexp.MustCompile(`placeholder\-\d+`)
	matches := patt.FindAllString(replaced, -1)
	assert.Len(t, matches, 2)
	assert.Equal(t, matches[0], matches[1])
}

var validWorkflowTemplateWithResourceManifest = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-with-resource-expr
spec:
  entrypoint: whalesay
  templates:
    - name: whalesay
      inputs:
        parameters:
          - name: intParam
            value: '20'
          - name: strParam
            value: 'foobarbaz'
      outputs: {}
      metadata: {}
      resource:
        action: create
        setOwnerReference: true
        manifest: |
          apiVersion: v1
          kind: Pod
          metadata:
            name: foo
          spec:
            restartPolicy: Never
            containers:
            - name: 'foo'
              image: docker/whalesay
              command: [cowsay]
              args: ["{{=sprig.replace("bar", "baz", inputs.parameters.strParam)}}"]
              ports:
              - containerPort: {{=asInt(inputs.parameters.intParam)}}
`

func TestWorkflowTemplateWithResourceManifest(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), validWorkflowTemplateWithResourceManifest)
	require.NoError(t, err)
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
	err := validateWorkflowTemplate(logging.TestContext(t.Context()), validActiveDeadlineSecondsArgoVariable, Opts{})
	require.NoError(t, err)
}

func TestMaxLengthName(t *testing.T) {
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: strings.Repeat("a", 70)}}
	err := Workflow(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, wf, nil, Opts{})
	require.EqualError(t, err, "workflow name \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\" must not be more than 63 characters long (currently 70)")

	wftmpl := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: strings.Repeat("a", 70)}}
	err = WorkflowTemplate(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, wftmpl, nil, Opts{})
	require.EqualError(t, err, "workflow template name \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\" must not be more than 63 characters long (currently 70)")

	cwftmpl := &wfv1.ClusterWorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: strings.Repeat("a", 70)}}
	err = ClusterWorkflowTemplate(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, cwftmpl, nil, Opts{})
	require.EqualError(t, err, "cluster workflow template name \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\" must not be more than 63 characters long (currently 70)")

	cwf := &wfv1.CronWorkflow{ObjectMeta: metav1.ObjectMeta{Name: strings.Repeat("a", 60)}}
	err = CronWorkflow(logging.TestContext(t.Context()), wftmplGetter, cwftmplGetter, cwf, nil)
	require.EqualError(t, err, "cron workflow name \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\" must not be more than 52 characters long (currently 60)")
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
	err := validate(logging.TestContext(t.Context()), invalidContainerSetDependencyNotFound)
	require.ErrorContains(t, err, "templates.main.containerSet.containers.b dependency 'c' not defined")
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
		err := validateWorkflowTemplate(logging.TestContext(t.Context()), manifest, Opts{})
		require.ErrorContains(t, err, "containerSet.containers must have a container named \"main\" for input or output")
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
	err := sortDAGTasks(logging.TestContext(t.Context()), &tmpl, dagValidationCtx)
	require.NoError(t, err)
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
        image: alpine:3.23
        command: [sleep, "5"]

    - name: printer
      inputs:
        parameters:
          - name: startedat
          - name: finishedat
          - name: id
      container:
        image: alpine:3.23
        command: [echo, "{{inputs.parameters.startedat}}"]`
	err := validate(logging.TestContext(t.Context()), wf)
	require.NoError(t, err)
}

var templateReferenceWorkflowConfigMapRefArgument = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: arguments-parameters-from-configmap-
spec:
  entrypoint: whalesay
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
	err := validate(logging.TestContext(t.Context()), templateReferenceWorkflowConfigMapRefArgument)
	require.NoError(t, err)
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
	err := validate(logging.TestContext(t.Context()), stepsOutputParametersForScript)
	require.NoError(t, err)
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
	err := validate(logging.TestContext(t.Context()), stepsOutputParametersForContainerSet)
	require.NoError(t, err)
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
	err := validate(logging.TestContext(t.Context()), globalAnnotationsAndLabels)
	require.NoError(t, err)
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
    - image: alpine:3.23
      # name: sleep
      command:
      - sleep
      - "15"
    container:
      image: alpine:3.23
      command:
      - echo
      - "i am running"
`

func TestInitContainerHasName(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), testInitContainerHasName)
	require.EqualError(t, err, "templates.main.tasks.spurious initContainers must all have container name")
}

var nodeNamePlumbsCorrectly = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
    generateName: hello-world-
spec:
    entrypoint: main
    templates:
      - name: main
        dag:
          tasks:
            - name: this-is-part-1
              template: main2
      - name: main2
        steps:
          - - name: this-is-part-2
              template: main3
      - name: main3
        dag:
          tasks:
            - name: this-is-part-3
              template: whalesay
      - name: whalesay
        container:
          image: docker/whalesay:latest
          command: [cowsay]
          args: ["{{ node.name }}"]`

func TestNodeNameParameterInterpoliates(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), nodeNamePlumbsCorrectly)
	require.NoError(t, err)
}

func TestSubstituteGlobalVariablesLabelsAnnotations(t *testing.T) {
	wfDefaults := wfv1.Workflow{
		Spec: wfv1.WorkflowSpec{
			WorkflowMetadata: &wfv1.WorkflowMetadata{
				Labels: map[string]string{
					"default-label": "thisLabelIsFromWorkflowDefaults",
				},
			},
		},
	}

	tests := []struct {
		name             string
		workflow         string
		workflowTemplate string
		expectedSuccess  bool
	}{
		{
			// entire template referenced; value not contained in WorkflowTemplate or Workflow
			workflow:         "@testdata/workflow-sub-test-1.yaml",
			workflowTemplate: "@testdata/workflow-template-sub-test-1.yaml",
			expectedSuccess:  false,
		},
		{
			// entire template referenced; value is in Workflow.Labels
			workflow:         "@testdata/workflow-sub-test-2.yaml",
			workflowTemplate: "@testdata/workflow-template-sub-test-1.yaml",
			expectedSuccess:  true,
		},
		{
			// entire template referenced; value is in WorkflowTemplate.workflowMetadata
			workflow:         "@testdata/workflow-sub-test-1.yaml",
			workflowTemplate: "@testdata/workflow-template-sub-test-2.yaml",
			expectedSuccess:  true,
		},

		{
			// entire template referenced; value is in Workflow.workflowMetadata
			workflow:         "@testdata/workflow-sub-test-3.yaml",
			workflowTemplate: "@testdata/workflow-template-sub-test-3.yaml",
			expectedSuccess:  true,
		},
		{
			// just a single template from the WorkflowTemplate is referenced:
			// shouldn't have access to the global scope of the WorkflowTemplate
			workflow:         "@testdata/workflow-sub-test-4.yaml",
			workflowTemplate: "@testdata/workflow-template-sub-test-2.yaml",
			expectedSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())

			wf := wfv1.MustUnmarshalWorkflow(tt.workflow)
			wftmpl := wfv1.MustUnmarshalWorkflowTemplate(tt.workflowTemplate)
			err := createWorkflowTemplate(ctx, wftmpl)
			if err != nil {
				require.NoError(t, err)
			}

			err = Workflow(ctx, wftmplGetter, cwftmplGetter, wf, &wfDefaults, Opts{})
			if tt.expectedSuccess {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			_ = deleteWorkflowTemplate(ctx, wftmpl.Name)
		})
	}
}

var spacedParameterWorkflowTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
    generateName: hello-world-
spec:
  entrypoint: helloworld

  templates:
  - name: helloworld
    container:
      image: "alpine:3.23"
      command: ["echo", "{{  workflow.thisdoesnotexist  }}"]
`

func TestShouldCheckValidationToSpacedParameters(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), spacedParameterWorkflowTemplate)
	// Do not allow leading or trailing spaces in parameters
	require.ErrorContains(t, err, "failed to resolve {{  workflow.thisdoesnotexist  }}")
}

var dynamicWorkflowTemplateARefB = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-a
spec:
  templates:
  - name: template-a
    inputs:
      parameters:
        - name: message
    steps:
      - - name: step-a
          templateRef:
            name: workflow-template-b
            template: "{{ inputs.parameters.message }}"
`

var dynamicWorkflowTemplateRefB = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: workflow-template-b
spec:
  templates:
  - name: template-b
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello from template"]
`

var dynamicTemplateRefWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dynamic-workflow-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    steps:
      - - name: whalesay
          templateRef:
            name: workflow-template-a
            template: template-a
          arguments:
            parameters:
              - name: message
                value: "template-b"
`

func TestDynamicWorkflowTemplateRef(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(dynamicTemplateRefWorkflow)
	wftmplA := wfv1.MustUnmarshalWorkflowTemplate(dynamicWorkflowTemplateARefB)
	wftmplB := wfv1.MustUnmarshalWorkflowTemplate(dynamicWorkflowTemplateRefB)

	err := createWorkflowTemplate(ctx, wftmplA)
	require.NoError(t, err)
	err = createWorkflowTemplate(ctx, wftmplB)
	require.NoError(t, err)

	err = Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{})
	require.NoError(t, err)

	_ = deleteWorkflowTemplate(ctx, wftmplA.Name)
	_ = deleteWorkflowTemplate(ctx, wftmplB.Name)
}

var parameterizedGlobalArtifactsWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: test-parameterized-global-artifacts-
spec:
  entrypoint: main
  onExit: exit-handler
  arguments:
    parameters:
    - name: variable
      value: "car"
  templates:
  - name: main
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo 'test data' > /tmp/result.txt"]
    outputs:
      artifacts:
      - name: result
        globalName: output-result-{{workflow.parameters.variable}}
        path: /tmp/result.txt
        archive:
          none: {}
  - name: exit-handler
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo 'Access artifact: {{workflow.outputs.artifacts.output-result-car}}'"]
`

var parameterizedGlobalArtifactsWithWorkflowTemplateRef = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: test-wft-ref-
spec:
  onExit: exit-handler
  arguments:
    parameters:
    - name: variable
      value: "test"
  workflowTemplateRef:
    name: parameterized-artifacts-template
`

var parameterizedArtifactsWorkflowTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: parameterized-artifacts-template
spec:
  entrypoint: main
  onExit: exit-handler
  templates:
  - name: main
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo 'test data' > /tmp/result.txt"]
    outputs:
      artifacts:
      - name: result
        globalName: output-result-{{workflow.parameters.variable}}
        path: /tmp/result.txt
        archive:
          none: {}
  - name: exit-handler
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo 'Access artifact: {{workflow.outputs.artifacts.output-result-test}}'"]
`

func TestParameterizedGlobalArtifactsInExitHandler(t *testing.T) {
	// Test that parameterized global artifacts can be referenced in exit handlers
	err := validate(logging.TestContext(t.Context()), parameterizedGlobalArtifactsWorkflow)
	require.NoError(t, err)
}

func TestParameterizedGlobalArtifactsWithWorkflowTemplateRef(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// Create the workflow template with parameterized artifacts
	err := createWorkflowTemplateFromSpec(ctx, parameterizedArtifactsWorkflowTemplate)
	require.NoError(t, err)

	// Test that parameterized global artifacts from workflow template refs can be referenced in exit handlers
	err = validate(ctx, parameterizedGlobalArtifactsWithWorkflowTemplateRef)
	require.NoError(t, err)
}

var workflowWithoutParameterizedArtifacts = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: test-no-parameterized-
spec:
  entrypoint: main
  onExit: exit-handler
  templates:
  - name: main
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo 'test data' > /tmp/result.txt"]
    outputs:
      artifacts:
      - name: result
        globalName: simple-artifact
        path: /tmp/result.txt
        archive:
          none: {}
  - name: exit-handler
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo 'Access artifact: {{workflow.outputs.artifacts.nonexistent}}'"]
`

func TestWorkflowWithoutParameterizedArtifactsFails(t *testing.T) {
	// Test that referencing non-existent global artifacts still fails validation when there are no parameterized artifacts
	err := validate(logging.TestContext(t.Context()), workflowWithoutParameterizedArtifacts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to resolve {{workflow.outputs.artifacts.nonexistent}}")
}
