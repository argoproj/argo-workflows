package main

import (
	"strings"
)

var kinds = []string{"ClusterWorkflowTemplate", "CronWorkflow", "Workflow", "WorkflowTemplate"}

type swagger obj

var intOrString = obj{"x-kubernetes-int-or-string": true}
var any = obj{"x-kubernetes-preserve-unknown-fields": true}

func (s swagger) definitionByName(name string) obj {
	switch name {
	case "io.argoproj.workflow.v1alpha1.Item":
		return any
	case "io.k8s.apimachinery.pkg.api.resource.Quantity":
		return intOrString
	case "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta":
		return obj{"type": "object"}
	}
	return s["definitions"].(obj)[name].(obj)
}

func (s swagger) expand(definition obj) obj {
	if ref, ok := definition["$ref"]; ok {
		delete(definition, "$ref")
		return s.expand(s.definitionByName(strings.TrimPrefix(ref.(string), "#/definitions/")))
	}
	if _, ok := definition["anyOf"]; ok {
		return any
	}
	s.scrub(definition)

	if items, ok := definition["items"].(obj); ok {
		if _, ok := items["anyOf"]; ok {
			definition["items"] = any
		} else if ref, ok := items["$ref"]; ok {
			delete(items, "$ref")
			definition["items"] = s.expand(s.definitionByName(strings.TrimPrefix(ref.(string), "#/definitions/")))
		}
	}
	if properties, ok := definition["properties"].(obj); ok {
		for name, value := range properties {
			if _, ok := value.(obj)["anyOf"]; ok {
				definition[name] = any
			} else {
				properties[name] = s.expand(value.(obj))
			}
		}
	}
	if properties, ok := definition["additionalProperties"].(obj); ok {
		definition["additionalProperties"] = s.expand(properties)
	}
	if format, ok := definition["format"]; ok && format == "int-or-string" {
		return intOrString
	}
	return definition
}

func (s swagger) scrub(definition obj) {
	delete(definition, "description")
	delete(definition, "x-kubernetes-group-version-kind")
	delete(definition, "x-kubernetes-patch-merge-key")
	delete(definition, "x-kubernetes-patch-strategy")
	properties, ok := definition["properties"].(obj)
	if ok {
		for _, v := range properties {
			s.scrub(v.(obj))
		}
	}
}
