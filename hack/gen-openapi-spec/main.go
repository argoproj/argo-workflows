package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-openapi/spec"
	"k8s.io/kube-openapi/pkg/common"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// Generate OpenAPI spec definitions for Workflow Resource
func main() {
	if len(os.Args) <= 1 {
		log.Fatal("Supply a version")
	}
	version := os.Args[1]
	oAPIDefs := wfv1.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.MustCreateRef("#/definitions/" + common.EscapeJsonPointer(swaggify(name)))
	})
	defs := spec.Definitions{}
	for defName, val := range oAPIDefs {
		defs[swaggify(defName)] = val.Schema
	}
	defs["io.k8s.apimachinery.pkg.runtime.Object"] = spec.Schema{
		SchemaProps: spec.SchemaProps{
			Title: "This is a hack do deal with this problem: https://github.com/kubernetes/kube-openapi/issues/174",
		},
	}
	swagger := spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger:     "2.0",
			Definitions: defs,
			Paths:       &spec.Paths{Paths: map[string]spec.PathItem{}},
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title:       "Argo Kube API",
					Description: "The Kubernetes based API for Argo",
					Version:     version,
				},
			},
		},
	}
	jsonBytes, err := json.MarshalIndent(swagger, "", "  ")
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(string(jsonBytes))
}

// swaggify converts the github package
// e.g.:
// github.com/argoproj/argo/pkg/apis/workflow/v1alpha1.Workflow
// to:
// io.argoproj.workflow.v1alpha1.Workflow
func swaggify(name string) string {
	name = strings.Replace(name, "github.com/argoproj/argo/pkg/apis", "argoproj.io", -1)
	parts := strings.Split(name, "/")
	hostParts := strings.Split(parts[0], ".")
	// reverses something like k8s.io to io.k8s
	for i, j := 0, len(hostParts)-1; i < j; i, j = i+1, j-1 {
		hostParts[i], hostParts[j] = hostParts[j], hostParts[i]
	}
	parts[0] = strings.Join(hostParts, ".")
	return strings.Join(parts, ".")
}
