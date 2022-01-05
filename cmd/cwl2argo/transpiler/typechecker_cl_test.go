package transpiler

import "testing"

var (
	exampleCLI1Id      = "exampleCLI1"
	dockerRequirements = []CWLRequirements{DockerRequirement{}}
)

var exampleCLI1 = CommandlineTool{
	Inputs:       make([]CommandlineInputParameter, 0),
	Outputs:      make([]CommandlineOutputParameter, 0),
	Class:        "CommandlineTool",
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
	err := TypeCheckCommandlineTool(exampleCLI1, make(map[interface{}]interface{}))
	if err == nil {
		t.Errorf("Failed to type check: %s", err)
	}
	exampleCLI1.Requirements = dockerRequirements

	err = TypeCheckCommandlineTool(exampleCLI1, map[interface{}]interface{}{})
	if err != nil {
		t.Errorf("Failed to type check: %s", err)
	}
}
