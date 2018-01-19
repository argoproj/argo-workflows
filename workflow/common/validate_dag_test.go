package common

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
	err := validate(dagCycle)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "cycle")
	}
}

var duplicateDependencies = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-dup-depends-
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
        template: echo
      - name: B
        dependencies: [A, A]
        template: echo
`

func TestDuplicateDependencies(t *testing.T) {
	err := validate(duplicateDependencies)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "duplicate")
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
	err := validate(dagUndefinedTemplate)
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
            value: "{{dependencies.A.outputs.parameters.unresolvable}}"
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
            value: "{{dependencies.A.outputs.parameters.hosts}}"
`

func TestDAGVariableResolution(t *testing.T) {
	err := validate(dagUnresolvedVar)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "failed to resolve {{dependencies.A.outputs.parameters.unresolvable}}")
	}
	err = validate(dagResolvedVar)
	assert.Nil(t, err)
}
