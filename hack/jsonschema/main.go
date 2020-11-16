package main

import (
	"encoding/json"
	"os"
)

func main() {
	swagger := map[string]interface{}{}
	{
		f, err := os.Open("api/openapi-spec/swagger.json")
		if err != nil {
			panic(err)
		}
		err = json.NewDecoder(f).Decode(&swagger)
		if err != nil {
			panic(err)
		}
	}
	{
		schema := map[string]interface{}{
			"$id":     "http://workflows.argoproj.io/workflows.json", // don't really know what this should be
			"$schema": "http://json-schema.org/schema#",
			"type":    "object",
			"oneOf": []interface{}{
				map[string]string{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplate"},
				map[string]string{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.CronWorkflow"},
				map[string]string{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.Workflow"},
				map[string]string{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.WorkflowEventBinding"},
				map[string]string{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.WorkflowTemplate"},
			},
			"definitions": swagger["definitions"],
		}
		f, err := os.Create("api/jsonschema/schema.json")
		if err != nil {
			panic(err)
		}
		e := json.NewEncoder(f)
		e.SetIndent("", "  ")
		err = e.Encode(schema)
		if err != nil {
			panic(err)
		}
		err = f.Close()
		if err != nil {
			panic(err)
		}
	}
}
