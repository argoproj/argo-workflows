package main

import (
	"encoding/json"
	"io/ioutil"
	"reflect"

	log "github.com/sirupsen/logrus"
)

type obj = map[string]interface{}

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
			log.Infof("replacing bad definition %s", n)
			definitions[n] = kd
		}
	}
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
