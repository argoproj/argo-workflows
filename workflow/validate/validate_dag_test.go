package validate

import (
	"testing"

	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

var dagCycle = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-cycle-
spec:
  entrypoint: cycle
  templates:
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
  - name: cycle
    dag:
      tasks:
      - name: A
        dependencies: [C]
        template: echo
      - name: B
        dependencies: [A]
        template: echo
      - name: C
        dependencies: [A]
        template: echo
`

func TestDAGCycle(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagCycle)
	require.ErrorContains(t, err, "cycle")
}

var dagAnyWithoutExpandingTask = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-cycle-
spec:
  entrypoint: entry
  templates:
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
  - name: entry
    dag:
      tasks:
      - name: A
        template: echo
      - name: B
        depends: A.AnySucceeded
        template: echo
`

func TestAnyWithoutExpandingTask(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagAnyWithoutExpandingTask)
	require.ErrorContains(t, err, "does not contain any items")
}

var dagUndefinedTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-undefined-
spec:
  entrypoint: undef
  templates:
  - name: undef
    dag:
      tasks:
      - name: A
        template: echo
`

func TestDAGUndefinedTemplate(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagUndefinedTemplate)
	require.ErrorContains(t, err, "undefined")
}

var dagUnresolvedVar = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-cycle-
spec:
  entrypoint: unresolved
  templates:
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.message}}"]
    outputs:
      parameters:
      - name: hosts
        valueFrom:
          path: /etc/hosts
  - name: unresolved
    dag:
      tasks:
      - name: A
        template: echo
        arguments:
          parameters:
          - name: message
            value: val
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.A.outputs.parameters.unresolvable}}"
`

var dagResolvedVar = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-cycle-
spec:
  entrypoint: unresolved
  templates:
  - name: echo
    inputs:
      parameters:
      - name: message
      - name: startedat
      - name: finishedat
      - name: id
      - name: hostnodename
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.message}}"]
    outputs:
      parameters:
      - name: hosts
        valueFrom:
          path: /etc/hosts
  - name: unresolved
    dag:
      tasks:
      - name: A
        template: echo
        arguments:
          parameters:
          - name: message
            value: val
          - name: startedat
            value: "test"
          - name: finishedat
            value: "test"
          - name: id
            value: "1"
          - name: hostnodename
            value: "test"
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.A.outputs.parameters.hosts}}"
          - name: startedat
            value: "{{tasks.A.startedAt}}"
          - name: finishedat
            value: "{{tasks.A.finishedAt}}"
          - name: id
            value: "{{tasks.A.id}}"
          - name: hostnodename
            value: "{{tasks.A.hostNodeName}}"
      - name: C
        dependencies: [B]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.A.outputs.parameters.hosts}}"
          - name: startedat
            value: "{{tasks.A.startedAt}}"
          - name: finishedat
            value: "{{tasks.A.finishedAt}}"
          - name: id
            value: "{{tasks.A.id}}"
          - name: hostnodename
            value: "{{tasks.A.hostNodeName}}"
`

var dagResolvedVarNotAncestor = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-cycle-
spec:
  entrypoint: unresolved
  templates:
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.message}}"]
    outputs:
      parameters:
      - name: hosts
        valueFrom:
          path: /etc/hosts
  - name: unresolved
    dag:
      tasks:
      - name: A
        template: echo
        arguments:
          parameters:
          - name: message
            value: val
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.A.outputs.parameters.hosts}}"
      - name: C
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.B.outputs.parameters.hosts}}"
`

var dagResolvedGlobalVar = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-global-var-
spec:
  entrypoint: unresolved
  templates:
  - name: first
    container:
      image: alpine:3.23
    outputs:
      parameters:
      - name: hosts
        valueFrom:
          path: /etc/hosts
        globalName: global
  - name: second
    container:
      image: alpine:3.23
      command: [echo, "{{workflow.outputs.parameters.global}}"]
  - name: unresolved
    dag:
      tasks:
      - name: A
        template: first
      - name: B
        dependencies: [A]
        template: second
