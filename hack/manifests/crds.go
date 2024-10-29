package main

import (
	"fmt"
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

// minimizeCRD is a workaround for two separate issues:
// 1. "Request entity too large: limit is 3145728" errors due to https://github.com/kubernetes/kubernetes/issues/82292
// 2. "spec.validation.openAPIV3Schema.properties[spec].properties[tasks].additionalProperties.properties[steps].items.items: Required value: must be specified" due to kubebuilder issues (https://github.com/argoproj/argo-workflows/pull/3809#discussion_r472383090)
func minimizeCRD(filename string) {
	data, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		panic(err)
	}

	shouldMinimize := false
	if len(data) > 1024*1024 {
		fmt.Printf("Minimizing %s due to CRD size (%d) exceeding 1MB\n", filename, len(data))
		shouldMinimize = true
	}

	crd := make(obj)
	err = yaml.Unmarshal(data, &crd)
	if err != nil {
		panic(err)
	}

	if !shouldMinimize {
		name := crd["metadata"].(obj)["name"].(string)
		switch name {
		case "cronworkflows.argoproj.io", "clusterworkflowtemplates.argoproj.io", "workflows.argoproj.io", "workflowtemplates.argoproj.io", "workflowtasksets.argoproj.io":
			fmt.Printf("Minimizing %s due to kubebuilder issues\n", filename)
			shouldMinimize = true
		}
	}

	if !shouldMinimize {
		return
	}

	crd = stripSpecAndStatusFields(crd)

	data, err = yaml.Marshal(crd)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(filename, data, 0o600)
	if err != nil {
		panic(err)
	}
}

// stripSpecAndStatusFields strips the "spec" and "status" fields from the CRD, as those are usually the largest.
func stripSpecAndStatusFields(crd obj) obj {
	spec := crd["spec"].(obj)
	versions := spec["versions"].([]interface{})
	version := versions[0].(obj)
	properties := version["schema"].(obj)["openAPIV3Schema"].(obj)["properties"].(obj)
	for k := range properties {
		if k == "spec" || k == "status" {
			properties[k] = obj{"type": "object", "x-kubernetes-preserve-unknown-fields": true, "x-kubernetes-map-type": "atomic"}
		}
	}
	return crd
}
