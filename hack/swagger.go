package main

import (
	"strings"
)

var kinds = []string{"ClusterWorkflowTemplate", "CronWorkflow", "Workflow", "WorkflowTemplate"}

type swagger obj

func (s swagger) definitionByName(name string) obj {
	switch name {
	case "io.k8s.apimachinery.pkg.api.resource.Quantity",
		"io.argoproj.workflow.v1alpha1.Item":
		return obj{
			"x-kubernetes-int-or-string": true,
		}
	}
	return s["definitions"].(obj)[name].(obj)
}

func (s swagger) expand(definition obj) obj {
	ref, ok := definition["$ref"]
	if ok {
		delete(definition, "$ref")
		return s.expand(s.definitionByName(strings.TrimPrefix(ref.(string), "#/definitions/")))
	}
	s.scrub(definition)
	delete(definition, "anyOf")
	items, ok := definition["items"].(obj)
	if ok {
		delete(items, "anyOf")
		ref, ok := items["$ref"]
		if ok {
			delete(items, "$ref")
			definition["items"] = s.expand(s.definitionByName(strings.TrimPrefix(ref.(string), "#/definitions/")))
		}
	}
	properties, ok := definition["properties"].(obj)
	if ok {
		for name, value := range properties {
			delete(properties, "anyOf")
			if name == "metadata" {
				properties[name] = obj{"type": "object"}
			} else {
				properties[name] = s.expand(value.(obj))
			}
		}
	}
	properties, ok = definition["additionalProperties"].(obj)
	if ok {
		definition["additionalProperties"] = s.expand(properties)
	}
	format, ok := definition["format"]
	if ok {
		if format == "int-or-string" {
			delete(definition, "format")
			delete(definition, "type")
			definition["anyOf"] = array{
				obj{"type": "integer"},
				obj{"type": "string"},
			}
			definition["x-kubernetes-int-or-string"] = true
		}
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
