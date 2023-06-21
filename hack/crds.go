package main

import (
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

func cleanCRD(filename string) {
	data, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		panic(err)
	}
	crd := make(obj)
	err = yaml.Unmarshal(data, &crd)
	if err != nil {
		panic(err)
	}
	delete(crd, "status")
	metadata := crd["metadata"].(obj)
	delete(metadata, "annotations")
	delete(metadata, "creationTimestamp")
	spec := crd["spec"].(obj)
	versions := spec["versions"].([]interface{})
	version := versions[0].(obj)
	schema := version["schema"].(obj)["openAPIV3Schema"].(obj)
	name := crd["metadata"].(obj)["name"].(string)
	switch name {
	case "cronworkflows.argoproj.io":
		properties := schema["properties"].(obj)["spec"].(obj)["properties"].(obj)["workflowSpec"].(obj)["properties"].(obj)["templates"].(obj)["items"].(obj)["properties"]
		properties.(obj)["container"].(obj)["required"] = []string{"image"}
		properties.(obj)["script"].(obj)["required"] = []string{"image", "source"}
	case "clusterworkflowtemplates.argoproj.io", "workflows.argoproj.io", "workflowtemplates.argoproj.io":
		properties := schema["properties"].(obj)["spec"].(obj)["properties"].(obj)["templates"].(obj)["items"].(obj)["properties"]
		properties.(obj)["container"].(obj)["required"] = []string{"image"}
		properties.(obj)["script"].(obj)["required"] = []string{"image", "source"}
	}
	data, err = yaml.Marshal(crd)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(filename, data, 0o600)
	if err != nil {
		panic(err)
	}
}

func removeCRDValidation(filename string) {
	data, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		panic(err)
	}
	crd := make(obj)
	err = yaml.Unmarshal(data, &crd)
	if err != nil {
		panic(err)
	}
	spec := crd["spec"].(obj)
	versions := spec["versions"].([]interface{})
	version := versions[0].(obj)
	properties := version["schema"].(obj)["openAPIV3Schema"].(obj)["properties"].(obj)
	for k := range properties {
		if k == "spec" || k == "status" {
			properties[k] = obj{"type": "object", "x-kubernetes-preserve-unknown-fields": true, "x-kubernetes-map-type": "atomic"}
		}
	}
	data, err = yaml.Marshal(crd)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(filename, data, 0o600)
	if err != nil {
		panic(err)
	}
}