`

var dagResolvedGlobalVarReversed = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-global-var-
spec:
  entrypoint: unresolved
  templates:
  - name: first
    container:
      image: alpine:3.23
    outputs:
      parameters:
      - name: hosts
        valueFrom:
          path: /etc/hosts
        globalName: global
  - name: second
    container:
      image: alpine:3.23
      command: [echo, "{{workflow.outputs.parameters.global}}"]
  - name: unresolved
    dag:
      tasks:
      - name: B
        dependencies: [A]
        template: second
      - name: A
        template: first
`

func TestDAGVariableResolution(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	err := validate(ctx, dagUnresolvedVar)
	require.ErrorContains(t, err, "failed to resolve {{tasks.A.outputs.parameters.unresolvable}}")

	err = validate(ctx, dagResolvedVar)
	require.NoError(t, err)

	err = validate(ctx, dagResolvedVarNotAncestor)
	require.ErrorContains(t, err, "templates.unresolved.tasks.C missing dependency 'B' for parameter 'message'")

	err = validate(ctx, dagResolvedGlobalVar)
	require.NoError(t, err)
	err = validate(ctx, dagResolvedGlobalVarReversed)
	require.NoError(t, err)
}

var dagResolvedArt = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-arg-passing-
spec:
  entrypoint: dag-arg-passing
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

  - name: dag-arg-passing
    dag:
      tasks:
      - name: A
        template: generate
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: val
          artifacts:
          - name: passthrough
            from: "{{tasks.A.outputs.artifacts.generated_hosts}}"
`

func TestDAGArtifactResolution(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagResolvedArt)
	require.NoError(t, err)
}

var dagStatusReference = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-arg-passing-
spec:
  entrypoint: dag-arg-passing
  templates:
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.message}}"]

  - name: dag-arg-passing
    dag:
      tasks:
      - name: A
        template: echo
        continueOn:
          failed: true
        arguments:
          parameters:
          - name: message
            value: "Hello!"
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.A.status}}"
`

var dagStatusNoFutureReferenceSimple = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-arg-passing-
spec:
  entrypoint: dag-arg-passing
  templates:
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.message}}"]

  - name: dag-arg-passing
    dag:
      tasks:
      - name: A
        template: echo
        continueOn:
          failed: true
        arguments:
          parameters:
          - name: message
            value: "{{tasks.B.status}}"
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.A.status}}"
`

var dagStatusNoFutureReferenceWhenFutureReferenceHasChild = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-arg-passing-
spec:
  entrypoint: dag-arg-passing
  templates:
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.message}}"]

  - name: dag-arg-passing
    dag:
      tasks:
      - name: A
        template: echo
        continueOn:
          failed: true
        arguments:
          parameters:
          - name: message
            value: "{{tasks.B.status}}"
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.A.status}}"
      - name: C
        dependencies: [B]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.B.status}}"
`

var dagStatusPastReferenceChain = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-arg-passing-
spec:
  entrypoint: dag-arg-passing
  templates:
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.message}}"]

  - name: dag-arg-passing
    dag:
      tasks:
      - name: A
        template: echo
        continueOn:
          failed: true
        arguments:
          parameters:
          - name: message
            value: "Hello"
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.A.status}}"
      - name: C
        dependencies: [B]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.B.status}}"
      - name: D
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.A.status}}"
      - name: E
        dependencies: [D]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.D.status}}"
`

var dagStatusOnlyDirectAncestors = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-arg-passing-
spec:
  entrypoint: dag-arg-passing
  templates:
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.message}}"]

  - name: dag-arg-passing
    dag:
      tasks:
      - name: A
        template: echo
        continueOn:
          failed: true
        arguments:
          parameters:
          - name: message
            value: "Hello"
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.A.status}}"
      - name: C
        dependencies: [B]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.B.status}}"
      - name: D
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.A.status}}"
      - name: E
        dependencies: [D]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.B.status}}"
`

