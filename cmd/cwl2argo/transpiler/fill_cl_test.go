package transpiler

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCLIConversion(t *testing.T) {
	dstr := `class: DockerRequirement
dockerPull: postgres/db
names:
  - key: value
`
	d := DockerRequirement{}
	err := yaml.Unmarshal([]byte(dstr), &d)

	yamlData := make(map[string]interface{})
	inputs := make(map[string]map[string]interface{})
	message := make(map[string]interface{})
	inputBinding := make(map[string]interface{})

	inputBinding["position"] = 1

	message["type"] = "string"
	message["inputBinding"] = inputBinding

	inputs["message"] = message

	yamlData["cwlVersion"] = "v1.0"
	yamlData["class"] = "CommandlineTool"
	yamlData["id"] = "main"
	yamlData["inputs"] = inputs
	yamlData["baseCommand"] = "echo"

	_, err = FillCommandlineTool(yamlData)
	if err != nil {
		t.Errorf("Was unable to convert dynamic yaml to <CommandlineTool>: %+v", err)
	}

}

func TestGenericStringFill(t *testing.T) {
	oldVal := exampleCLI1.Id
	var m map[string]interface{}
	m = make(map[string]interface{})
	key := "id"
	value := "#main"
	m[key] = value
	fillString(&exampleCLI1.Id, m, key)

	if exampleCLI1.Id == nil {
		t.Errorf("fillString was passed a value yet Id remained nil")
	}

	if *exampleCLI1.Id != m[key] {
		t.Errorf("fillString was passed %s but %s was set", value, *exampleCLI1.Id)
	}

	exampleCLI1.Id = oldVal

	oldVal = exampleCLI1.Label
	key = "label"
	value = "label_value"
	m[key] = value

	fillString(&exampleCLI1.Label, m, key)

	if exampleCLI1.Label == nil {
		t.Errorf("fillString was passed a value yet Label remained nil")
	}

	if *exampleCLI1.Label != m[key] {
		t.Errorf("fillString was passed %s but %s was set", value, *exampleCLI1.Label)
	}
	exampleCLI1.Label = oldVal
}
