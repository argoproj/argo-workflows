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
	kubernetesDefinitions := getKubernetesSwagger()["definitions"].(obj)
	for n, d1 := range definitions {
		d, ok := kubernetesDefinitions[n]
		if ok && !reflect.DeepEqual(d1, d) {
			log.Infof("replacing bad definition %s", n)
			definitions[n] = d
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
