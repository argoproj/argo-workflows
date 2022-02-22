package v1alpha1

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// MustUnmarshal is a utility function to unmarshall either a file, byte array, or string of JSON or YAMl into a object.
// text - a byte array or string, if starts with "@" it assumed to be a file and read from disk, is starts with "{" assumed to be JSON, otherwise assumed to be YAML
// v - a pointer to an object
func MustUnmarshal(text, v interface{}) {
	switch x := text.(type) {
	case string:
		MustUnmarshal([]byte(x), v)
	case []byte:
		if len(x) == 0 {
			panic("no text to unmarshal")
		}
		if x[0] == '@' {
			filename := string(x[1:])
			y, err := ioutil.ReadFile(filepath.Clean(filename))
			if err != nil {
				panic(fmt.Errorf("failed to read file %s: %w", filename, err))
			}
			MustUnmarshal(y, v)
		} else if x[0] == '{' {
			if err := json.Unmarshal(x, v); err != nil {
				panic(fmt.Errorf("failed to unmarshal JSON %q: %w", string(x), err))
			}
		} else {
			if err := yaml.UnmarshalStrict(x, v); err != nil {
				panic(fmt.Errorf("failed to unmarshal YAML %q: %w", string(x), err))
			}
		}
	default:
		panic(fmt.Errorf("cannot unmarshal type %T", text))
	}
}

func MustMarshallJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func MustUnmarshalClusterWorkflow(text interface{}) *ClusterWorkflowTemplate {
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
