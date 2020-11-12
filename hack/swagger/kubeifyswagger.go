package main

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
)

func kubeifySwagger(in, out string) {
	data, err := ioutil.ReadFile(in)
	if err != nil {
		panic(err)
	}
	swagger := obj{}
	err = json.Unmarshal(data, &swagger)
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
	// "omitempty" does not work for non-nil structs, so we must change it here
	definitions["io.argoproj.workflow.v1alpha1.CronWorkflow"].(obj)["required"] = array{"metadata", "spec"}
	definitions["io.argoproj.workflow.v1alpha1.Workflow"].(obj)["required"] = array{"metadata", "spec"}
	definitions["io.argoproj.workflow.v1alpha1.ScriptTemplate"].(obj)["required"] = array{"image", "source"}
	definitions["io.k8s.api.core.v1.Container"].(obj)["required"] = array{"image"}
	data, err = json.MarshalIndent(swagger, "", "  ")
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(out, data, 0644)
	if err != nil {
		panic(err)
	}
}

func getKubernetesSwagger() obj {
	data, err := ioutil.ReadFile("dist/kubernetes.swagger.json")
	if err != nil {
		panic(err)
	}
	swagger := obj{}
	err = json.Unmarshal(data, &swagger)
	if err != nil {
		panic(err)
	}
	return swagger
}
