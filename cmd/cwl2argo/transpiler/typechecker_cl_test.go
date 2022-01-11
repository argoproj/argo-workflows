package transpiler

import "testing"

var (
	exampleCLI1Id      = "exampleCLI1"
	dockerPull         = "python:3.7"
	dockerRequirements = []CWLRequirements{DockerRequirement{DockerPull: &dockerPull}}
)

var exampleCLI1 = CommandlineTool{
	Inputs:       make([]CommandlineInputParameter, 0),
	Outputs:      make([]CommandlineOutputParameter, 0),
	Class:        "CommandLineTool",
	Id:           &exampleCLI1Id,
	Doc:          make([]string, 0),
	Requirements: make([]CWLRequirements, 0),
	Hints:        make([]interface{}, 0),
	CWLVersion:   nil,
	Intent:       make([]string, 0),
	BaseCommand:  []string{"echo", "hello world"},
	Arguments:    make([]CommandlineArgument, 0),
	Stdin:        nil,
	Stderr:       nil,
	Stdout:       nil,
}

func TestCLIRequirementTypeChecking(t *testing.T) {
	err := TypeCheckCommandlineTool(&exampleCLI1, make(map[string]interface{}))
	if err == nil {
		t.Errorf("Failed to type check: %s", err)
	}
	exampleCLI1.Requirements = dockerRequirements

	err = TypeCheckCommandlineTool(&exampleCLI1, map[string]interface{}{})
	if err != nil {
		t.Errorf("Failed to type check: %s", err)
	}

	oldReqs := exampleCLI1.Requirements
	exampleCLI1.Requirements = nil
	err = TypeCheckCommandlineTool(&exampleCLI1, make(map[string]interface{}))
	if err == nil {
		t.Errorf("Expected <%s> but got nil", errorDockerRequirement(exampleCLI1.Id))
	}
	exampleCLI1.Requirements = oldReqs
}
