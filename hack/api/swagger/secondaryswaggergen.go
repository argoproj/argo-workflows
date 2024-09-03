package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/go-openapi/jsonreference"
	"k8s.io/kube-openapi/pkg/validation/spec"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

/*
The GRPC code generation does not correctly support "inline". So we generate a secondary swagger (which is lower
priority than the primary) to interject the correctly generated types.

We do some hackerey here too:

* Change "/" into "." in names.
* Change "argo-workflows" into "argo_workflows".
*/
func secondarySwaggerGen() {
	definitions := make(map[string]interface{})
	for n, d := range wfv1.GetOpenAPIDefinitions(func(path string) spec.Ref {
		return spec.Ref{
			Ref: jsonreference.MustCreateRef("#/definitions/" + strings.ReplaceAll(path, "/", ".")),
		}
	}) {
		n = strings.ReplaceAll(n, "/", ".")
		println(n)
		definitions[n] = d.Schema
	}
	swagger := map[string]interface{}{
		"definitions": definitions,
	}
	f, err := os.Create("pkg/apiclient/_.secondary.swagger.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	err = e.Encode(swagger)
	if err != nil {
		panic(err)
	}

	read, err := os.ReadFile("pkg/apiclient/_.secondary.swagger.json")
	if err != nil {
		panic(err)
	}
	newContents := strings.ReplaceAll(string(read), "argoproj.argo-workflows", "argoproj.argo_workflows")
	err = os.WriteFile("pkg/apiclient/_.secondary.swagger.json", []byte(newContents), 0o600)
	if err != nil {
		panic(err)
	}
}
