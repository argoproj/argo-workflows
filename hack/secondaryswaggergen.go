package main

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/go-openapi/jsonreference"
	spec "github.com/go-openapi/spec"
	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

/*
	The GRPC code generation does not correctly support "inline". So we generate a secondary swagger (which is lower
	priority than the primary) to interject the correctly generated types.

	We do some hackerey here too:

	* Change "/" into "." in names.
*/
func secondarySwaggerGen() {
	definitions := make(map[string]interface{})
	for n, d := range wfv1.GetOpenAPIDefinitions(func(path string) spec.Ref {
		return spec.Ref{
			Ref: jsonreference.MustCreateRef("#/definitions/" + strings.ReplaceAll(path, "/", ".")),
		}
	}) {
		n = strings.ReplaceAll(n, "/", ".")
		log.Println(n)
		definitions[n] = d.Schema
	}
	swagger := map[string]interface{}{
		"definitions": definitions,
	}
	data, err := json.MarshalIndent(swagger, "", "  ")
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("pkg/apiclient/_.secondary.swagger.json", data, 0644)
	if err != nil {
		panic(err)
	}
}
