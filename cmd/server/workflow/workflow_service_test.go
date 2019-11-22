package workflow

import (
	"fmt"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
	"testing"
)

var wf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2019-09-16T22:56:45Z"
  generateName: scripts-bash-
  generation: 9
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Failed
  name: scripts-bash-5ksp4
  namespace: default
  resourceVersion: "1414877"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/default/workflows/scripts-bash-5ksp4
  uid: 41a16c4b-d8d5-11e9-8938-025000000001
spec:
  arguments: {}
  entrypoint: bash-script-example
  templates:
  - arguments: {}
    inputs: {}
    metadata: {}
    name: bash-script-example
    outputs: {}
    steps:
    - - arguments: {}
        name: generate
        template: gen-random-int
    - - arguments:
          parameters:
          - name: message
            value: '{{steps.generate.outputs.result}}'
        name: print
        template: print-message
  - arguments: {}
    inputs: {}
    metadata: {}
    name: gen-random-int
    outputs: {}
    script:
      command:
      - bash
      image: debian:9.4
      name: ""
      resources: {}
      source: |
        cat /dev/urandom | od -N2 -An -i | awk -v f=1 -v r=100 '{printf "%i\n", f + r * $1 / 65536}'
  - arguments: {}
    container:
      args:
      - 'echo -e " apiVersion: policy/v1beta1 kind: PodDisruptionBudget metadata:
        name: zk-pdb spec: minAvailable: 2 selector: matchLabels: workflows.argoproj.io/workflow:
        {{workflow.name}} " | tee pdb.yaml |sleep 120|kubectl create -f pdb.yaml '
      command:
      - sh
      - -c
      image: lachlanevenson/k8s-kubectl
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: message
    metadata: {}
    name: print-message
    outputs: {}
status:
  finishedAt: "2019-09-16T22:58:59Z"
  message: child 'scripts-bash-5ksp4-1961198978' failed
  nodes:
    scripts-bash-5ksp4:
      children:
      - scripts-bash-5ksp4-2570590690
      displayName: scripts-bash-5ksp4
      finishedAt: "2019-09-16T22:58:59Z"
      id: scripts-bash-5ksp4
      message: child 'scripts-bash-5ksp4-1961198978' failed
      name: scripts-bash-5ksp4
      outboundNodes:
      - scripts-bash-5ksp4-1961198978
      phase: Failed
      startedAt: "2019-09-16T22:56:45Z"
      templateName: bash-script-example
      type: Steps
    scripts-bash-5ksp4-315841411:
      boundaryID: scripts-bash-5ksp4
      children:
      - scripts-bash-5ksp4-3576997567
      displayName: generate
      finishedAt: "2019-09-16T22:56:51Z"
      id: scripts-bash-5ksp4-315841411
      name: scripts-bash-5ksp4[0].generate
      outputs:
        result: "50"
      phase: Succeeded
      startedAt: "2019-09-16T22:56:45Z"
      templateName: gen-random-int
      type: Pod
    scripts-bash-5ksp4-1961198978:
      boundaryID: scripts-bash-5ksp4
      displayName: print
      finishedAt: "2019-09-16T22:58:58Z"
      id: scripts-bash-5ksp4-1961198978
      inputs:
        parameters:
        - name: message
          value: "50"
      message: failed with exit code 1
      name: scripts-bash-5ksp4[1].print
      phase: Failed
      startedAt: "2019-09-16T22:56:53Z"
      templateName: print-message
      type: Pod
    scripts-bash-5ksp4-2570590690:
      boundaryID: scripts-bash-5ksp4
      children:
      - scripts-bash-5ksp4-315841411
      displayName: '[0]'
      finishedAt: "2019-09-16T22:56:53Z"
      id: scripts-bash-5ksp4-2570590690
      name: scripts-bash-5ksp4[0]
      phase: Succeeded
      startedAt: "2019-09-16T22:56:45Z"
      templateName: bash-script-example
      type: StepGroup
    scripts-bash-5ksp4-3576997567:
      boundaryID: scripts-bash-5ksp4
      children:
      - scripts-bash-5ksp4-1961198978
      displayName: '[1]'
      finishedAt: "2019-09-16T22:58:59Z"
      id: scripts-bash-5ksp4-3576997567
      message: child 'scripts-bash-5ksp4-1961198978' failed
      name: scripts-bash-5ksp4[1]
      phase: Failed
      startedAt: "2019-09-16T22:56:53Z"
      templateName: bash-script-example
      type: StepGroup
  phase: Failed
  startedAt: "2019-09-16T22:56:45Z"
`

func unmarshalWF(yamlStr string) *wfv1.Workflow {
	var wf wfv1.Workflow
	err := yaml.Unmarshal([]byte(yamlStr), &wf)
	if err != nil {
		panic(err)
	}
	return &wf
}

func TestMarshalling(t *testing.T) {

	workf := unmarshalWF(wf)

	wr := WorkflowResponse{Workflows: workf}
	bytes, err := wr.Marshal()
	if err != nil {

	}
	wr1 := WorkflowResponse{}
	wr1.Unmarshal(bytes)
	fmt.Println(wr1)
	assert.Equal(t, wr, wr1)

}
