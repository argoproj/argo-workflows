package main

import (
	"encoding/json"
	"os"
	"reflect"
)

func kubeifySwagger(in, out string) {
	f, err := os.Open(in)
	if err != nil {
		panic(err)
	}
	swagger := obj{}
	err = json.NewDecoder(f).Decode(&swagger)
	if err != nil {
		panic(err)
	}
	definitions := swagger["definitions"].(obj)
	definitions["io.k8s.apimachinery.pkg.apis.meta.v1.Fields"] = obj{}
	definitions["io.k8s.apimachinery.pkg.apis.meta.v1.Initializer"] = obj{}
	definitions["io.k8s.apimachinery.pkg.apis.meta.v1.Initializers"] = obj{}
	definitions["io.k8s.apimachinery.pkg.apis.meta.v1.Status"] = obj{}
	definitions["io.k8s.apimachinery.pkg.apis.meta.v1.StatusCause"] = obj{}
	definitions["io.k8s.apimachinery.pkg.apis.meta.v1.StatusDetails"] = obj{}
	delete(definitions, "io.k8s.apimachinery.pkg.apis.meta.v1.Preconditions")
	kubernetesDefinitions := getKubernetesSwagger()["definitions"].(obj)
	for n, d := range definitions {
		kd, ok := kubernetesDefinitions[n]
		if ok && !reflect.DeepEqual(d, kd) {
			println("replacing bad definition " + n)
			definitions[n] = kd
		}
	}

	//loop again to handle any new bad definitions
	for _, d := range definitions {
		if d.(obj)["properties"] != nil {
			props := d.(obj)["properties"].(obj)
			for _, content := range props {
				if content.(obj)["format"] == "int32" || content.(obj)["format"] == "int64" {
					delete(content.(obj), "format")
				}
			}
		}
	}

	definitions["io.k8s.apimachinery.pkg.util.intstr.IntOrString"] = obj{"type": array{"string", "integer"}}
	// "omitempty" does not work for non-nil structs, so we must change it here
	definitions["io.argoproj.workflow.v1alpha1.CronWorkflow"].(obj)["required"] = array{"metadata", "spec"}
	definitions["io.argoproj.workflow.v1alpha1.Workflow"].(obj)["required"] = array{"metadata", "spec"}
	definitions["io.argoproj.workflow.v1alpha1.ScriptTemplate"].(obj)["required"] = array{"image", "source"}
	definitions["io.k8s.api.core.v1.Container"].(obj)["required"] = array{"image"}

	f, err = os.Create(out)
	if err != nil {
		panic(err)
	}
	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	err = e.Encode(swagger)
	if err != nil {
		panic(err)
	}
}

func getKubernetesSwagger() obj {
	f, err := os.Open("dist/kubernetes.swagger.json")
	if err != nil {
		panic(err)
	}
	swagger := obj{}
	err = json.NewDecoder(f).Decode(&swagger)
	if err != nil {
		panic(err)
	}
	return swagger
}