func TestDAGStatusReference(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	err := validate(ctx, dagStatusReference)
	require.NoError(t, err)

	err = validate(ctx, dagStatusNoFutureReferenceSimple)
	// Can't reference the status of steps that have not run yet
	require.ErrorContains(t, err, "failed to resolve {{tasks.B.status}}")

	err = validate(ctx, dagStatusNoFutureReferenceWhenFutureReferenceHasChild)
	// Can't reference the status of steps that have not run yet, even if the referenced steps have children
	require.ErrorContains(t, err, "failed to resolve {{tasks.B.status}}")

	err = validate(ctx, dagStatusPastReferenceChain)
	require.NoError(t, err)

	err = validate(ctx, dagStatusOnlyDirectAncestors)
	// Can't reference steps that are not direct ancestors of node
	// Here Node E references the status of Node B, even though it is not its descendent
	require.ErrorContains(t, err, "failed to resolve {{tasks.B.status}}")
}

var dagNonexistantTarget = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-target-
spec:
  entrypoint: dag-target
  templates:
  - name: dag-target
    dag:
      target: DOESNTEXIST
      tasks:
      - name: A
        template: echo
        arguments:
          parameters: [{name: message, value: A}]
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters: [{name: message, value: B}]
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.message}}"]
`

func TestDAGNonExistantTarget(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagNonexistantTarget)
	require.ErrorContains(t, err, "target 'DOESNTEXIST' is not defined")

}

var dagTargetSubstitution = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-target-
spec:
  entrypoint: dag-target
  arguments:
    parameters:
    - name: target
      value: B
  templates:
  - name: dag-target
    dag:
      target: "{{workflow.parameters.target}}"
      tasks:
      - name: A
        template: echo
        arguments:
          parameters: [{name: message, value: A}]
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters: [{name: message, value: B}]
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.message}}"]
`

func TestDAGTargetSubstitution(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagTargetSubstitution)
	require.NoError(t, err)
}

var dagTargetMissingInputParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-target-
spec:
  entrypoint: dag-target
  arguments:
    parameters:
    - name: target
      value: A
  templates:
  - name: dag-target
    dag:
      target: "{{inputs.parameters.target}}"
      tasks:
      - name: A
        template: echo
        arguments:
          parameters: [{name: message, value: A}]
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.message}}"]
`

func TestDAGTargetMissingInputParam(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagTargetMissingInputParam)
	require.Error(t, err)
}

var dagDependsAndDependencies = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-target-
spec:
  entrypoint: dag-target
  templates:
  - name: dag-target
    dag:
      tasks:
      - name: A
        template: echo
      - name: B
        dependencies: [A]
        template: echo
      - name: C
        depends: "B"
        template: echo

  - name: echo
    container:
      image: alpine:3.23
      command: [echo, "hello"]
`

func TestDependsAndDependencies(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagDependsAndDependencies)
	require.ErrorContains(t, err, "templates.dag-target cannot use both 'depends' and 'dependencies' in the same DAG template")
}

var dagDependsAndContinueOn = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-target-
spec:
  entrypoint: dag-target
  templates:
  - name: dag-target
    dag:
      tasks:
      - name: A
        template: echo
      - name: B
        continueOn:
          failed: true
        template: echo
      - name: C
        depends: "B"
        template: echo

  - name: echo
    container:
      image: alpine:3.23
      command: [echo, "hello"]
`

func TestDependsAndContinueOn(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagDependsAndContinueOn)
	require.ErrorContains(t, err, "templates.dag-target cannot use 'continueOn' when using 'depends'. Instead use 'dep-task.Failed'/'dep-task.Errored'")
}

var dagDependsDigit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-diamond-
spec:
  entrypoint: diamond
  templates:
    - name: diamond
      dag:
        tasks:
          - name: 5A
            template: pass
          - name: B
            depends: 5A
            template: pass
          - name: C
            depends: 5A
            template: fail
          - name: should-execute-1
            depends: "'5A' && (C.Succeeded || C.Failed)"   # For more information about this depends field, see: docs/enhanced-depends-logic.md
            template: pass
          - name: should-execute-2
            depends: B || C
            template: pass
          - name: should-not-execute
            depends: B && C
            template: pass
          - name: should-execute-3
            depends: should-execute-2.Succeeded || should-not-execute
            template: pass
    - name: pass
      container:
        image: alpine:3.23
        command:
          - sh
          - -c
          - exit 0
    - name: fail
      container:
        image: alpine:3.23
        command:
          - sh
          - -c
          - exit 1
`

