package v1alpha1

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// MustUnmarshal is a utility function to unmarshall either a file, byte array, or string of JSON or YAMl into a object.
// text - a byte array or string, if starts with "@" it assumed to be a file and read from disk, is starts with "{" assumed to be JSON, otherwise assumed to be YAML
// v - a pointer to an object
func MustUnmarshal(text, v interface{}) {
	if err := Unmarshal(text, v); err != nil {
		panic(err)
	}
}

// Unmarshal is a utility function to unmarshall either a file, byte array, or string of JSON or YAMl into a object.
// same as MustUnmarshal, but returns an error
func Unmarshal(text, v interface{}) error {
	switch x := text.(type) {
	case string:
		return Unmarshal([]byte(x), v)
	case []byte:
		if len(x) == 0 {
			return fmt.Errorf("no text to unmarshal")
		}
		switch x[0] {
		case '@':
			filename := string(x[1:])
			y, err := os.ReadFile(filepath.Clean(filename))
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", filename, err)
			}
			return Unmarshal(y, v)
		case '{':
			if err := json.Unmarshal(x, v); err != nil {
				return fmt.Errorf("failed to unmarshal JSON %q: %w", string(x), err)
			}
		default:
			if err := yaml.UnmarshalStrict(x, v); err != nil {
				return fmt.Errorf("failed to unmarshal YAML %q: %w", string(x), err)
			}
		}
	default:
		return fmt.Errorf("cannot unmarshal type %T", text)
	}
	return nil
}

func MustMarshallJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func MustUnmarshalClusterWorkflowTemplate(text interface{}) *ClusterWorkflowTemplate {
	x := &ClusterWorkflowTemplate{}
	MustUnmarshal(text, &x)
	return x
}

func MustUnmarshalCronWorkflow(text interface{}) *CronWorkflow {
	x := &CronWorkflow{}
	MustUnmarshal(text, &x)
	return x
}

func MustUnmarshalTemplate(text interface{}) *Template {
	x := &Template{}
	MustUnmarshal(text, &x)
	return x
}

func MustUnmarshalWorkflow(text interface{}) *Workflow {
	x := &Workflow{}
	MustUnmarshal(text, &x)
	return x
}

func MustUnmarshalWorkflowTemplate(text interface{}) *WorkflowTemplate {
	x := &WorkflowTemplate{}
	MustUnmarshal(text, &x)
	return x
}

func MustUnmarshalWorkflowArtifactGCTask(text interface{}) *WorkflowArtifactGCTask {
	x := &WorkflowArtifactGCTask{}
	MustUnmarshal(text, &x)
	return x
}
