package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/go-openapi/jsonreference"
	"github.com/go-openapi/spec"

	"github.com/argoproj/argo/hack/jsonschema/k8s"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func main() {
	pleasentName := func(n string) string {
		return strings.Replace(
			strings.Replace(
				strings.ReplaceAll(n, "/", "."),
				"github.com.argoproj.argo.pkg.apis",
				"io.argoproj",
				1,
			),
			"k8s.io",
			"io.k8s",
			1,
		)
	}
	objectify := func(x interface{}) obj {
		data, err := json.Marshal(x)
		if err != nil {
			panic(err)
		}
		y := obj{}
		err = json.Unmarshal(data, &y)
		if err != nil {
			panic(err)
		}
		return y
	}
	definitions := make(map[string]obj)
	for n, s := range wfv1.GetOpenAPIDefinitions(func(path string) spec.Ref {
		return spec.Ref{Ref: jsonreference.MustCreateRef("#/definitions/" + pleasentName(path))}
	}) {
		definitions[pleasentName(n)] = objectify(s.Schema)
	}
	for n, s := range k8s.GetOpenAPIDefinitions(func(path string) spec.Ref {
		return spec.Ref{Ref: jsonreference.MustCreateRef("#/definitions/" + pleasentName(path))}
	}) {
		definitions[pleasentName(n)] = objectify(s.Schema)
	}
	for n, v := range definitions {
		println(n)
		if p, ok := v["properties"]; ok {
			for _, v := range p.(obj) {
				switch v.(obj)["format"] {
				case "int32", "int64":
					delete(v.(obj), "format")
				}
			}
		}
		kind := strings.TrimPrefix(n, "io.argoproj.workflow.v1alpha1.")
		switch kind {
		case "CronWorkflow", "ClusterWorkflowTemplate", "Workflow", "WorkflowEventBinding", "WorkflowTemplate":
			v["properties"].(obj)["apiVersion"].(obj)["const"] = "argoproj.io/v1alpha1"
			v["properties"].(obj)["kind"].(obj)["const"] = kind
		}
		switch n {
		case "io.k8s.apimachinery.pkg.util.intstr.IntOrString":
			v["type"] = array{"string", "integer"}
			delete(v, "format")
		case "io.argoproj.workflow.v1alpha1.CronWorkflow":
			v["required"] = array{"metadata", "spec"}
		case "io.argoproj.workflow.v1alpha1.Workflow":
			v["required"] = array{"metadata", "spec"}
		case "io.argoproj.workflow.v1alpha1.ScriptTemplate":
			v["required"] = array{"image", "source"}
		case "io.k8s.api.core.v1.Container":
			v["required"] = array{"image"}
		}
		definitions[n] = v
	}
	delete(definitions, "io.k8s.apimachinery.pkg.apis.meta.v1.InternalEvent")
	{
		schema := obj{
			"$id":     "http://workflows.argoproj.io/workflows.json", // don't really know what this should be
			"$schema": "http://json-schema.org/schema#",
			"type":    "object",
			"oneOf": []interface{}{
				obj{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplate"},
				obj{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.CronWorkflow"},
				obj{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.Workflow"},
				obj{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.WorkflowEventBinding"},
				obj{"$ref": "#/definitions/io.argoproj.workflow.v1alpha1.WorkflowTemplate"},
			},
			"definitions": definitions,
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