func TestDAGDependsDigit(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagDependsDigit)
	require.ErrorContains(t, err, "templates.diamond.tasks.5A name cannot begin with a digit when using either 'depends' or 'dependencies'")
}

var dagDependenciesDigit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-diamond-
spec:
  entrypoint: diamond
  templates:
    - name: diamond
      dag:
        tasks:
          - name: 5A
            template: pass
          - name: B
            dependencies: [5A]
            template: pass
          - name: C
            dependencies: [5A]
            template: fail
          - name: should-execute-1
            depends: "'5A' && (C.Succeeded || C.Failed)"   # For more information about this depends field, see: docs/enhanced-depends-logic.md
            template: pass
          - name: should-execute-2
            depends: B || C
            template: pass
          - name: should-not-execute
            depends: B && C
            template: pass
          - name: should-execute-3
            depends: should-execute-2.Succeeded || should-not-execute
            template: pass
    - name: pass
      container:
        image: alpine:3.23
        command:
          - sh
          - -c
          - exit 0
    - name: fail
      container:
        image: alpine:3.23
        command:
          - sh
          - -c
          - exit 1
`

func TestDAGDependenciesDigit(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagDependenciesDigit)
	require.ErrorContains(t, err, "templates.diamond.tasks.5A name cannot begin with a digit when using either 'depends' or 'dependencies'")
}

var dagWithDigitNoDepends = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-diamond-
spec:
  entrypoint: diamond
  templates:
    - name: diamond
      dag:
        tasks:
          - name: 5A
            template: pass
          - name: B
            template: pass
    - name: pass
      container:
        image: alpine:3.23
        command:
          - sh
          - -c
          - exit 0
`

func TestDAGWithDigitNameNoDepends(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagWithDigitNoDepends)
	require.NoError(t, err)
}

var dagOutputsResolveTaskAggregatedOutputs = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: loops-
spec:
  serviceAccountName: default
  entrypoint: dag
  templates:
  - name: dag
    dag:
      tasks:
      - name: fanout
        template: fanout
        arguments:
          parameters:
          - name: input
            value: "[1, 2]"
      - name: dag-process
        template: sub-dag
        depends: fanout
        arguments:
          parameters:
          - name: item
            value: '{{item}}' 
          - name: input
            value: '{{tasks.fanout.outputs.parameters.output}}'
        withParam: "{{tasks.fanout.outputs.parameters.output}}"

  - name: sub-dag
    inputs:
      parameters:
      - name: input
      - name: item
    outputs:
      parameters:
      - name: output
        valueFrom:
          parameter: "{{tasks.process.outputs.parameters}}"
    dag:
      tasks:
      - name: fanout
        template: fanout
        arguments:
          parameters:
          - name: input
            value: '{{inputs.parameters.input}}'
      - name: process
        template: process
        depends: fanout
        arguments:
          parameters:
          - name: item
            value: '{{item}}'
        withParam: "{{tasks.fanout.outputs.parameters.output}}"

  - name: fanout
    inputs:
      parameters:
      - name: input
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["echo {{inputs.parameters.input}} | tee /tmp/output"]
    outputs:
      parameters:
      - name: output
        valueFrom:
          path: /tmp/output

  - name: process
    inputs:
      parameters:
      - name: item
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["echo {{inputs.parameters.item}} | tee /tmp/output"]
    outputs:
      parameters:
      - name: output
        valueFrom:
          path: /tmp/output
`

func TestDAGOutputsResolveTaskAggregatedOutputs(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagOutputsResolveTaskAggregatedOutputs)
	require.NoError(t, err)
}

var dagMissingParamValueInTask = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec:
  entrypoint: root
  templates:
    - name: template
      inputs:
        parameters:
          - name: data
      container:
        name: main
        image: alpine
    - name: root
      inputs:
        parameters:
          - name: anything_param
      dag:
        tasks:
          - name: task
            template: template
            arguments:
              parameters:
                - name: data
                  valueFrom:
                    parameter: "{{inputs.parameters.anything_param}}"
  arguments:
    parameters:
      - name: anything_param
        value: anything_param
`

