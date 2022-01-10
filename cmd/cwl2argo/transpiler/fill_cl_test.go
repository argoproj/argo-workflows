package transpiler

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSimpleCWL(t *testing.T) {
	data := `cwlVersion: v1.2
class: CommandLineTool 
requirements: 
  - class: DockerRequirement 
    dockerPull: python:3.7
baseCommand: echo 
id: echo-tool 
inputs: 
  message:
    type: string 
    inputBinding:
      position: 1 
outputs: []
`

	var cliTool CommandlineTool
	err := yaml.Unmarshal([]byte(data), &cliTool)
	if err != nil {
		t.Error(err)
	}

}
