package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
      image: alpine:3.7
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
	_, err := validate(dagCycle)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cycle")
	}
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
	_, err := validate(dagUndefinedTemplate)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "undefined")
	}
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
      image: alpine:3.7
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
    container:
      image: alpine:3.7
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
        dependencies: [B]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "{{tasks.A.outputs.parameters.hosts}}"
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
      image: alpine:3.7
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
            value: "{{tasks.B.outputs.parameters.unresolvable}}"
`

func TestDAGVariableResolution(t *testing.T) {
	_, err := validate(dagUnresolvedVar)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "failed to resolve {{tasks.A.outputs.parameters.unresolvable}}")
	}
	_, err = validate(dagResolvedVar)
	assert.NoError(t, err)

	_, err = validate(dagResolvedVarNotAncestor)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "failed to resolve {{tasks.B.outputs.parameters.unresolvable}}")
	}
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
	_, err := validate(dagResolvedArt)
	assert.NoError(t, err)
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
      image: alpine:3.7
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
      image: alpine:3.7
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
      image: alpine:3.7
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
      image: alpine:3.7
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
      image: alpine:3.7
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
	_, err := validate(dagStatusReference)
	assert.NoError(t, err)

	_, err = validate(dagStatusNoFutureReferenceSimple)
	// Can't reference the status of steps that have not run yet
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "failed to resolve {{tasks.B.status}}")
	}

	_, err = validate(dagStatusNoFutureReferenceWhenFutureReferenceHasChild)
	// Can't reference the status of steps that have not run yet, even if the referenced steps have children
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "failed to resolve {{tasks.B.status}}")
	}

	_, err = validate(dagStatusPastReferenceChain)
	assert.NoError(t, err)

	_, err = validate(dagStatusOnlyDirectAncestors)
	// Can't reference steps that are not direct ancestors of node
	// Here Node E references the status of Node B, even though it is not its descendent
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "failed to resolve {{tasks.B.status}}")
	}
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
      image: alpine:3.7
      command: [echo, "{{inputs.parameters.message}}"]
`

func TestDAGNonExistantTarget(t *testing.T) {
	_, err := validate(dagNonexistantTarget)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "target 'DOESNTEXIST' is not defined")
	}
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
      image: alpine:3.7
      command: [echo, "{{inputs.parameters.message}}"]
`

func TestDAGTargetSubstitution(t *testing.T) {
	_, err := validate(dagTargetSubstitution)
	assert.NoError(t, err)
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
      image: alpine:3.7
      command: [echo, "{{inputs.parameters.message}}"]
`

func TestDAGTargetMissingInputParam(t *testing.T) {
	_, err := validate(dagTargetMissingInputParam)
	assert.NotNil(t, err)
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
      image: alpine:3.7
      command: [echo, "hello"]
`

func TestDependsAndDependencies(t *testing.T) {
	_, err := validate(dagDependsAndDependencies)
	assert.Error(t, err, "templates.dag-target cannot use both 'depends' and 'dependencies' in the same DAG template")
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
      image: alpine:3.7
      command: [echo, "hello"]
`

func TestDependsAndContinueOn(t *testing.T) {
	_, err := validate(dagDependsAndContinueOn)
	assert.Error(t, err, "templates.dag-target cannot use 'continueOn' when using 'depends'. Instead use 'dep-task.Failed'/'dep-task.Errored'")
}