func TestDAGMissingParamValueInTask(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagMissingParamValueInTask)
	require.ErrorContains(t, err, ".valueFrom only allows: default, configMapKeyRef and supplied")
}

var dagArgParamValueFromConfigMapInTask = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec:
  entrypoint: root
  templates:
    - name: template
      inputs:
        parameters:
          - name: data
      container:
        name: main
        image: alpine
    - name: root
      dag:
        tasks:
          - name: task
            template: template
            arguments:
              parameters:
                - name: data
                  valueFrom:
                    configMapKeyRef:
                      name: my-config
                      key: my-data
                    default: my-default
`

func TestDAGArgParamValueFromConfigMapInTask(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), dagArgParamValueFromConfigMapInTask)
	require.NoError(t, err)
}

var failDagArgParamValueFromPathInTask = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec:
  entrypoint: root
  templates:
    - name: template
      inputs:
        parameters:
          - name: data
      container:
        name: main
        image: alpine
    - name: root
      dag:
        tasks:
          - name: task
            template: template
            arguments:
              parameters:
                - name: data
                  valueFrom:
                    path: /tmp/my-path
`

func TestFailDAGArgParamValueFromPathInTask(t *testing.T) {
	err := validate(logging.TestContext(t.Context()), failDagArgParamValueFromPathInTask)
	require.ErrorContains(t, err, "valueFrom only allows: default, configMapKeyRef and supplied")
}

var dagWithItemTemplateRefTmpl = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: 363-test-tmp
  namespace: default
spec:
  templates:
    - name: 363-test-tmp
      nodeSelector:
        nodegroup: arm-spot
      inputs:
        parameters:
          - name: path
          - name: service
          - name: build_arg
          - name: run_on
          - name: arch
          - name: docker_org
      container:
        image: alpine
        command:
          - sh
          - -c
          - |
            echo "path: {{inputs.parameters.path}}"
            echo "service: {{inputs.parameters.service}}"
            echo "build_arg: {{inputs.parameters.build_arg}}"
            echo "run_on: {{inputs.parameters.run_on}}"
            echo "arch: {{inputs.parameters.arch}}"
            echo "docker_org: {{inputs.parameters.docker_org}}"
`

var dagWithItemTemplateRefWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: 363-test-
  namespace: default
spec:
  activeDeadlineSeconds: 10800
  entrypoint: main
  serviceAccountName: ci

  templates:
    - name: main
      dag:
        tasks:
          - name: withitems
            templateRef:
              name: 363-test-tmp
              template: 363-test-tmp
            arguments:
              parameters:
                - name: path
                  value: "{{item.path}}"
                - name: service
                  value: "{{item.service}}"
                - name: build_arg
                  value: "{{item.arg}}"
                - name: run_on
                  value: "{{item.run_on}}"
                - name: arch
                  value: "{{item.arch}}"
                - name: docker_org
                  value: "{{item.docker_org}}"
            withItems:
              - {
                  path: "services",
                  service: "id",
                  arg: "",
                  run_on: "arm-spot",
                  arch: "arm64",
                  docker_org: "pipekit13",
                }
              - {
                  path: "services",
                  service: "events-handler",
                  arg: "",
                  run_on: "arm-spot",
                  arch: "arm64",
                  docker_org: "pipekit13",
                }
`

func TestDagWithItemTemplateRefTmpl(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(dagWithItemTemplateRefWf)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(dagWithItemTemplateRefTmpl)

	err := createWorkflowTemplate(ctx, wftmpl)
	require.NoError(t, err)

	err = Workflow(ctx, wftmplGetter, cwftmplGetter, wf, nil, Opts{})
	require.NoError(t, err)

	_ = deleteWorkflowTemplate(ctx, wftmpl.Name)
}
